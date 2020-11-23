package resolvers

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/vx-labs/alveoli/alveoli/auth"
	nest "github.com/vx-labs/nest/nest/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
)

type applicationResolver struct {
	*Resolver
}

func (a *applicationResolver) ID(ctx context.Context, obj *vespiary.Application) (string, error) {
	return obj.ID, nil

}
func (a *applicationResolver) Name(ctx context.Context, obj *vespiary.Application) (string, error) {
	return obj.Name, nil
}

func (a *applicationResolver) Profiles(ctx context.Context, obj *vespiary.Application) ([]*vespiary.ApplicationProfile, error) {
	authContext := auth.Informations(ctx)
	out, err := a.Vespiary.ListApplicationProfilesByApplication(ctx,
		&vespiary.ListApplicationProfilesByApplicationRequest{
			ApplicationID: obj.ID,
			AccountID:     authContext.AccountID,
		})
	if err != nil {
		return nil, err
	}
	return out.ApplicationProfiles, nil
}
func (a *applicationResolver) Records(ctx context.Context, obj *vespiary.Application, userPattern *string) ([]*nest.Record, error) {
	authContext := auth.Informations(ctx)
	pattern := "#"
	if userPattern != nil {
		pattern = *userPattern
	}
	finalPattern := []byte(fmt.Sprintf("%s/%s/%s", authContext.AccountID, obj.ID, pattern))

	stream, err := a.Nest.GetTopics(ctx, &nest.GetTopicsRequest{
		Pattern:       finalPattern,
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
func (a *applicationResolver) Topics(ctx context.Context, obj *vespiary.Application, userPattern *string) ([]*nest.TopicMetadata, error) {
	authContext := auth.Informations(ctx)
	pattern := "#"
	if userPattern != nil {
		pattern = *userPattern
	}
	finalPattern := []byte(fmt.Sprintf("%s/%s/%s", authContext.AccountID, obj.ID, pattern))
	out, err := a.Nest.ListTopics(ctx, &nest.ListTopicsRequest{
		Pattern: finalPattern,
	})
	if err != nil {
		return nil, err
	}
	return out.TopicMetadatas, nil
}
