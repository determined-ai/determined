package command

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// NSCAuthZRBAC is the RBAC implementation of the NSCAuthZ interface.
type NSCAuthZRBAC struct{}

// CanGetNSC returns true and nil error unless the developer master config option
// security.authz._strict_ntsc_enabled is true then it returns a boolean if the user is
// an admin or if the user owns the task and a nil error.
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

// AccessibleScopesTB returns the set of scopes that the user should be limited to.
func (a *NSCAuthZRBAC) AccessibleScopesTB(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	return (&NSCAuthZBasic{}).AccessibleScopesTB(ctx, curUser, requestedScope)
}

// CanGetTensorboard always returns true and nil error.
func (a *NSCAuthZRBAC) CanGetTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) (canGetTensorboards bool, serverError error) {
	return (&NSCAuthZBasic{}).CanGetTensorboard(ctx, curUser, workspaceID)
}

// CanTerminateTensorboard always returns nil.
func (a *NSCAuthZRBAC) CanTerminateTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return (&NSCAuthZBasic{}).CanTerminateTensorboard(ctx, curUser, workspaceID)
}

func init() {
	AuthZProvider.Register("rbac", &NSCAuthZRBAC{})
}
