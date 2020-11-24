package resolvers

//go:generate go run github.com/99designs/gqlgen --verbose
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/vx-labs/alveoli/alveoli/auth"
	"github.com/vx-labs/alveoli/alveoli/graph/generated"
	"github.com/vx-labs/alveoli/alveoli/graph/model"
	nest "github.com/vx-labs/nest/nest/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/v4/wasp/api"
)

type resolver struct {
	nest     nest.MessagesClient
	wasp     wasp.MQTTClient
	vespiary vespiary.VespiaryClient
	mqtt     mqtt.Client
}

func Root(mqttClient mqtt.Client, waspClient wasp.MQTTClient, vespiaryClient vespiary.VespiaryClient, nestClient nest.MessagesClient) generated.ResolverRoot {
	return &resolver{
		nest:     nestClient,
		wasp:     waspClient,
		vespiary: vespiaryClient,
		mqtt:     mqttClient,
	}
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
	out, err := r.wasp.ListSessionMetadatas(ctx, &wasp.ListSessionMetadatasRequest{})
	if err != nil {
		return nil, err
	}
	filtered := make([]*wasp.SessionMetadatas, 0)
	for _, sessionMetadatas := range out.SessionMetadatasList {
		if strings.HasPrefix(sessionMetadatas.MountPoint, fmt.Sprintf("_root/%s", authContext.AccountID)) {
			filtered = append(filtered, sessionMetadatas)
		}
	}
	return filtered, nil
}

func (r *queryResolver) Applications(ctx context.Context) ([]*vespiary.Application, error) {
	authContext := auth.Informations(ctx)
	out, err := r.vespiary.ListApplicationsByAccountID(ctx, &vespiary.ListApplicationsByAccountIDRequest{
		AccountID: authContext.AccountID,
	})
	if err != nil {
		return nil, err
	}
	return out.Applications, nil
}

func (r *queryResolver) ApplicationProfiles(ctx context.Context) ([]*vespiary.ApplicationProfile, error) {
	authContext := auth.Informations(ctx)
	out, err := r.vespiary.ListApplicationProfilesByAccountID(ctx, &vespiary.ListApplicationProfilesByAccountIDRequest{
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
	finalPattern := []byte(fmt.Sprintf("_root/%s/+/%s", authContext.AccountID, pattern))
	out, err := r.nest.ListTopics(ctx, &nest.ListTopicsRequest{
		Pattern: finalPattern,
	})
	if err != nil {
		return nil, err
	}
	return out.TopicMetadatas, nil
}

type queryResolver struct{ *resolver }

func (r *resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *resolver }

func (m *mutationResolver) DeleteAccount(ctx context.Context) (string, error) {
	authContext := auth.Informations(ctx)
	_, err := m.vespiary.DeleteAccount(ctx, &vespiary.DeleteAccountRequest{
		ID: authContext.AccountID,
	})
	if err != nil {
		return "", err
	}
	return authContext.AccountID, nil
}
func (m *mutationResolver) CreateApplication(ctx context.Context, input vespiary.CreateApplicationRequest) (*model.CreateApplicationOutput, error) {
	authContext := auth.Informations(ctx)
	out, err := m.vespiary.CreateApplication(ctx, &vespiary.CreateApplicationRequest{
		AccountID: authContext.AccountID,
		Name:      input.Name,
	})
	if err != nil {
		return nil, err
	}
	resp, err := m.vespiary.GetApplicationByAccountID(ctx, &vespiary.GetApplicationByAccountIDRequest{
		AccountID: authContext.AccountID,
		Id:        out.ID,
	})
	if err != nil {
		return nil, err
	}
	return &model.CreateApplicationOutput{
		Application: resp.Application,
		Success:     true,
	}, nil
}
func (m *mutationResolver) DeleteApplication(ctx context.Context, id string) (string, error) {
	authContext := auth.Informations(ctx)

	_, err := m.vespiary.DeleteApplicationByAccountID(ctx, &vespiary.DeleteApplicationByAccountIDRequest{
		AccountID: authContext.AccountID,
		ID:        id,
	})
	return id, err
}

func (m *mutationResolver) CreateApplicationProfile(ctx context.Context, input vespiary.CreateApplicationProfileRequest) (*model.CreateApplicationProfileOutput, error) {
	authContext := auth.Informations(ctx)
	out, err := m.vespiary.CreateApplicationProfile(ctx, &vespiary.CreateApplicationProfileRequest{
		AccountID:     authContext.AccountID,
		Name:          input.Name,
		ApplicationID: input.ApplicationID,
		Password:      input.Password,
	})
	if err != nil {
		return nil, err
	}
	resp, err := m.vespiary.GetApplicationProfileByAccountID(ctx, &vespiary.GetApplicationProfileByAccountIDRequest{
		AccountID: authContext.AccountID,
		ID:        out.ID,
	})
	if err != nil {
		return nil, err
	}
	return &model.CreateApplicationProfileOutput{
		ApplicationProfile: resp.ApplicationProfile,
		Success:            true,
	}, nil
}

func (m *mutationResolver) DeleteApplicationProfile(ctx context.Context, id string) (string, error) {
	authContext := auth.Informations(ctx)

	_, err := m.vespiary.DeleteApplicationProfileByAccountID(ctx, &vespiary.DeleteApplicationProfileByAccountIDRequest{
		AccountID: authContext.AccountID,
		ID:        id,
	})
	return id, err
}

type subscriptionResolver struct{ *resolver }

type auditEvent struct {
	Timestamp  int64                  `json:"timestamp,omitempty"`
	Service    string                 `json:"service,omitempty"`
	Kind       string                 `json:"kind,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

func (s *subscriptionResolver) AuditEvents(ctx context.Context) (<-chan *model.AuditEvent, error) {
	authContext := auth.Informations(ctx)
	topic := fmt.Sprintf("%s/$SYS/_audit/events", authContext.AccountID)
	applicationsTopic := fmt.Sprintf("%s/+/$SYS/_audit/events", authContext.AccountID)
	ch := make(chan *model.AuditEvent)
	token := s.mqtt.SubscribeMultiple(map[string]byte{
		topic:             2,
		applicationsTopic: 2,
	}, func(c mqtt.Client, m mqtt.Message) {
		input := auditEvent{}
		err := json.Unmarshal(m.Payload(), &input)
		if err != nil {
			log.Print(err)
			return
		}
		ev := &model.AuditEvent{}
		switch input.Kind {
		case "application_created":
			out, err := s.vespiary.GetApplicationByAccountID(ctx, &vespiary.GetApplicationByAccountIDRequest{
				AccountID: authContext.AccountID,
				Id:        input.Attributes["application_id"].(string),
			})
			if err != nil {
				return
			}
			ev.Type = model.AuditEventTypeApplicationCreated
			ev.Payload = model.ApplicationCreatedEvent{
				Application: out.Application,
			}
		case "application_deleted":
			ev.Type = model.AuditEventTypeApplicationDeleted
			ev.Payload = model.ApplicationDeletedEvent{
				ID: input.Attributes["application_id"].(string),
			}
		case "application_profile_created":
			ev.Type = model.AuditEventTypeApplicationProfileCreated
			out, err := s.vespiary.GetApplicationProfileByAccountID(ctx, &vespiary.GetApplicationProfileByAccountIDRequest{
				AccountID: authContext.AccountID,
				ID:        input.Attributes["application_profile_id"].(string),
			})
			if err != nil {
				return
			}
			ev.Payload = model.ApplicationProfileCreatedEvent{
				ApplicationProfile: out.ApplicationProfile,
			}
		case "application_profile_deleted":
			ev.Type = model.AuditEventTypeApplicationProfileDeleted
			ev.Payload = model.ApplicationProfileDeletedEvent{
				ID: input.Attributes["application_profile_id"].(string),
			}
		case "session_connected":
			ev.Type = model.AuditEventTypeSessionConnected
			tokens := strings.Split(input.Attributes["session_id"].(string), "/")
			ev.Payload = model.SessionConnectedEvent{
				ID:       tokens[len(tokens)-1],
				ClientID: input.Attributes["client_id"].(string),
			}
		case "session_disconnected":
			ev.Type = model.AuditEventTypeSessionDisconnected
			tokens := strings.Split(input.Attributes["session_id"].(string), "/")
			ev.Payload = model.SessionDisconnectedEvent{
				ID: tokens[len(tokens)-1],
			}
		}

		select {
		case <-ctx.Done():
			return
		case ch <- ev:
			return
		}
	})
	go func() {
		<-ctx.Done()
		token := s.mqtt.Unsubscribe(topic)
		token.Wait()
		close(ch)
	}()
	token.Wait()
	if err := token.Error(); err != nil {
		close(ch)
		return nil, err
	}
	return ch, nil
}
func (r *resolver) Mutation() generated.MutationResolver         { return &mutationResolver{r} }
func (r *resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

func (r *resolver) ApplicationProfile() generated.ApplicationProfileResolver {
	return &applicationProfileResolver{r}
}
func (r *resolver) Application() generated.ApplicationResolver { return &applicationResolver{r} }
func (r *resolver) Record() generated.RecordResolver           { return &recordResolver{r} }
func (r *resolver) Topic() generated.TopicResolver             { return &topicResolver{r} }
func (r *resolver) Session() generated.SessionResolver         { return &sessionResolver{r} }
