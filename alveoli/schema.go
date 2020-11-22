package alveoli

import (
	"errors"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/vx-labs/alveoli/alveoli/auth"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
	wasp "github.com/vx-labs/wasp/v4/wasp/api"
)

func Schema(vespiaryClient vespiary.VespiaryClient, waspClient wasp.MQTTClient) graphql.Schema {
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
