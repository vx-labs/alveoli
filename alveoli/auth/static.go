package auth

import (
	"context"
)

type static struct {
	accountID string
	tenant    string
}

func (l *static) Authenticate(ctx context.Context, token string) (string, error) {
	return l.tenant, nil
}
func (l *static) Validate(ctx context.Context, token string) (UserMetadata, error) {
	return UserMetadata{
		Principal:       l.tenant,
		Name:            "mocked static account",
		AccountID:       l.accountID,
		DeviceUsernames: []string{l.tenant},
	}, nil
}
func (l *static) ResolveUserEmail(header string) (string, error) {
	return "test@example.net", nil
}

func Static(accountID, tenant string) Provider {
	return &static{tenant: tenant, accountID: accountID}
}
