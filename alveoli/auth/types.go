package auth

import (
	"context"
	"log"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
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
	ResolveUserEmail(header string) (string, error)
	Validate(ctx context.Context, token string) (UserMetadata, error)
}

func StoreInformations(ctx context.Context, md UserMetadata) context.Context {
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

func Handler(provider Provider, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" &&
			r.Header.Get("Sec-Websocket-Protocol") == "graphql-ws" {
			log.Printf("bypassing header auth for websocket")
			next.ServeHTTP(w, r)
			return
		}

		token, err := jwtmiddleware.FromAuthHeader(r)
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status_code": 401, "message": "missing or invalid credentials","reason": "token is empty"}`))
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status_code": 401, "message": "missing or invalid credentials","reason": "` + err.Error() + `"}`))
			return
		}

		md, err := provider.Validate(r.Context(), token)
		if err != nil {
			w.WriteHeader(403)
			w.Write([]byte(`{"message": "permission denied", "status_code": 403, "reason": "` + err.Error() + `"}`))
			return
		}
		log.Printf("authentication done for account %s", md.AccountID)
		next.ServeHTTP(w, r.WithContext(StoreInformations(r.Context(), md)))
	})
}
