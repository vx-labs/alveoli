package resolvers

import (
	"context"
	"errors"
	"strings"
	"time"

	wasp "github.com/vx-labs/wasp/v4/wasp/api"
)

type sessionResolver struct {
	*Resolver
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
	tokens := strings.SplitN(obj.SessionID, "/", 2)
	if len(tokens) != 2 {
		return "", errors.New("failed to find applicationProfileId in session id")
	}
	return tokens[0], nil
}
func (s *sessionResolver) ApplicationProfileID(ctx context.Context, obj *wasp.SessionMetadatas) (string, error) {
	tokens := strings.SplitN(obj.MountPoint, "/", 2)
	if len(tokens) != 2 {
		return "", errors.New("failed to find applicationeId in session id")
	}
	return tokens[1], nil
}
func (s *sessionResolver) ConnectedAt(ctx context.Context, obj *wasp.SessionMetadatas) (*time.Time, error) {
	t := time.Unix(0, obj.ConnectedAt)
	return &t, nil
}
