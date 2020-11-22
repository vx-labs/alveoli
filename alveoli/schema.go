package alveoli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/vx-labs/alveoli/alveoli/auth"
	nest "github.com/vx-labs/nest/nest/api"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/v4/wasp/api"
)

func Schema(vespiaryClient vespiary.VespiaryClient, waspClient wasp.MQTTClient, nestClient nest.MessagesClient) graphql.Schema {
	accountType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Account",
		Description: "A user account.",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The id of the account.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if account, ok := p.Source.(auth.UserMetadata); ok {
						return account.AccountID, nil
					}
					return nil, nil
				},
			},
			"name": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The account name.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if acccount, ok := p.Source.(auth.UserMetadata); ok {
						return acccount.Name, nil
					}
					return nil, nil
				},
			},
		},
	})
	sessionType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Session",
		Description: "A connected session.",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The unique id of the session.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if session, ok := p.Source.(*wasp.SessionMetadatas); ok {
						tokens := strings.SplitN(session.SessionID, "/", 2)
						if len(tokens) != 2 {
							return nil, errors.New("failed to find id in session id")
						}
						return tokens[1], nil
					}
					return nil, nil
				},
			},
			"applicationProfileId": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The application profile this session belongs to.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if session, ok := p.Source.(*wasp.SessionMetadatas); ok {
						tokens := strings.SplitN(session.SessionID, "/", 2)
						if len(tokens) != 2 {
							return nil, errors.New("failed to find applicationProfileId in session id")
						}
						return tokens[0], nil
					}
					return nil, nil
				},
			},
			"applicationId": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The application this session belongs to.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if session, ok := p.Source.(*wasp.SessionMetadatas); ok {
						tokens := strings.SplitN(session.MountPoint, "/", 2)
						if len(tokens) != 2 {
							return nil, errors.New("failed to find applicationeId in session id")
						}
						return tokens[1], nil
					}
					return nil, nil
				},
			},
			"connectedAt": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "The time this session has logged in.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if session, ok := p.Source.(*wasp.SessionMetadatas); ok {
						return session.ConnectedAt, nil
					}
					return nil, nil
				},
			},
			"clientId": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The session's MQTT client-id.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if session, ok := p.Source.(*wasp.SessionMetadatas); ok {
						return session.ClientID, nil
					}
					return nil, nil
				},
			},
		},
	})
	recordType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Record",
		Description: "A message record.",
		Fields: graphql.Fields{
			"topicName": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The topic this message was published to.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if record, ok := p.Source.(*nest.Record); ok {
						tokens := strings.SplitN(string(record.Topic), "/", 3)
						if len(tokens) != 3 {
							return nil, errors.New("failed to extract name from topic")
						}
						return tokens[2], nil
					}
					return nil, nil
				},
			},
			"applicationId": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The id of the application profile.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if record, ok := p.Source.(*nest.Record); ok {
						tokens := strings.SplitN(string(record.Topic), "/", 3)
						if len(tokens) != 3 {
							return nil, errors.New("failed to extract name from topic")
						}
						return tokens[1], nil
					}
					return nil, nil
				},
			},
			"payload": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The record payload.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if record, ok := p.Source.(*nest.Record); ok {
						return string(record.Payload), nil
					}
					return nil, nil
				},
			},
			"sentBy": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The sender ID.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if record, ok := p.Source.(*nest.Record); ok {
						return string(record.Sender), nil
					}
					return nil, nil
				},
			},
			"sentAt": &graphql.Field{
				Type:        graphql.DateTime,
				Description: "The time this message was published.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					record := p.Source.(*nest.Record)
					return time.Unix(0, record.Timestamp), nil
				},
			},
		},
	})
	topicType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Topic",
		Description: "A message topic.",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The name of the topic.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if topic, ok := p.Source.(*nest.TopicMetadata); ok {
						tokens := strings.SplitN(string(topic.Name), "/", 3)
						if len(tokens) != 3 {
							return nil, errors.New("failed to extract name from topic")
						}
						return tokens[2], nil
					}
					return nil, nil
				},
			},
			"applicationId": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The id of the application profile.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if topic, ok := p.Source.(*nest.TopicMetadata); ok {
						tokens := strings.SplitN(string(topic.Name), "/", 3)
						if len(tokens) != 3 {
							return nil, errors.New("failed to extract name from topic")
						}
						return tokens[1], nil
					}
					return nil, nil
				},
			},
			"guessedContentType": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The guessed content-type.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if topic, ok := p.Source.(*nest.TopicMetadata); ok {
						return topic.GuessedContentType, nil
					}
					return nil, nil
				},
			},
			"messageCount": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "The number of messages in the topic.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if topic, ok := p.Source.(*nest.TopicMetadata); ok {
						return topic.MessageCount, nil
					}
					return nil, nil
				},
			},
			"sizeInBytes": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "The size of the topic in bytes.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if topic, ok := p.Source.(*nest.TopicMetadata); ok {
						return topic.SizeInBytes, nil
					}
					return nil, nil
				},
			},
			"lastRecord": &graphql.Field{
				Type:        recordType,
				Description: "The last record published in this topic.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if topic, ok := p.Source.(*nest.TopicMetadata); ok {
						return topic.LastRecord, nil
					}
					return nil, nil
				},
			},
			"records": &graphql.Field{
				Type:        &graphql.List{OfType: recordType},
				Description: "The records published in this topic.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if topic, ok := p.Source.(*nest.TopicMetadata); ok {
						stream, err := nestClient.GetTopics(p.Context, &nest.GetTopicsRequest{
							Pattern:       topic.Name,
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
					return nil, nil
				},
			},
		},
	})

	applicationProfileType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "ApplicationProfile",
		Description: "An application profile.",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The id of the application profile.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if applicationProfile, ok := p.Source.(*vespiary.ApplicationProfile); ok {
						return applicationProfile.ID, nil
					}
					return nil, nil
				},
			},
			"applicationId": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The application of the application profile.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if applicationProfile, ok := p.Source.(*vespiary.ApplicationProfile); ok {
						return applicationProfile.ApplicationID, nil
					}
					return nil, nil
				},
			},
			"name": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The application name.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if applicationProfile, ok := p.Source.(*vespiary.ApplicationProfile); ok {
						return applicationProfile.Name, nil
					}
					return nil, nil
				},
			},
			"enabled": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Is this application profile enabled?",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if applicationProfile, ok := p.Source.(*vespiary.ApplicationProfile); ok {
						return applicationProfile.Enabled, nil
					}
					return nil, nil
				},
			},
			"sessions": &graphql.Field{
				Type:        &graphql.List{OfType: sessionType},
				Description: "Connected sessions using this application profile.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if applicationProfile, ok := p.Source.(*vespiary.ApplicationProfile); ok {
						authContext := auth.Informations(p.Context)
						out, err := waspClient.ListSessionMetadatas(p.Context, &wasp.ListSessionMetadatasRequest{})
						if err != nil {
							return nil, err
						}
						filtered := make([]*wasp.SessionMetadatas, 0)
						for _, sessionMetadatas := range out.SessionMetadatasList {
							if strings.HasPrefix(sessionMetadatas.MountPoint, authContext.AccountID) && strings.HasPrefix(sessionMetadatas.SessionID, applicationProfile.ID) {
								filtered = append(filtered, sessionMetadatas)
							}
						}
						return filtered, nil
					}
					return nil, nil
				},
			},
		},
	})
	applicationType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Application",
		Description: "An application.",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "The id of the application.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if application, ok := p.Source.(*vespiary.Application); ok {
						return application.ID, nil
					}
					return nil, nil
				},
			},
			"name": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "The application name.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if application, ok := p.Source.(*vespiary.Application); ok {
						return application.Name, nil
					}
					return nil, nil
				},
			},
			"profiles": &graphql.Field{
				Type:        &graphql.List{OfType: applicationProfileType},
				Description: "The application profiles.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)

					if application, ok := p.Source.(*vespiary.Application); ok {
						out, err := vespiaryClient.ListApplicationProfilesByApplication(p.Context,
							&vespiary.ListApplicationProfilesByApplicationRequest{
								ApplicationID: application.ID,
								AccountID:     authContext.AccountID,
							})
						if err != nil {
							return nil, err
						}
						return out.ApplicationProfiles, nil
					}
					return nil, nil
				},
			},
			"topics": &graphql.Field{
				Description: "The message topics published in this application",
				Type:        &graphql.List{OfType: topicType},
				Args: graphql.FieldConfigArgument{
					"pattern": &graphql.ArgumentConfig{
						Description:  "A pattern to match topics.",
						Type:         graphql.String,
						DefaultValue: nil,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					if application, ok := p.Source.(*vespiary.Application); ok {
						pattern := "#"
						if v, ok := p.Args["pattern"]; ok && v != nil {
							pattern = v.(string)
						}
						finalPattern := []byte(fmt.Sprintf("%s/%s/%s", authContext.AccountID, application.ID, pattern))
						out, err := nestClient.ListTopics(p.Context, &nest.ListTopicsRequest{
							Pattern: finalPattern,
						})
						if err != nil {
							return nil, err
						}
						return out.TopicMetadatas, nil
					}
					return nil, nil
				},
			},
			"sessions": &graphql.Field{
				Type:        &graphql.List{OfType: sessionType},
				Description: "Connected sessions using this application.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if application, ok := p.Source.(*vespiary.Application); ok {
						authContext := auth.Informations(p.Context)
						out, err := waspClient.ListSessionMetadatas(p.Context, &wasp.ListSessionMetadatasRequest{})
						if err != nil {
							return nil, err
						}
						filtered := make([]*wasp.SessionMetadatas, 0)
						for _, sessionMetadatas := range out.SessionMetadatasList {
							if strings.HasPrefix(sessionMetadatas.MountPoint, authContext.AccountID) && strings.HasSuffix(sessionMetadatas.MountPoint, application.ID) {
								filtered = append(filtered, sessionMetadatas)
							}
						}
						return filtered, nil
					}
					return nil, nil
				},
			},
		},
	})
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"account": &graphql.Field{
				Type: accountType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					return authContext, nil
				},
			},
			"topics": &graphql.Field{
				Type: &graphql.List{OfType: topicType},
				Args: graphql.FieldConfigArgument{
					"pattern": &graphql.ArgumentConfig{
						Description:  "A pattern to match topics.",
						Type:         graphql.String,
						DefaultValue: nil,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					pattern := "#"
					if v, ok := p.Args["pattern"]; ok && v != nil {
						pattern = v.(string)
					}
					finalPattern := []byte(fmt.Sprintf("%s/+/%s", authContext.AccountID, pattern))
					out, err := nestClient.ListTopics(p.Context, &nest.ListTopicsRequest{
						Pattern: finalPattern,
					})
					if err != nil {
						return nil, err
					}
					return out.TopicMetadatas, nil
				},
			},
			"application": &graphql.Field{
				Type: applicationType,
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Description: "The name of the application.",
						Type:        graphql.String,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					if p.Args["name"] != nil {
						authContext := auth.Informations(p.Context)
						out, err := vespiaryClient.GetApplicationByName(p.Context, &vespiary.GetApplicationByNameRequest{
							Name:      p.Args["name"].(string),
							AccountID: authContext.AccountID,
						})
						if err != nil {
							return nil, err
						}
						return out.Application, nil
					}
					return nil, errors.New("missing name")
				},
			},
			"sessions": &graphql.Field{
				Type: &graphql.List{OfType: sessionType},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					out, err := waspClient.ListSessionMetadatas(p.Context, &wasp.ListSessionMetadatasRequest{})
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
				},
			},
			"applications": &graphql.Field{
				Type: &graphql.List{OfType: applicationType},
				Args: graphql.FieldConfigArgument{},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					out, err := vespiaryClient.ListApplicationsByAccountID(p.Context,
						&vespiary.ListApplicationsByAccountIDRequest{
							AccountID: authContext.AccountID,
						})
					if err != nil {
						return nil, err
					}
					return out.Applications, nil
				},
			},
			"applicationProfiles": &graphql.Field{
				Type: &graphql.List{OfType: applicationProfileType},
				Args: graphql.FieldConfigArgument{
					"applicationId": &graphql.ArgumentConfig{
						Description: "id of the application",
						Type:        graphql.String,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					switch p.Args["applicationId"] {
					case nil:
						out, err := vespiaryClient.ListApplicationProfilesByAccountID(p.Context,
							&vespiary.ListApplicationProfilesByAccountIDRequest{
								AccountID: authContext.AccountID,
							})
						if err != nil {
							return nil, err
						}
						return out.ApplicationProfiles, nil
					default:
						out, err := vespiaryClient.ListApplicationProfilesByApplication(p.Context,
							&vespiary.ListApplicationProfilesByApplicationRequest{
								ApplicationID: p.Args["applicationId"].(string),
								AccountID:     authContext.AccountID,
							})
						if err != nil {
							return nil, err
						}
						return out.ApplicationProfiles, nil
					}
				},
			},
		},
	})

	mutations := graphql.NewObject(graphql.ObjectConfig{
		Name: "mutations",
		Fields: graphql.Fields{
			"deleteApplication": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "Delete an application",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					_, err := vespiaryClient.DeleteApplicationByAccountID(p.Context, &vespiary.DeleteApplicationByAccountIDRequest{
						AccountID: authContext.AccountID,
						ID:        p.Args["id"].(string),
					})
					if err != nil {
						return nil, err
					}
					return p.Args["id"].(string), nil
				},
			},
			"createApplication": &graphql.Field{
				Type:        graphql.NewNonNull(applicationType),
				Description: "Create a new application",
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					create, err := vespiaryClient.CreateApplication(p.Context, &vespiary.CreateApplicationRequest{
						AccountID: authContext.AccountID,
						Name:      p.Args["name"].(string),
					})
					if err != nil {
						return nil, err
					}
					out, err := vespiaryClient.GetApplicationByAccountID(p.Context, &vespiary.GetApplicationByAccountIDRequest{
						AccountID: authContext.AccountID,
						Id:        create.ID,
					})
					if err != nil {
						return nil, err
					}
					return out.Application, nil
				},
			},
			"createApplicationProfile": &graphql.Field{
				Type:        graphql.NewNonNull(applicationProfileType),
				Description: "Create a new application profile",
				Args: graphql.FieldConfigArgument{
					"applicationId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"name": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
					"password": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					create, err := vespiaryClient.CreateApplicationProfile(p.Context, &vespiary.CreateApplicationProfileRequest{
						AccountID:     authContext.AccountID,
						ApplicationID: p.Args["applicationId"].(string),
						Name:          p.Args["name"].(string),
						Password:      p.Args["password"].(string),
					})
					if err != nil {
						return nil, err
					}
					out, err := vespiaryClient.GetApplicationProfileByAccountID(p.Context, &vespiary.GetApplicationProfileByAccountIDRequest{
						AccountID: authContext.AccountID,
						ID:        create.ID,
					})
					if err != nil {
						return nil, err
					}
					return out.ApplicationProfile, nil
				},
			},
			"deleteApplicationProfile": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.ID),
				Description: "Delete an application profile",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					authContext := auth.Informations(p.Context)
					_, err := vespiaryClient.DeleteApplicationProfileByAccountID(p.Context, &vespiary.DeleteApplicationProfileByAccountIDRequest{
						AccountID: authContext.AccountID,
						ID:        p.Args["id"].(string),
					})
					if err != nil {
						return nil, err
					}
					return p.Args["id"].(string), nil
				},
			},
		},
	})
	vespiarySchema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutations,
	})
	if err != nil {
		panic(err)
	}
	return vespiarySchema
}
