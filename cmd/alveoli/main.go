package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vx-labs/alveoli/alveoli/rpc"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/wasp/api"
)

type Logger struct {
	handler http.Handler
}

func (l *Logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	}()
	l.handler.ServeHTTP(w, r)
}

func getTenant(domain string, r *http.Request) (string, error) {
	email, err := userEmail(domain, r.Header.Get("Authorization"))
	if err != nil {
		log.Print(err)
		return "", nil
	}
	// Temp hack to avoid test devices migration
	if email == "julien@bonachera.fr" {
		email = "vx:psk"
	}
	return email, nil
}

type UpdateManifest struct {
	Active *bool `json:"active"`
}

func UpdateDevice(client vespiary.VespiaryClient, domain string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		tenant, err := getTenant(domain, r)
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
		manifest := UpdateManifest{}
		err = json.NewDecoder(r.Body).Decode(&manifest)
		if err != nil {
			log.Print(err)
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(err)
			return
		}
		if manifest.Active != nil {
			if *manifest.Active == false {
				_, err = client.DisableDevice(r.Context(), &vespiary.DisableDeviceRequest{Owner: tenant, ID: ps.ByName("device_id")})
			} else {
				_, err = client.EnableDevice(r.Context(), &vespiary.EnableDeviceRequest{Owner: tenant, ID: ps.ByName("device_id")})
			}
			if err != nil {
				log.Print(err)
				w.WriteHeader(500)
				return
			}
		}
	}
}

func fillWithMetadata(tenant string, sessions []*wasp.SessionMetadatas, client *Device) {
	for idx := range sessions {
		session := sessions[idx]
		if session.MountPoint == tenant && session.ClientID == client.Name {
			client.Connected = true
			return
		}
	}
}

func fillWithSubscriptions(tenant string, subscriptions []*wasp.CreateSubscriptionRequest, client *Device) {
	for idx := range subscriptions {
		subscription := subscriptions[idx]
		if bytes.HasPrefix(subscription.Pattern, []byte(tenant)) && subscription.SessionID == client.ID {
			client.SubscriptionCount++
			return
		}
	}
}

func ListDevices(client vespiary.VespiaryClient, waspClient wasp.MQTTClient, domain string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		tenant, err := getTenant(domain, r)
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}

		authDevices, err := client.ListDevices(r.Context(), &vespiary.ListDevicesRequest{Owner: tenant})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
		sessions, err := waspClient.ListSessionMetadatas(r.Context(), &wasp.ListSessionMetadatasRequest{})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
		subscriptions, err := waspClient.ListSubscriptions(r.Context(), &wasp.ListSubscriptionsRequest{})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}

		out := make([]Device, len(authDevices.Devices))
		for idx := range out {
			out[idx] = Device{
				ID:        authDevices.Devices[idx].ID,
				Name:      authDevices.Devices[idx].Name,
				Active:    authDevices.Devices[idx].Active,
				CreatedAt: authDevices.Devices[idx].CreatedAt,
				Password:  authDevices.Devices[idx].Password,

				Connected:         false,
				SentBytes:         0,
				ReceivedBytes:     0,
				SubscriptionCount: 0,
			}
			fillWithMetadata(tenant, sessions.SessionMetadatasList, &out[idx])
			fillWithSubscriptions(tenant, subscriptions.Subscriptions, &out[idx])
		}
		json.NewEncoder(w).Encode(out)
	}
}
func GetDevice(client vespiary.VespiaryClient, waspClient wasp.MQTTClient, domain string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		tenant, err := getTenant(domain, r)
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}

		device, err := client.GetDevice(r.Context(), &vespiary.GetDeviceRequest{Owner: tenant, ID: ps.ByName("device_id")})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
		json.NewEncoder(w).Encode(device.Device)
	}
}

func DeleteDevice(client vespiary.VespiaryClient, domain string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		tenant, err := getTenant(domain, r)
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}

		_, err = client.DeleteDevice(r.Context(), &vespiary.DeleteDeviceRequest{Owner: tenant, ID: ps.ByName("device_id")})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
	}
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

			jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
				ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
					// Verify 'aud' claim
					aud := config.GetString("auth0-api-id")
					checkAud := token.Claims.(jwt.MapClaims).VerifyAudience(aud, false)
					if !checkAud {
						return token, errors.New("Invalid audience")
					}
					// Verify 'iss' claim
					iss := fmt.Sprintf("https://%s/", config.GetString("auth0-client-domain"))
					checkIss := token.Claims.(jwt.MapClaims).VerifyIssuer(iss, false)
					if !checkIss {
						return token, errors.New("Invalid issuer")
					}

					cert, err := getPemCert(iss, token)
					if err != nil {
						panic(err.Error())
					}

					result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
					return result, nil
				},
				SigningMethod: jwt.SigningMethodRS256,
			})

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

			domain := fmt.Sprintf("https://%s/", config.GetString("auth0-client-domain"))
			authClient := vespiary.NewVespiaryClient(authConn)
			waspClient := wasp.NewMQTTClient(brokerConn)
			router.GET("/devices/", ListDevices(authClient, waspClient, domain))
			router.GET("/devices/:device_id", GetDevice(authClient, waspClient, domain))
			router.PATCH("/devices/:device_id", UpdateDevice(authClient, domain))
			router.DELETE("/devices/:device_id", DeleteDevice(authClient, domain))

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
			log.Fatal(http.ListenAndServe(port, corsHandler.Handler(&Logger{handler: jwtMiddleware.Handler(router)})))
		},
	}
	cmd.Flags().Bool("insecure", false, "Disable GRPC client-side TLS validation.")

	cmd.Flags().String("rpc-tls-certificate-authority-file", "", "x509 certificate authority used by RPC Server.")
	cmd.Flags().String("rpc-tls-certificate-file", "", "x509 certificate used by RPC Server.")
	cmd.Flags().String("rpc-tls-private-key-file", "", "Private key used by RPC Server.")

	cmd.Flags().String("auth0-client-domain", "", "Auth0 client domain.")
	cmd.Flags().String("auth0-api-id", "", "Auth0 API ID.")
	cmd.Flags().Int("port", 8080, "Run REST API on this port.")

	cmd.AddCommand(TLSHelper(config))

	cmd.Execute()
}
