package resolvers

import (
	"context"
	"errors"
	"strings"
	"time"

	nest "github.com/vx-labs/nest/nest/api"
)

type recordResolver struct {
	*resolver
}

func (r *recordResolver) TopicName(ctx context.Context, obj *nest.Record) (string, error) {
	tokens := strings.SplitN(string(obj.Topic), "/", 3)
	if len(tokens) != 3 {
		return "", errors.New("failed to extract name from topic")
	}
	return tokens[2], nil
}
func (r *recordResolver) ApplicationID(ctx context.Context, obj *nest.Record) (string, error) {
	tokens := strings.SplitN(string(obj.Topic), "/", 3)
	if len(tokens) != 3 {
		return "", errors.New("failed to extract name from topic")
	}
	return tokens[1], nil
}
func (r *recordResolver) Payload(ctx context.Context, obj *nest.Record) (string, error) {
	return string(obj.Payload), nil
}
func (r *recordResolver) SentBy(ctx context.Context, obj *nest.Record) (string, error) {
	return obj.Sender, nil
}
func (r *recordResolver) SentAt(ctx context.Context, obj *nest.Record) (*time.Time, error) {
	t := time.Unix(0, obj.Timestamp)
	return &t, nil
}
