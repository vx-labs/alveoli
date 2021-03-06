package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vx-labs/alveoli/alveoli/auth"
	"github.com/vx-labs/alveoli/alveoli/graph/generated"
	"github.com/vx-labs/alveoli/alveoli/graph/resolvers"
	"github.com/vx-labs/alveoli/alveoli/handlers"
	"github.com/vx-labs/alveoli/alveoli/rpc"
	nest "github.com/vx-labs/nest/nest/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/v4/wasp/api"
	"github.com/vx-labs/wasp/vaultacme"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	logConfig := zap.NewProductionConfig()
	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}

	config := viper.New()
	config.SetEnvPrefix("alveoli")
	config.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	config.AutomaticEnv()
	cmd := cobra.Command{
		Use: "alveoli",
		PreRun: func(cmd *cobra.Command, _ []string) {
			config.BindPFlags(cmd.Flags())
		},
		Run: func(cmd *cobra.Command, _ []string) {

			if nrLicenseKey := config.GetString("newrelic-license-key"); nrLicenseKey != "" {
				newrelic.NewApplication(
					newrelic.ConfigAppName("alveoli"),
					newrelic.ConfigLicense(nrLicenseKey),
					newrelic.ConfigDistributedTracerEnabled(true),
				)
			}
			rpcDialer := rpc.GRPCDialer(rpc.ClientConfig{
				InsecureSkipVerify:          config.GetBool("insecure"),
				TLSCertificatePath:          config.GetString("rpc-tls-certificate-file"),
				TLSPrivateKeyPath:           config.GetString("rpc-tls-private-key-file"),
				TLSCertificateAuthorityPath: config.GetString("rpc-tls-certificate-authority-file"),
			})

			authConn, err := rpcDialer(config.GetString("vespiary-grpc-address"))
			if err != nil {
				panic(err)
			}
			brokerConn, err := rpcDialer(config.GetString("wasp-grpc-address"))
			if err != nil {
				panic(err)
			}
			nestConn, err := rpcDialer(config.GetString("nest-grpc-address"))
			if err != nil {
				panic(err)
			}

			vespiaryClient := vespiary.NewVespiaryClient(authConn)
			waspClient := wasp.NewMQTTClient(brokerConn)
			nestClient := nest.NewMessagesClient(nestConn)

			var authProvider auth.Provider
			switch config.GetString("authentication-provider") {
			case "static":
				authProvider = auth.Static(config.GetString("authentication-provider-static-account-id"), config.GetString("authentication-provider-static-tenant"))
			case "auth0":
				authProvider = auth.Auth0(config.GetString("auth0-client-domain"), config.GetString("auth0-api-id"), vespiaryClient)
			default:
				panic("unknown authentication provider specified")
			}
			var mqttClient mqtt.Client

			if config.GetString("rpc-tls-private-key-file") != "" && config.GetString("rpc-tls-certificate-file") != "" {
				mqttBrokerURL, err := url.Parse(fmt.Sprintf("tls://%s:8883", config.GetString("subscriptions-mqtt-broker")))
				if err != nil {
					panic("invalid broker url")
				}
				cert, err := tls.LoadX509KeyPair(config.GetString("rpc-tls-certificate-file"), config.GetString("rpc-tls-private-key-file"))
				if err != nil {
					log.Panicf("failed to load tls credentials: %v", err)
				}
				pool, err := x509.SystemCertPool()
				if err != nil {
					panic("failed to load tls system cert pool")
				}
				mqttClient = mqtt.NewClient(&mqtt.ClientOptions{
					Servers:        []*url.URL{mqttBrokerURL},
					AutoReconnect:  true,
					ClientID:       fmt.Sprintf("alveoli-%s", uuid.New().String()),
					CleanSession:   true,
					KeepAlive:      30,
					PingTimeout:    20 * time.Second,
					ConnectTimeout: 15 * time.Second,
					TLSConfig: &tls.Config{
						ServerName:   config.GetString("subscriptions-mqtt-broker-sni"),
						Certificates: []tls.Certificate{cert},
						RootCAs:      pool,
					},
					OnConnect: func(c mqtt.Client) {
						log.Printf("connected to mqtt broker: %s", mqttBrokerURL.String())
					},
					OnConnectionLost: func(c mqtt.Client, err error) {
						log.Printf("connection lost to mqtt broker %s: %v", mqttBrokerURL.String(), err)
					},
				})
				log.Printf("connecting to mqtt broker: %s", mqttBrokerURL.String())
				if token := mqttClient.Connect(); token.Wait() {
					if err := token.Error(); err != nil {
						log.Panicf("failed to connect to mqtt broker: %v", err)
					}
				}
			}
			srv := handler.New(
				generated.NewExecutableSchema(
					generated.Config{
						Resolvers: resolvers.Root(
							mqttClient,
							waspClient,
							vespiaryClient,
							nestClient,
						),
					},
				),
			)

			srv.AddTransport(transport.Websocket{
				KeepAlivePingInterval: 10 * time.Second,
				InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, error) {
					md, err := authProvider.Validate(ctx, initPayload.Authorization())
					if err != nil {
						log.Printf("websocket auth failed: %v", err)
						return nil, err
					}
					log.Printf("websocket session started for account %s", md.AccountID)
					return auth.StoreInformations(ctx, md), nil
				},
				Upgrader: websocket.Upgrader{
					CheckOrigin: func(r *http.Request) bool {
						return true
					},
				},
			})
			srv.AddTransport(transport.Options{})
			srv.AddTransport(transport.GET{})
			srv.AddTransport(transport.POST{})
			srv.AddTransport(transport.MultipartForm{})

			srv.SetQueryCache(lru.New(1000))

			srv.Use(extension.Introspection{})
			srv.Use(extension.AutomaticPersistedQuery{
				Cache: lru.New(100),
			})

			mux := http.NewServeMux()
			router := httprouter.New()
			handlers.Register(router, authProvider, vespiaryClient)
			mux.Handle("/account/", router)
			mux.Handle("/graphql", auth.Handler(authProvider, srv))
			mux.Handle("/", playground.Handler("GraphQL playground", "/graphql"))

			corsHandler := cors.New(cors.Options{
				AllowedMethods: []string{
					http.MethodGet,
					http.MethodPatch,
					http.MethodPost,
					http.MethodDelete,
				},
				AllowedHeaders: []string{
					"authorization",
					"content-type",
					"x-vx-product",
				},
				AllowCredentials: true,
			})
			listenAddr := fmt.Sprintf(":%d", config.GetInt("port"))

			if config.GetBool("use-vault") {
				tlsConfig, err := vaultacme.GetConfig(ctx, config.GetString("tls-cn"), logger)
				if err != nil {
					logger.Fatal("failed to get TLS certificate from ACME", zap.Error(err))
				}
				listener, err := tls.Listen("tcp", listenAddr, tlsConfig)
				if err != nil {
					logger.Fatal("failed to listen", zap.Error(err))
				}
				log.Fatal(http.Serve(listener, corsHandler.Handler(&Logger{handler: mux})))
			} else {
				listener, err := net.Listen("tcp", listenAddr)
				if err != nil {
					logger.Fatal("failed to listen tcp", zap.Error(err))
				}
				log.Fatal(http.Serve(listener, corsHandler.Handler(&Logger{handler: mux})))
			}
		},
	}
	cmd.Flags().Bool("insecure", false, "Disable GRPC client-side TLS validation.")

	cmd.Flags().String("rpc-tls-certificate-authority-file", "", "x509 certificate authority used by RPC Server.")
	cmd.Flags().String("rpc-tls-certificate-file", "", "x509 certificate used by RPC Server.")
	cmd.Flags().String("rpc-tls-private-key-file", "", "Private key used by RPC Server.")

	cmd.Flags().String("subscriptions-mqtt-broker", "broker.iot.cloud.vx-labs.net", "MQTT Broker to connect.")
	cmd.Flags().String("subscriptions-mqtt-broker-sni", "broker.iot.cloud.vx-labs.net", "MQTT Broker server name to use in TLS handshale.")

	cmd.Flags().String("auth0-client-domain", "", "Auth0 client domain.")
	cmd.Flags().String("auth0-api-id", "", "Auth0 API ID.")
	cmd.Flags().Int("port", 8080, "Run REST API on this port.")
	cmd.Flags().String("authentication-provider", "auth0", "How shall we authenticate user requests? Supported values are auth0 and static.")
	cmd.Flags().String("authentication-provider-static-tenant", "vx:psk", "The default tenant to use when using static authentication provider.")
	cmd.Flags().String("authentication-provider-static-account-id", "1", "The account-id to use when using static authentication provider.")
	cmd.Flags().Bool("use-vault", false, "Use Hashicorp Vault to store private keys and certificates.")
	cmd.Flags().String("tls-cn", "localhost", "Get ACME certificat for this Common Name.")

	cmd.Flags().String("vespiary-grpc-address", "auth.iot.cloud.vx-labs.net:443", "auth service endpoint")
	cmd.Flags().String("nest-grpc-address", "messages.iot.cloud.vx-labs.net:443", "auth service endpoint")
	cmd.Flags().String("wasp-grpc-address", "rpc.iot.cloud.vx-labs.net:443", "auth service endpoint")

	cmd.AddCommand(TLSHelper(config))

	cmd.Execute()
}
