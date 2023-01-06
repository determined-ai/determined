package command

import (
	"context"

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

// AccessibleScopes returns the set of scopes that the user should be limited to.
func (a *NSCAuthZBasic) AccessibleScopes(
	ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
) (*model.AccessScopeSet, error) {
	if requestedScope == 0 {
		return nil, nil
	}
	return &model.AccessScopeSet{requestedScope: true}, nil
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

func init() {
	AuthZProvider.Register("basic", &NSCAuthZBasic{})
}
