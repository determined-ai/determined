package command

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// NSCAuthZRBAC is the RBAC implementation of the NSCAuthZ interface.
type NSCAuthZRBAC struct{}

// CanGetNSC always returns a nil error.
func (a *NSCAuthZRBAC) CanGetNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) (canGetCmd bool, err error) {
	return (&NSCAuthZBasic{}).CanGetNSC(ctx, curUser, workspaceID)
}

// CanGetActiveTasksCount always returns a nil error.
func (a *NSCAuthZRBAC) CanGetActiveTasksCount(ctx context.Context, curUser model.User) error {
	return (&NSCAuthZBasic{}).CanGetActiveTasksCount(ctx, curUser)
}

// CanTerminateNSC always returns a nil error.
func (a *NSCAuthZRBAC) CanTerminateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return (&NSCAuthZBasic{}).CanTerminateNSC(ctx, curUser, workspaceID)
}

// CanCreateNSC always returns a nil error.
func (a *NSCAuthZRBAC) CanCreateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return (&NSCAuthZBasic{}).CanCreateNSC(ctx, curUser, workspaceID)
}

// CanSetNSCsPriority always returns a nil error.
func (a *NSCAuthZRBAC) CanSetNSCsPriority(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, priority int,
) error {
	return (&NSCAuthZBasic{}).CanSetNSCsPriority(ctx, curUser, workspaceID, priority)
}

// AccessibleScopes returns the set of scopes that the user should be limited to.
func (a *NSCAuthZRBAC) AccessibleScopes(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	return (&NSCAuthZBasic{}).AccessibleScopes(ctx, curUser, requestedScope)
}

// FilterTensorboards returns the tensorboards the user has access to.
func (a *NSCAuthZRBAC) FilterTensorboards(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
	tensorboards []*tensorboardv1.Tensorboard,
) ([]*tensorboardv1.Tensorboard, error) {
	return (&NSCAuthZBasic{}).FilterTensorboards(ctx, curUser, requestedScope, tensorboards)
}

// CanGetTensorboard always returns true and nil error.
func (a *NSCAuthZRBAC) CanGetTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	experimentIDs []int32, trialIDs []int32,
) (canGetTensorboards bool, serverError error) {
	return (&NSCAuthZBasic{}).CanGetTensorboard(ctx, curUser, workspaceID, experimentIDs, trialIDs)
}

// CanTerminateTensorboard always returns nil.
func (a *NSCAuthZRBAC) CanTerminateTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	experimentIDs []int32, trialIDs []int32,
) error {
	return (&NSCAuthZBasic{}).CanTerminateTensorboard(ctx, curUser, workspaceID, experimentIDs,
		trialIDs)
}

func init() {
	AuthZProvider.Register("rbac", &NSCAuthZRBAC{})
}
