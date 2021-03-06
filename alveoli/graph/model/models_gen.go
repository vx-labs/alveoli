// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"

	"github.com/vx-labs/vespiary/vespiary/api"
	api1 "github.com/vx-labs/wasp/v4/wasp/api"
)

type AuditEventPayload interface {
	IsAuditEventPayload()
}

type ApplicationCreatedEvent struct {
	Application *api.Application `json:"application"`
}

func (ApplicationCreatedEvent) IsAuditEventPayload() {}

type ApplicationDeletedEvent struct {
	ID string `json:"id"`
}

func (ApplicationDeletedEvent) IsAuditEventPayload() {}

type ApplicationProfileCreatedEvent struct {
	ApplicationProfile *api.ApplicationProfile `json:"applicationProfile"`
}

func (ApplicationProfileCreatedEvent) IsAuditEventPayload() {}

type ApplicationProfileDeletedEvent struct {
	ID string `json:"id"`
}

func (ApplicationProfileDeletedEvent) IsAuditEventPayload() {}

type AuditEvent struct {
	Type    AuditEventType    `json:"type"`
	Payload AuditEventPayload `json:"payload"`
}

type CreateApplicationOutput struct {
	Application *api.Application `json:"application"`
	Success     bool             `json:"success"`
}

type CreateApplicationProfileOutput struct {
	ApplicationProfile *api.ApplicationProfile `json:"applicationProfile"`
	Success            bool                    `json:"success"`
}

type SessionConnectedEvent struct {
	Session *api1.SessionMetadatas `json:"session"`
}

func (SessionConnectedEvent) IsAuditEventPayload() {}

type SessionDisconnectedEvent struct {
	ID string `json:"id"`
}

func (SessionDisconnectedEvent) IsAuditEventPayload() {}

type AuditEventType string

const (
	AuditEventTypeApplicationCreated        AuditEventType = "applicationCreated"
	AuditEventTypeApplicationDeleted        AuditEventType = "applicationDeleted"
	AuditEventTypeApplicationProfileCreated AuditEventType = "applicationProfileCreated"
	AuditEventTypeApplicationProfileDeleted AuditEventType = "applicationProfileDeleted"
	AuditEventTypeSessionConnected          AuditEventType = "sessionConnected"
	AuditEventTypeSessionDisconnected       AuditEventType = "sessionDisconnected"
)

var AllAuditEventType = []AuditEventType{
	AuditEventTypeApplicationCreated,
	AuditEventTypeApplicationDeleted,
	AuditEventTypeApplicationProfileCreated,
	AuditEventTypeApplicationProfileDeleted,
	AuditEventTypeSessionConnected,
	AuditEventTypeSessionDisconnected,
}

func (e AuditEventType) IsValid() bool {
	switch e {
	case AuditEventTypeApplicationCreated, AuditEventTypeApplicationDeleted, AuditEventTypeApplicationProfileCreated, AuditEventTypeApplicationProfileDeleted, AuditEventTypeSessionConnected, AuditEventTypeSessionDisconnected:
		return true
	}
	return false
}

func (e AuditEventType) String() string {
	return string(e)
}

func (e *AuditEventType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = AuditEventType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid AuditEventType", str)
	}
	return nil
}

func (e AuditEventType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
