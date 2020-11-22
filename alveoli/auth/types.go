package auth

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type vxContextKey string

const userInformationsContextKey vxContextKey = "vx:user_informations"

// UserMetadata contains informations about a user.
type UserMetadata struct {
	AccountID       string
	Name            string
	DeviceUsernames []string
	Principal       string
}

// Provider handles an http request, and injects user informations in context "User" value.
type Provider interface {
	Handler(h http.Handler) http.Handler
	ResolveUserEmail(header string) (string, error)
}

func storeInformations(ctx context.Context, md UserMetadata) context.Context {
	return context.WithValue(ctx, userInformationsContextKey, md)
}

func Informations(ctx context.Context) UserMetadata {
	return ctx.Value(userInformationsContextKey).(UserMetadata)
}

func RequireAccountCreated(f func(w http.ResponseWriter, r *http.Request, ps httprouter.Params)) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		authContext := Informations(r.Context())
		if authContext.AccountID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status_code": 401, "message": "account not registered"}`))
			return
		}
		f(w, r, ps)
	}
}
