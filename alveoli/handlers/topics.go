package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	nest "github.com/vx-labs/nest/nest/api"

	"github.com/julienschmidt/httprouter"
	"github.com/vx-labs/alveoli/alveoli/auth"
)

type topics struct {
	nest nest.MessagesClient
}

type GetTopicsRequest struct {
	Pattern string `json:"pattern,omitempty"`
}

func registerTopics(router *httprouter.Router, nestClient nest.MessagesClient) {
	topics := &topics{nest: nestClient}
	router.GET("/topics/", topics.List())
	router.POST("/topics/", topics.Get())
}

type topic struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	MessageCount uint64 `json:"messageCount,omitempty"`
}

type record struct {
	Topic     string `json:"topic,omitempty"`
	Payload   string `json:"payload,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

func mapMessage(mountpoint string, t *nest.Record) record {
	out := record{
		Timestamp: t.Timestamp,
		Topic:     strings.TrimPrefix(string(t.Topic), mountpoint+"/"),
		Payload:   string(t.Payload),
	}
	return out
}
func mapTopic(mountpoint string, t *nest.TopicMetadata) topic {
	name := strings.TrimPrefix(string(t.Name), mountpoint+"/")
	out := topic{
		ID:           base64.StdEncoding.EncodeToString([]byte(name)),
		Name:         name,
		MessageCount: t.MessageCount,
	}
	return out
}
func (d *topics) List() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		response, err := d.nest.ListTopics(r.Context(), &nest.ListTopicsRequest{
			Pattern: []byte(fmt.Sprintf("%s/#", authContext.Tenant)),
		})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch topic list"}`))
			return
		}

		out := make([]topic, len(response.TopicMetadatas))
		for idx := range out {
			out[idx] = mapTopic(authContext.Tenant, response.TopicMetadatas[idx])
		}
		json.NewEncoder(w).Encode(out)
	}
}
func (d *topics) Get() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())

		body := GetTopicsRequest{}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status_code": 400, "message": "malformed JSON"`))
			return
		}
		if len(body.Pattern) == 0 {
			body.Pattern = "#"
		}
		stream, err := d.nest.GetTopics(r.Context(), &nest.GetTopicsRequest{
			Pattern: []byte(fmt.Sprintf("%s/%s", authContext.Tenant, body.Pattern)),
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
			for idx := range msg.Records {
				if (count) > 0 {
					w.Write([]byte(`,`))
				}
				encoder.Encode(mapMessage(authContext.Tenant, msg.Records[idx]))
				count++
			}
		}
		w.Write([]byte(`]`))
	}
}
