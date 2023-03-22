package cluster

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// MiscAuthZPermissive is permissive implementation of the MiscAuthZ interface.
type MiscAuthZPermissive struct{}

// CanUpdateAgents calls the RBAC implemenation but always allows access.
func (a *MiscAuthZPermissive) CanUpdateAgents(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	_, _ = (&MiscAuthZRBAC{}).CanUpdateAgents(ctx, curUser)
	return (&MiscAuthZBasic{}).CanUpdateAgents(ctx, curUser)
}

func init() {
	AuthZProvider.Register("permissive", &MiscAuthZPermissive{})
}
