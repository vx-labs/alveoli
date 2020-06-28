package handlers

import (
	"github.com/julienschmidt/httprouter"
	nest "github.com/vx-labs/nest/nest/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/wasp/api"
)

// Register install resource handlers on the provided router
func Register(router *httprouter.Router, vespiaryClient vespiary.VespiaryClient, nestClient nest.MessagesClient, eventsClient nest.EventsClient, waspClient wasp.MQTTClient) {
	registerDevices(router, vespiaryClient, waspClient)
	registerTopics(router, nestClient)
	registerEvents(router, eventsClient)
}
