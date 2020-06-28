package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	nest "github.com/vx-labs/nest/nest/api"

	"github.com/julienschmidt/httprouter"
	"github.com/vx-labs/alveoli/alveoli/auth"
)

type events struct {
	nest nest.EventsClient
}

type GetEventsRequest struct {
	Since string `json:"since,omitempty"`
}

func registerEvents(router *httprouter.Router, nestClient nest.EventsClient) {
	topics := &events{nest: nestClient}
	router.POST("/events/", topics.Get())
}

type Event struct {
	Timestamp  int64             `json:"timestamp,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Service    string            `json:"service,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

func mapEvent(mountpoint string, t *nest.Event) Event {
	attributes := make(map[string]string, len(t.Attributes))
	for _, attribute := range t.Attributes {
		attributes[attribute.Key] = attribute.Value
	}
	out := Event{
		Timestamp:  t.Timestamp,
		Kind:       t.Kind,
		Service:    t.Service,
		Attributes: attributes,
	}
	return out
}

func (d *events) Get() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())

		body := GetEventsRequest{}

		fromTimestamp, err := parseSince(body.Since)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status_code": 400, "message": "malformed Since request parameter"`))
			return
		}

		err = json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status_code": 400, "message": "malformed JSON"`))
			return
		}
		stream, err := d.nest.GetEvents(r.Context(), &nest.GetEventRequest{
			FromTimestamp: fromTimestamp,
			Tenant:        authContext.Tenant,
		})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch topic messages"}`))
			return
		}
		encoder := json.NewEncoder(w)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[`))
		count := 0
		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				return
			}
			for idx := range msg.Events {
				if (count) > 0 {
					w.Write([]byte(`,`))
				}
				encoder.Encode(mapEvent(authContext.Tenant, msg.Events[idx]))
				count++
			}
		}
		w.Write([]byte(`]`))
	}
}
