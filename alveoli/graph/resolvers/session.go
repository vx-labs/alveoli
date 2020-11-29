package resolvers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/vx-labs/alveoli/alveoli/auth"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/v4/wasp/api"
)

type sessionResolver struct {
	*resolver
}

func (s *sessionResolver) ID(ctx context.Context, obj *wasp.SessionMetadatas) (string, error) {
	tokens := strings.SplitN(obj.SessionID, "/", 2)
	if len(tokens) != 2 {
		return "", errors.New("failed to find id in session id")
	}
	return tokens[1], nil
}

func (s *sessionResolver) ClientID(ctx context.Context, obj *wasp.SessionMetadatas) (string, error) {
	return string(obj.ClientID), nil
}
func (s *sessionResolver) ApplicationID(ctx context.Context, obj *wasp.SessionMetadatas) (string, error) {
	tokens := strings.SplitN(obj.MountPoint, "/", 3)
	if len(tokens) != 3 {
		return "", errors.New("failed to find applicationeId in session id")
	}
	return tokens[2], nil
}
func (s *sessionResolver) ConnectedAt(ctx context.Context, obj *wasp.SessionMetadatas) (*time.Time, error) {
	t := time.Unix(0, obj.ConnectedAt)
	return &t, nil
}
func (s *sessionResolver) ApplicationProfileID(ctx context.Context, obj *wasp.SessionMetadatas) (string, error) {
	tokens := strings.SplitN(obj.SessionID, "/", 2)
	if len(tokens) != 2 {
		return "", errors.New("failed to find applicationProfileId in session id")
	}
	return tokens[0], nil
}
func (a *sessionResolver) Application(ctx context.Context, obj *wasp.SessionMetadatas) (*vespiary.Application, error) {
	authContext := auth.Informations(ctx)
	id, err := a.ApplicationID(ctx, obj)
	if err != nil {
		return nil, err
	}
	out, err := a.vespiary.GetApplicationByAccountID(ctx, &vespiary.GetApplicationByAccountIDRequest{
		AccountID: authContext.AccountID,
		Id:        id,
	})
	if err != nil {
		return nil, err
	}
	return out.Application, nil
}
func (a *sessionResolver) ApplicationProfile(ctx context.Context, obj *wasp.SessionMetadatas) (*vespiary.ApplicationProfile, error) {
	authContext := auth.Informations(ctx)
	id, err := a.ApplicationProfileID(ctx, obj)
	if err != nil {
		return nil, err
	}
	out, err := a.vespiary.GetApplicationProfileByAccountID(ctx, &vespiary.GetApplicationProfileByAccountIDRequest{
		AccountID: authContext.AccountID,
		ID:        id,
	})
	if err != nil {
		return nil, err
	}
	return out.ApplicationProfile, nil
}
