package resolvers

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/vx-labs/alveoli/alveoli/auth"
	nest "github.com/vx-labs/nest/nest/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
)

type topicResolver struct {
	*resolver
}

func (r *topicResolver) Name(ctx context.Context, obj *nest.TopicMetadata) (string, error) {
	tokens := strings.SplitN(string(obj.Name), "/", 4)
	if len(tokens) != 4 {
		return "", errors.New("failed to extract name from topic")
	}
	return tokens[3], nil
}
func (r *topicResolver) ApplicationID(ctx context.Context, obj *nest.TopicMetadata) (string, error) {
	tokens := strings.SplitN(string(obj.Name), "/", 4)
	if len(tokens) != 4 {
		return "", errors.New("failed to extract name from topic")
	}
	return tokens[2], nil
}
func (a *topicResolver) Application(ctx context.Context, obj *nest.TopicMetadata) (*vespiary.Application, error) {
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

func (r *topicResolver) MessageCount(ctx context.Context, obj *nest.TopicMetadata) (int, error) {
	return int(obj.MessageCount), nil
}
func (r *topicResolver) SizeInBytes(ctx context.Context, obj *nest.TopicMetadata) (int, error) {
	return int(obj.SizeInBytes), nil
}
func (r *topicResolver) LastRecord(ctx context.Context, obj *nest.TopicMetadata) (*nest.Record, error) {
	return obj.LastRecord, nil
}
func (r *topicResolver) Records(ctx context.Context, obj *nest.TopicMetadata) ([]*nest.Record, error) {
	stream, err := r.nest.GetTopics(ctx, &nest.GetTopicsRequest{
		Pattern:       obj.Name,
		Watch:         false,
		FromTimestamp: time.Now().Add(-15 * 24 * time.Hour).UnixNano(),
	})
	if err != nil {
		return nil, err
	}
	out := []*nest.Record{}
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return out, nil
			}
			return nil, err
		}
		out = append(out, msg.Records...)
	}
}
