package resolvers

import (
	"context"

	"github.com/vx-labs/alveoli/alveoli/auth"
	vespiary "github.com/vx-labs/vespiary/vespiary/api"
)

type applicationProfileResolver struct {
	*resolver
}

func (a *applicationProfileResolver) ID(ctx context.Context, obj *vespiary.ApplicationProfile) (string, error) {
	return obj.ID, nil

}
func (a *applicationProfileResolver) Name(ctx context.Context, obj *vespiary.ApplicationProfile) (string, error) {
	return obj.Name, nil
}
func (a *applicationProfileResolver) ApplicationID(ctx context.Context, obj *vespiary.ApplicationProfile) (string, error) {
	return obj.ApplicationID, nil
}
func (a *applicationProfileResolver) Application(ctx context.Context, obj *vespiary.ApplicationProfile) (*vespiary.Application, error) {
	authContext := auth.Informations(ctx)

	out, err := a.vespiary.GetApplicationByAccountID(ctx, &vespiary.GetApplicationByAccountIDRequest{
		AccountID: authContext.AccountID,
		Id:        obj.ApplicationID,
	})
	if err != nil {
		return nil, err
	}
	return out.Application, nil
}
func (a *applicationProfileResolver) Enabled(ctx context.Context, obj *vespiary.ApplicationProfile) (bool, error) {
	return obj.Enabled, nil
}
