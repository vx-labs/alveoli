package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/julienschmidt/httprouter"
	"github.com/vx-labs/alveoli/alveoli/auth"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
)

func registerAccounts(router *httprouter.Router, vespiaryClient vespiary.VespiaryClient, authProvider auth.Provider) {
	accountHandler := &accounts{
		vespiary:     vespiaryClient,
		authProvider: authProvider,
	}
	router.GET("/account/info", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		auth.Handler(authProvider, accountHandler.Informations()).ServeHTTP(w, r)
	})
	router.POST("/account/", accountHandler.Create())
}

type AccountInformations struct {
	ID        string   `json:"id,omitempty"`
	Usernames []string `json:"usernames,omitempty"`
}

type accounts struct {
	vespiary     vespiary.VespiaryClient
	authProvider auth.Provider
}

func (d *accounts) Informations() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authContext := auth.Informations(r.Context())
		d.vespiary.GetAccountByPrincipal(r.Context(), &vespiary.GetAccountByPrincipalRequest{
			Principal: authContext.Principal,
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AccountInformations{
			ID:        authContext.AccountID,
			Usernames: authContext.DeviceUsernames,
		})
	})
}
func (d *accounts) Create() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		token, err := jwtmiddleware.FromAuthHeader(r)
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status_code": 401, "message": "missing or invalid credentials","reason": "token is empty"}`))
			return
		}
		tenant, err := d.authProvider.Authenticate(r.Context(), token)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"status_code": 403, "message": "authentication failed","reason": "%s"}`, err.Error())
			return
		}
		_, err = d.vespiary.GetAccountByPrincipal(r.Context(), &vespiary.GetAccountByPrincipalRequest{
			Principal: tenant,
		})
		if err == nil {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"status_code": 409, "message": "account already created"}`))
			return
		}
		userEmail, err := d.authProvider.ResolveUserEmail(r.Header.Get("Authorization"))
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to resolve user profile"}`))
			return
		}
		out, err := d.vespiary.CreateAccount(r.Context(), &vespiary.CreateAccountRequest{
			Name:            userEmail,
			DeviceUsernames: []string{userEmail},
			Principals:      []string{tenant},
		})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to create account"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AccountInformations{
			ID:        out.ID,
			Usernames: []string{userEmail},
		})
	}
}
