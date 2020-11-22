package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vx-labs/alveoli/alveoli"
	"github.com/vx-labs/alveoli/alveoli/auth"
	"github.com/vx-labs/alveoli/alveoli/handlers"
	"github.com/vx-labs/alveoli/alveoli/rpc"
	nest "github.com/vx-labs/nest/nest/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/v4/wasp/api"
)

type postData struct {
	Query     string                 `json:"query"`
	Operation string                 `json:"operation"`
	Variables map[string]interface{} `json:"variables"`
}

func main() {
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
			router := httprouter.New()

			rpcDialer := rpc.GRPCDialer(rpc.ClientConfig{
				InsecureSkipVerify:          config.GetBool("insecure"),
				TLSCertificatePath:          config.GetString("rpc-tls-certificate-file"),
				TLSPrivateKeyPath:           config.GetString("rpc-tls-private-key-file"),
				TLSCertificateAuthorityPath: config.GetString("rpc-tls-certificate-authority-file"),
			})

			authConn, err := rpcDialer("auth.iot.cloud.vx-labs.net:443")
			if err != nil {
				panic(err)
			}
			brokerConn, err := rpcDialer("rpc.iot.cloud.vx-labs.net:443")
			if err != nil {
				panic(err)
			}
			nestConn, err := rpcDialer("messages.iot.cloud.vx-labs.net:443")
			if err != nil {
				panic(err)
			}

			vespiaryClient := vespiary.NewVespiaryClient(authConn)
			waspClient := wasp.NewMQTTClient(brokerConn)
			nestClient := nest.NewMessagesClient(nestConn)
			eventsClient := nest.NewEventsClient(nestConn)

			graphQLSchema := alveoli.Schema(vespiaryClient, waspClient)

			var authProvider auth.Provider
			switch config.GetString("authentication-provider") {
			case "static":
				authProvider = auth.Static(config.GetString("authentication-provider-static-account-id"), config.GetString("authentication-provider-static-tenant"))
			case "auth0":
				authProvider = auth.Auth0(config.GetString("auth0-client-domain"), config.GetString("auth0-api-id"), vespiaryClient)
			default:
				panic("unknown authentication provider specified")
			}

			handlers.Register(router, authProvider, vespiaryClient, nestClient, eventsClient, waspClient)

			router.POST("/graphql", func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
				var p postData
				if err := json.NewDecoder(req.Body).Decode(&p); err != nil {
					w.WriteHeader(400)
					return
				}
				result := graphql.Do(graphql.Params{
					Context:        req.Context(),
					Schema:         graphQLSchema,
					RequestString:  p.Query,
					VariableValues: p.Variables,
					OperationName:  p.Operation,
				})
				w.Header().Add("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(result); err != nil {
					fmt.Printf("could not write result to response: %s", err)
				}
			})
			router.GET("/graphql", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
				fd, err := os.Open("./static/index.html")
				defer fd.Close() //Close after function return
				if err != nil {
					http.Error(w, "File not found.", 404)
					return
				}
				io.Copy(w, fd)
			})

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
			port := fmt.Sprintf(":%d", config.GetInt("port"))
			log.Fatal(http.ListenAndServe(port, corsHandler.Handler(&Logger{handler: authProvider.Handler(router)})))
		},
	}
	cmd.Flags().Bool("insecure", false, "Disable GRPC client-side TLS validation.")

	cmd.Flags().String("rpc-tls-certificate-authority-file", "", "x509 certificate authority used by RPC Server.")
	cmd.Flags().String("rpc-tls-certificate-file", "", "x509 certificate used by RPC Server.")
	cmd.Flags().String("rpc-tls-private-key-file", "", "Private key used by RPC Server.")

	cmd.Flags().String("auth0-client-domain", "", "Auth0 client domain.")
	cmd.Flags().String("auth0-api-id", "", "Auth0 API ID.")
	cmd.Flags().Int("port", 8080, "Run REST API on this port.")
	cmd.Flags().String("authentication-provider", "auth0", "How shall we authenticate user requests? Supported values are auth0 and static.")
	cmd.Flags().String("authentication-provider-static-tenant", "vx:psk", "The default tenant to use when using static authentication provider.")
	cmd.Flags().String("authentication-provider-static-account-id", "1", "The account-id to use when using static authentication provider.")
	cmd.AddCommand(TLSHelper(config))

	cmd.Execute()
}
