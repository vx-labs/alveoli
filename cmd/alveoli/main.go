package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vx-labs/alveoli/alveoli/auth"
	"github.com/vx-labs/alveoli/alveoli/rpc"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/wasp/api"
)

type UpdateManifest struct {
	Active *bool `json:"active"`
}

func UpdateDevice(client vespiary.VespiaryClient, domain string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		manifest := UpdateManifest{}
		err := json.NewDecoder(r.Body).Decode(&manifest)
		if err != nil {
			log.Print(err)
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(err)
			return
		}
		if manifest.Active != nil {
			if *manifest.Active == false {
				_, err = client.DisableDevice(r.Context(), &vespiary.DisableDeviceRequest{Owner: authContext.Tenant, ID: ps.ByName("device_id")})
			} else {
				_, err = client.EnableDevice(r.Context(), &vespiary.EnableDeviceRequest{Owner: authContext.Tenant, ID: ps.ByName("device_id")})
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

func fillWithSubscriptions(tenant string, subscriptions []*wasp.CreateSubscriptionRequest, sessions []*wasp.SessionMetadatas, client *Device) {

	for idx := range subscriptions {
		sessionID := ""
		subscription := subscriptions[idx]

		for sessionIdx := range sessions {
			session := sessions[sessionIdx]
			if session.ClientID == client.Name {
				sessionID = session.SessionID
			}
		}
		if sessionID != "" {
			if bytes.HasPrefix(subscription.Pattern, []byte(tenant)) && subscription.SessionID == sessionID {
				client.SubscriptionCount++
			}
		}
	}
}

func ListDevices(client vespiary.VespiaryClient, waspClient wasp.MQTTClient, domain string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		authDevices, err := client.ListDevices(r.Context(), &vespiary.ListDevicesRequest{Owner: authContext.Tenant})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch device list"}`))
			return
		}
		sessions, err := waspClient.ListSessionMetadatas(r.Context(), &wasp.ListSessionMetadatasRequest{})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch connected session list"}`))
			return
		}
		subscriptions, err := waspClient.ListSubscriptions(r.Context(), &wasp.ListSubscriptionsRequest{})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch subscription list"}`))
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
			fillWithMetadata(authContext.Tenant, sessions.SessionMetadatasList, &out[idx])
			fillWithSubscriptions(authContext.Tenant, subscriptions.Subscriptions, sessions.SessionMetadatasList, &out[idx])
			if out[idx].Active {
				if out[idx].Connected {
					out[idx].HumanStatus = "online"
				} else {
					out[idx].HumanStatus = "offline"
				}
			} else {
				out[idx].HumanStatus = "disabled"
			}
		}
		json.NewEncoder(w).Encode(out)
	}
}
func GetDevice(client vespiary.VespiaryClient, waspClient wasp.MQTTClient, domain string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())

		device, err := client.GetDevice(r.Context(), &vespiary.GetDeviceRequest{Owner: authContext.Tenant, ID: ps.ByName("device_id")})
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
		authContext := auth.Informations(r.Context())

		_, err := client.DeleteDevice(r.Context(), &vespiary.DeleteDeviceRequest{Owner: authContext.Tenant, ID: ps.ByName("device_id")})
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

			var authProvider auth.Provider
			switch config.GetString("authentication-provider") {
			case "static":
				authProvider = auth.Static(config.GetString("authentication-provider-static-tenant"))
			case "auth0":
				authProvider = auth.Auth0(config.GetString("auth0-client-domain"), config.GetString("auth0-api-id"))
			default:
				panic("unknown authentication provider specified")
			}

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
	cmd.AddCommand(TLSHelper(config))

	cmd.Execute()
}
