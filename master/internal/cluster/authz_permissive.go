package cluster

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// MiscAuthZPermissive is permissive implementation of the MiscAuthZ interface.
type MiscAuthZPermissive struct{}

// CanUpdateAgents calls the RBAC implementation but always allows access.
func (a *MiscAuthZPermissive) CanUpdateAgents(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	_, _ = (&MiscAuthZRBAC{}).CanUpdateAgents(ctx, curUser)
	return (&MiscAuthZBasic{}).CanUpdateAgents(ctx, curUser)
}

// CanGetMasterLogs returns calls the RBAC implementation but always allows access.
func (a *MiscAuthZPermissive) CanGetMasterLogs(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	_, _ = (&MiscAuthZRBAC{}).CanGetMasterLogs(ctx, curUser)
	return (&MiscAuthZBasic{}).CanGetMasterLogs(ctx, curUser)
}

// CanGetUsageDetails calls the RBAC implementation but always allows access.
func (a *MiscAuthZPermissive) CanGetUsageDetails(
	ctx context.Context, curUser *model.User,
) (permErr error, err error) {
	_, _ = (&MiscAuthZRBAC{}).CanGetUsageDetails(ctx, curUser)
	return (&MiscAuthZBasic{}).CanGetUsageDetails(ctx, curUser)
}

func init() {
	AuthZProvider.Register("permissive", &MiscAuthZPermissive{})
}
