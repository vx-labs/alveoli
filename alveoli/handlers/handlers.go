package handlers

import (
	"github.com/julienschmidt/httprouter"
	"github.com/vx-labs/alveoli/alveoli/auth"
	nest "github.com/vx-labs/nest/nest/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/v4/wasp/api"
)

// Register install resource handlers on the provided router
func Register(router *httprouter.Router, authProvider auth.Provider, vespiaryClient vespiary.VespiaryClient, nestClient nest.MessagesClient, eventsClient nest.EventsClient, waspClient wasp.MQTTClient) {
	registerAccounts(router, vespiaryClient, authProvider)
	registerDevices(router, vespiaryClient, waspClient)
	registerTopics(router, nestClient)
	registerEvents(router, eventsClient)
}
