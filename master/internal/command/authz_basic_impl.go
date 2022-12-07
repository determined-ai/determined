package command

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// CommandAuthZBasic is basic OSS controls.
type CommandAuthZBasic struct{}

// CanGetCommand returns true and nil error unless the developer master config option
// security.authz._strict_ntsc_enabled is true then it returns a boolean if the user is
// an admin or if the user owns the task and a nil error.
func (a *CommandAuthZBasic) CanGetCommand(
	ctx context.Context, curUser model.User, ownerID model.UserID, workspaceID model.AccessScopeID, jobType model.JobType,
) (canGetCmd bool, serverError error) {
	if !config.GetMasterConfig().Security.AuthZ.StrictNTSCEnabled {
		return true, nil
	}
	return curUser.Admin || curUser.ID == ownerID, nil
}

// CanGetActiveTasksCount always returns a nil error.
func (a *CommandAuthZBasic) CanGetActiveTasksCount(ctx context.Context, curUser model.User) error {
	return nil
}

// FilterCommandsQuery returns the query unmodified and a nil error.
func (a *CommandAuthZBasic) FilterCommandsQuery(
	ctx context.Context, curUser model.User, workspace *model.Workspace, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// CanTerminateCommand always returns a nil error.
func (a *CommandAuthZBasic) CanTerminateCommand(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, jobType model.JobType,
) error {
	return nil
}

// CanCreateCommand always returns a nil error.
func (a *CommandAuthZBasic) CanCreateCommand(
	ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, jobType model.JobType,
) error {
	return nil
}

// CanSetCommandsMaxSlots always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsMaxSlots(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, slots int,
) error {
	return nil
}

// CanSetCommandsWeight always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsWeight(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, weight float64,
) error {
	return nil
}

// CanSetCommandsPriority always returns a nil error.
func (a *CommandAuthZBasic) CanSetCommandsPriority(
	ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, priority int,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &CommandAuthZBasic{})
}
