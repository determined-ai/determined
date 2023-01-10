package command

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// NSCAuthZBasic is basic OSS controls.
type NSCAuthZBasic struct{}

// CanGetNSC returns a nil error.
func (a *NSCAuthZBasic) CanGetNSC(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
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

// AccessibleScopes returns the set of scopes that the user should be limited to.
func (a *NSCAuthZBasic) AccessibleScopes(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	var ids []int
	returnScope := model.AccessScopeSet{requestedScope: true}

	if requestedScope == 0 {
		err := db.Bun().NewSelect().Table("workspaces").Column("id").Scan(ctx, &ids)
		if err != nil {
			return nil, err
		}

		for _, id := range ids {
			returnScope[model.AccessScopeID(id)] = true
		}

		return returnScope, nil
	}
	return returnScope, nil
}

// AccessibleScopesTB returns the set of scopes of tensorboards that the user should be limited to.
func (a *NSCAuthZBasic) AccessibleScopesTB(ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
) (model.AccessScopeSet, error) {
	return a.AccessibleScopes(ctx, curUser, requestedScope)
}

// CanGetTensorboard returns true and nil error unless the developer master config option
// security.authz._strict_ntsc_enabled is true then it returns a boolean if the user is
// an admin or if the user owns the tensorboard and a nil error.
func (a *NSCAuthZBasic) CanGetTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) (canGetTensorboards bool, serverError error) {
	return true, nil
}

// CanTerminateTensorboard always returns nil.
func (a *NSCAuthZBasic) CanTerminateTensorboard(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &NSCAuthZBasic{})
}
