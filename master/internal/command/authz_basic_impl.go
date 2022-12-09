package command

import (
	"context"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// NSCAuthZBasic is basic OSS controls.
type NSCAuthZBasic struct{}

// CanGetNSC returns true and nil error unless the developer master config option
// security.authz._strict_ntsc_enabled is true then it returns a boolean if the user is
// an admin or if the user owns the task and a nil error.
func (a *NSCAuthZBasic) CanGetNSC(
	ctx context.Context, curUser model.User, ownerID model.UserID, workspaceID model.AccessScopeID,
) (canGetCmd bool, serverError error) {
	if !config.GetMasterConfig().Security.AuthZ.StrictNTSCEnabled {
		return true, nil
	}
	return curUser.Admin || curUser.ID == ownerID, nil
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
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, priority int,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &NSCAuthZBasic{})
}
