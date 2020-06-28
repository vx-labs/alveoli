package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/wasp/api"

	"github.com/julienschmidt/httprouter"
	"github.com/vx-labs/alveoli/alveoli/auth"
)

type devices struct {
	vespiary vespiary.VespiaryClient
	wasp     wasp.MQTTClient
}

func registerDevices(router *httprouter.Router, vespiaryClient vespiary.VespiaryClient, waspClient wasp.MQTTClient) {
	devices := &devices{vespiary: vespiaryClient, wasp: waspClient}
	router.POST("/devices/", devices.Create())
	router.GET("/devices/", devices.List())
	router.GET("/devices/:device_id", devices.Get())
	router.PATCH("/devices/:device_id", devices.Update())
	router.DELETE("/devices/:device_id", devices.Delete())
}

type device struct {
	Active            bool   `json:"active"`
	Connected         bool   `json:"connected"`
	CreatedAt         int64  `json:"createdAt"`
	ID                string `json:"id"`
	Name              string `json:"name"`
	Password          string `json:"password"`
	ReceivedBytes     int64  `json:"receivedBytes"`
	SentBytes         int64  `json:"sentBytes"`
	SubscriptionCount int    `json:"subscriptionCount"`
	HumanStatus       string `json:"humanStatus"`
}

type updateDeviceRequest struct {
	Active *bool `json:"active"`
}
type createDeviceRequest struct {
	Name     string `json:"name"`
	Password string
	Active   bool
}

func (d *devices) Update() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		manifest := updateDeviceRequest{}
		err := json.NewDecoder(r.Body).Decode(&manifest)
		if err != nil {
			log.Print(err)
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(err)
			return
		}
		if manifest.Active != nil {
			if *manifest.Active == false {
				_, err = d.vespiary.DisableDevice(r.Context(), &vespiary.DisableDeviceRequest{Owner: authContext.Tenant, ID: ps.ByName("device_id")})
			} else {
				_, err = d.vespiary.EnableDevice(r.Context(), &vespiary.EnableDeviceRequest{Owner: authContext.Tenant, ID: ps.ByName("device_id")})
			}
			if err != nil {
				log.Print(err)
				w.WriteHeader(500)
				return
			}
		}
	}
}

func fillWithMetadata(tenant string, sessions []*wasp.SessionMetadatas, client *device) {
	for idx := range sessions {
		session := sessions[idx]
		if session.MountPoint == tenant && session.ClientID == client.Name {
			client.Connected = true
			return
		}
	}
}

func fillWithSubscriptions(tenant string, subscriptions []*wasp.CreateSubscriptionRequest, sessions []*wasp.SessionMetadatas, client *device) {

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

func mapDevice(vespiaryDevice *vespiary.Device, subscriptions []*wasp.CreateSubscriptionRequest, sessions []*wasp.SessionMetadatas) device {
	out := device{
		ID:        vespiaryDevice.ID,
		Name:      vespiaryDevice.Name,
		Active:    vespiaryDevice.Active,
		CreatedAt: vespiaryDevice.CreatedAt,
		Password:  vespiaryDevice.Password,

		Connected:         false,
		SentBytes:         0,
		ReceivedBytes:     0,
		SubscriptionCount: 0,
	}
	fillWithMetadata(vespiaryDevice.Owner, sessions, &out)
	fillWithSubscriptions(vespiaryDevice.Owner, subscriptions, sessions, &out)
	if out.Active {
		if out.Connected {
			out.HumanStatus = "online"
		} else {
			out.HumanStatus = "offline"
		}
	} else {
		out.HumanStatus = "disabled"
	}
	return out
}

func (d *devices) List() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		authDevices, err := d.vespiary.ListDevices(r.Context(), &vespiary.ListDevicesRequest{Owner: authContext.Tenant})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch device list"}`))
			return
		}
		sessions, err := d.wasp.ListSessionMetadatas(r.Context(), &wasp.ListSessionMetadatasRequest{})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch connected session list"}`))
			return
		}
		subscriptions, err := d.wasp.ListSubscriptions(r.Context(), &wasp.ListSubscriptionsRequest{})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch subscription list"}`))
			return
		}

		out := make([]device, len(authDevices.Devices))
		for idx := range out {
			out[idx] = mapDevice(authDevices.Devices[idx], subscriptions.Subscriptions, sessions.SessionMetadatasList)
		}
		json.NewEncoder(w).Encode(out)
	}
}
func (d *devices) Get() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())

		sessions, err := d.wasp.ListSessionMetadatas(r.Context(), &wasp.ListSessionMetadatasRequest{})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch connected session list"}`))
			return
		}
		subscriptions, err := d.wasp.ListSubscriptions(r.Context(), &wasp.ListSubscriptionsRequest{})
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"status_code": 502, "message": "failed to fetch subscription list"}`))
			return
		}

		device, err := d.vespiary.GetDevice(r.Context(), &vespiary.GetDeviceRequest{Owner: authContext.Tenant, ID: ps.ByName("device_id")})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
		json.NewEncoder(w).Encode(mapDevice(device.Device, subscriptions.Subscriptions, sessions.SessionMetadatasList))
	}
}

func (d *devices) Delete() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())

		_, err := d.vespiary.DeleteDevice(r.Context(), &vespiary.DeleteDeviceRequest{Owner: authContext.Tenant, ID: ps.ByName("device_id")})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
	}
}
func (d *devices) Create() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := auth.Informations(r.Context())
		manifest := createDeviceRequest{}
		err := json.NewDecoder(r.Body).Decode(&manifest)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status_code": 400, "message": "malformed JSON"`))
			return
		}

		if manifest.Name == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status_code": 400, "message": "invalid device name provided"`))
			return
		}
		if manifest.Password == "" && manifest.Active {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"status_code": 400, "message": "device password must be provided if active is set"`))
			return
		}

		_, err = d.vespiary.CreateDevice(r.Context(), &vespiary.CreateDeviceRequest{
			Owner:    authContext.Tenant,
			Name:     manifest.Name,
			Active:   manifest.Active,
			Password: manifest.Password,
		})
		if err != nil {
			log.Print(err)
			w.WriteHeader(500)
			return
		}
	}
}
