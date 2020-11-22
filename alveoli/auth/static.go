package auth

import "net/http"

type static struct {
	accountID string
	tenant    string
}

func (l *static) ResolveUserEmail(header string) (string, error) {
	return "test@example.net", nil
}
func (l *static) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(storeInformations(r.Context(), UserMetadata{
			Principal:       l.tenant,
			Name:            "mocked static account",
			AccountID:       l.accountID,
			DeviceUsernames: []string{l.tenant},
		})))
	})
}

func Static(accountID, tenant string) Provider {
	return &static{tenant: tenant, accountID: accountID}
}
