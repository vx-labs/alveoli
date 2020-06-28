package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/vx-labs/alveoli/alveoli/auth"
)

func registerAccounts(router *httprouter.Router) {
	accountHandler := &accounts{}
	router.GET("/account/info", accountHandler.Informations())
}

type AccountInformations struct {
	Username string `json:"username,omitempty"`
}

type accounts struct{}

func (d *accounts) Informations() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AccountInformations{
			Username: authContext.Tenant,
		})
	}
}
