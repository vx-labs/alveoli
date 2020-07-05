package auth

import "net/http"

type static struct {
	tenant string
}

func (l *static) ResolveUserEmail(header string) (string, error) {
	return "test@example.net", nil
}
func (l *static) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(storeInformations(r.Context(), UserMetadata{
			Principal:       l.tenant,
			AccountID:       "1",
			DeviceUsernames: []string{l.tenant},
		})))
	})
}

func Static(tenant string) Provider {
	return &static{tenant: tenant}
}
