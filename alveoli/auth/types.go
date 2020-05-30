package auth

import (
	"context"
	"net/http"
)

type vxContextKey string

const userInformationsContextKey vxContextKey = "vx:user_informations"

// UserMetadata contains informations about a user.
type UserMetadata struct {
	Tenant string
}

// Provider handles an http request, and injects user informations in context "User" value.
type Provider interface {
	Handler(h http.Handler) http.Handler
}

func storeInformations(ctx context.Context, md UserMetadata) context.Context {
	return context.WithValue(ctx, userInformationsContextKey, md)
}

func Informations(ctx context.Context) UserMetadata {
	return ctx.Value(userInformationsContextKey).(UserMetadata)
}
