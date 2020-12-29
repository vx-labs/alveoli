package handlers

import (
	"github.com/julienschmidt/httprouter"
	"github.com/vx-labs/alveoli/alveoli/auth"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
)

// Register install resource handlers on the provided router
func Register(router *httprouter.Router, authProvider auth.Provider, vespiaryClient vespiary.VespiaryClient) {
	registerAccounts(router, vespiaryClient, authProvider)
}
