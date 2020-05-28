package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vx-labs/alveoli/alveoli/rpc"
	"github.com/vx-labs/vespiary/vespiary/api"
)

func ListDevices(client api.VespiaryClient) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		claim := r.Context().Value("user").(*jwt.Token).Claims.(jwt.MapClaims)
		user := claim["sub"].(string)
		devices, err := client.ListDevices(r.Context(), &api.ListDevicesRequest{Owner: user})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
		if devices.Devices != nil {
			json.NewEncoder(w).Encode(devices.Devices)
		} else {
			json.NewEncoder(w).Encode([]struct{}{})
		}
	}
}
func GetDevice(client api.VespiaryClient) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		device, err := client.GetDevice(r.Context(), &api.GetDeviceRequest{Owner: "vx:psk", ID: ps.ByName("device_id")})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
		json.NewEncoder(w).Encode(device.Device)
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

			router.GET("/devices/", ListDevices(api.NewVespiaryClient(authConn)))
			router.GET("/devices/:device_id", GetDevice(api.NewVespiaryClient(authConn)))

			log.Fatal(http.ListenAndServe(":8080", jwtMiddleware.Handler(router)))
		},
	}
	cmd.Flags().Bool("insecure", false, "Disable GRPC client-side TLS validation.")

	cmd.Flags().String("rpc-tls-certificate-authority-file", "", "x509 certificate authority used by RPC Server.")
	cmd.Flags().String("rpc-tls-certificate-file", "", "x509 certificate used by RPC Server.")
	cmd.Flags().String("rpc-tls-private-key-file", "", "Private key used by RPC Server.")

	cmd.Flags().String("auth0-client-domain", "", "Auth0 client domain.")
	cmd.Flags().String("auth0-api-id", "", "Auth0 API ID.")

	cmd.AddCommand(TLSHelper(config))

	cmd.Execute()
}
