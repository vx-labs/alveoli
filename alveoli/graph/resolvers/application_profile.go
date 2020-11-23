package resolvers

import (
	"context"

	vespiary "github.com/vx-labs/vespiary/vespiary/api"
)

type applicationProfileResolver struct {
	*Resolver
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
func (a *applicationProfileResolver) Enabled(ctx context.Context, obj *vespiary.ApplicationProfile) (bool, error) {
	return obj.Enabled, nil
}
