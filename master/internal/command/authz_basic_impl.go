package command

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// NSCAuthZBasic is basic OSS controls.
type NSCAuthZBasic struct{}

// CanGetNSC returns a nil error.
func (a *NSCAuthZBasic) CanGetNSC(
	ctx context.Context, curUser model.User, ownerID model.UserID, workspaceID model.AccessScopeID,
) (canGetCmd bool, serverError error) {
	return true, nil
}

// CanGetActiveTasksCount always returns a nil error.
func (a *NSCAuthZBasic) CanGetActiveTasksCount(ctx context.Context, curUser model.User) error {
	return nil
}

// CanTerminateNSC always returns a nil error.
func (a *NSCAuthZBasic) CanTerminateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return nil
}

// CanCreateNSC always returns a nil error.
func (a *NSCAuthZBasic) CanCreateNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return nil
}

// CanSetNSCsPriority always returns a nil error.
func (a *NSCAuthZBasic) CanSetNSCsPriority(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, priority int,
) error {
	return nil
}

func (a *NSCAuthZBasic) AccessibleScopes(
	ctx context.Context, curUser model.User, scopes map[model.AccessScopeID]bool,
) (map[model.AccessScopeID]bool, error) {
	return scopes, nil
}

// CanGetTensorboard returns true and nil error unless the developer master config option
// security.authz._strict_ntsc_enabled is true then it returns a boolean if the user is
// an admin or if the user owns the tensorboard and a nil error.
func (a *NSCAuthZBasic) CanGetTensorboard(
	ctx context.Context, curUser model.User, ownerID model.UserID, workspaceID model.AccessScopeID,
) (canGetTensorboards bool, serverError error) {
	return true, nil
}

// CanTerminateTensorboard always returns nil.
func (a *NSCAuthZBasic) CanTerminateTensorboard(
	ctx context.Context, curUser model.User, tb *tensorboardv1.Tensorboard,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &NSCAuthZBasic{})
}
