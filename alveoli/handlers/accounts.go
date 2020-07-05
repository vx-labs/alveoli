package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vx-labs/alveoli/alveoli/auth"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
)

func registerAccounts(router *httprouter.Router, vespiaryClient vespiary.VespiaryClient, authProvider auth.Provider) {
	accountHandler := &accounts{
		vespiary:     vespiaryClient,
		authProvider: authProvider,
	}
	router.GET("/account/info", auth.RequireAccountCreated(accountHandler.Informations()))
}

type AccountInformations struct {
	ID        string   `json:"id,omitempty"`
	Usernames []string `json:"usernames,omitempty"`
}

type accounts struct {
	vespiary     vespiary.VespiaryClient
	authProvider auth.Provider
}

func (d *accounts) Informations() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		d.vespiary.GetAccountByPrincipal(r.Context(), &vespiary.GetAccountByPrincipalRequest{
			Principal: authContext.Principal,
		})
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AccountInformations{
			ID:        authContext.AccountID,
			Usernames: authContext.DeviceUsernames,
		})
	}
}
func (d *accounts) Create() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		if authContext.AccountID != "" {
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
			Principals:      []string{authContext.Principal},
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
