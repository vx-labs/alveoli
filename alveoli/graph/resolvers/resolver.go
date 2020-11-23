package resolvers

//go:generate go run github.com/99designs/gqlgen --verbose
import (
	"context"
	"fmt"
	"strings"

	"github.com/vx-labs/alveoli/alveoli/auth"
	"github.com/vx-labs/alveoli/alveoli/graph/generated"
	nest "github.com/vx-labs/nest/nest/api"
	"github.com/vx-labs/vespiary/vespiary/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/v4/wasp/api"
)

type Resolver struct {
	Nest     nest.MessagesClient
	Wasp     wasp.MQTTClient
	Vespiary vespiary.VespiaryClient
}

func (r *queryResolver) Account(ctx context.Context) (*vespiary.Account, error) {
	authContext := auth.Informations(ctx)
	return &vespiary.Account{
		ID:   authContext.AccountID,
		Name: authContext.Name,
	}, nil
}
func (r *queryResolver) Sessions(ctx context.Context) ([]*wasp.SessionMetadatas, error) {
	authContext := auth.Informations(ctx)
	out, err := r.Wasp.ListSessionMetadatas(ctx, &wasp.ListSessionMetadatasRequest{})
	if err != nil {
		return nil, err
	}
	filtered := make([]*wasp.SessionMetadatas, 0)
	for _, sessionMetadatas := range out.SessionMetadatasList {
		if strings.HasPrefix(sessionMetadatas.MountPoint, authContext.AccountID) {
			filtered = append(filtered, sessionMetadatas)
		}
	}
	return filtered, nil
}

func (r *queryResolver) Applications(ctx context.Context) ([]*api.Application, error) {
	authContext := auth.Informations(ctx)
	out, err := r.Vespiary.ListApplicationsByAccountID(ctx, &vespiary.ListApplicationsByAccountIDRequest{
		AccountID: authContext.AccountID,
	})
	if err != nil {
		return nil, err
	}
	return out.Applications, nil
}

func (r *queryResolver) ApplicationProfiles(ctx context.Context) ([]*api.ApplicationProfile, error) {
	authContext := auth.Informations(ctx)
	out, err := r.Vespiary.ListApplicationProfilesByAccountID(ctx, &vespiary.ListApplicationProfilesByAccountIDRequest{
		AccountID: authContext.AccountID,
	})
	if err != nil {
		return nil, err
	}
	return out.ApplicationProfiles, nil
}
func (r *queryResolver) Topics(ctx context.Context, userPattern *string) ([]*nest.TopicMetadata, error) {
	authContext := auth.Informations(ctx)
	pattern := "#"
	if userPattern != nil {
		pattern = *userPattern
	}
	finalPattern := []byte(fmt.Sprintf("%s/+/%s", authContext.AccountID, pattern))
	out, err := r.Nest.ListTopics(ctx, &nest.ListTopicsRequest{
		Pattern: finalPattern,
	})
	if err != nil {
		return nil, err
	}
	return out.TopicMetadatas, nil
}

func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }
func (r *Resolver) ApplicationProfile() generated.ApplicationProfileResolver {
	return &applicationProfileResolver{r}
}
func (r *Resolver) Application() generated.ApplicationResolver { return &applicationResolver{r} }
func (r *Resolver) Record() generated.RecordResolver           { return &recordResolver{r} }
func (r *Resolver) Topic() generated.TopicResolver             { return &topicResolver{r} }
func (r *Resolver) Session() generated.SessionResolver         { return &sessionResolver{r} }

type queryResolver struct{ *Resolver }
