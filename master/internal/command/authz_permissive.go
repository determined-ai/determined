package command

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"

	"github.com/determined-ai/determined/master/pkg/model"
)

// NSCAuthZPermissive is permissive implementation of the NSCAuthZ interface.
type NSCAuthZPermissive struct{}

// CanGetNSC returns true and nil error unless the developer master config option
// security.authz._strict_ntsc_enabled is true then it returns a boolean if the user is
// an admin or if the user owns the task and a nil error.
func (a *NSCAuthZPermissive) CanGetNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) (canGetCmd bool, err error) {
	_, _ = (&NSCAuthZRBAC{}).CanGetNSC(ctx, curUser, workspaceID)
	return (&NSCAuthZBasic{}).CanGetNSC(ctx, curUser, workspaceID)
}

// CanGetActiveTasksCount always returns a nil error.
func (a *NSCAuthZPermissive) CanGetActiveTasksCount(ctx context.Context, curUser model.User) error {
	_ = (&NSCAuthZRBAC{}).CanGetActiveTasksCount(ctx, curUser)
	return (&NSCAuthZBasic{}).CanGetActiveTasksCount(ctx, curUser)
}

// CanTerminateNSC always returns a nil error.
func (a *NSCAuthZPermissive) CanTerminateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	_ = (&NSCAuthZRBAC{}).CanTerminateNSC(ctx, curUser, workspaceID)
	return (&NSCAuthZBasic{}).CanTerminateNSC(ctx, curUser, workspaceID)
}

// CanCreateNSC always returns a nil error.
func (a *NSCAuthZPermissive) CanCreateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	_ = (&NSCAuthZRBAC{}).CanCreateNSC(ctx, curUser, workspaceID)
	return (&NSCAuthZBasic{}).CanCreateNSC(ctx, curUser, workspaceID)
}

// CanSetNSCsPriority always returns a nil error.
func (a *NSCAuthZPermissive) CanSetNSCsPriority(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, priority int,
) error {
	_ = (&NSCAuthZRBAC{}).CanSetNSCsPriority(ctx, curUser, workspaceID, priority)
	return (&NSCAuthZBasic{}).CanSetNSCsPriority(ctx, curUser, workspaceID, priority)
}

// AccessibleScopes returns the set of scopes that the user should be limited to.
func (a *NSCAuthZPermissive) AccessibleScopes(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	_, _ = (&NSCAuthZRBAC{}).AccessibleScopes(ctx, curUser, requestedScope)
	return (&NSCAuthZBasic{}).AccessibleScopes(ctx, curUser, requestedScope)
}

// FilterTensorboards returns the tensorboards the user has access to.
func (a *NSCAuthZPermissive) FilterTensorboards(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
	tensorboards []*tensorboardv1.Tensorboard,
) ([]*tensorboardv1.Tensorboard, error) {
	_, _ = (&NSCAuthZRBAC{}).FilterTensorboards(ctx, curUser, requestedScope, tensorboards)
	return (&NSCAuthZBasic{}).FilterTensorboards(ctx, curUser, requestedScope, tensorboards)
}

// CanGetTensorboard always returns true and nil error.
func (a *NSCAuthZPermissive) CanGetTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	experimentIDs []int32, trialIDs []int32,
) (canGetTensorboards bool, serverError error) {
	_, _ = (&NSCAuthZRBAC{}).CanGetTensorboard(ctx, curUser, workspaceID, experimentIDs, trialIDs)
	return (&NSCAuthZBasic{}).CanGetTensorboard(ctx, curUser, workspaceID, experimentIDs, trialIDs)
}

// CanTerminateTensorboard always returns nil.
func (a *NSCAuthZPermissive) CanTerminateTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	experimentIDs []int32, trialIDs []int32,
) error {
	_ = (&NSCAuthZRBAC{}).CanTerminateTensorboard(ctx, curUser, workspaceID, experimentIDs,
		trialIDs)
	return (&NSCAuthZBasic{}).CanTerminateTensorboard(ctx, curUser, workspaceID, experimentIDs,
		trialIDs)
}

func init() {
	AuthZProvider.Register("permissive", &NSCAuthZPermissive{})
}
