package command

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// CommandAuthZ describes authz methods for commands.
// TODO: rename to Command to NSC.
type CommandAuthZ interface {
	// GET /api/v1/commands/:cmd_id
	// GET /tasks
	CanGetCommand(
		ctx context.Context, curUser model.User, ownerID model.UserID, workspaceID model.AccessScopeID,
	) (canGetCmd bool, serverError error)

	// GET /api/v1/tasks/count
	CanGetActiveTasksCount(ctx context.Context, curUser model.User) error

	// POST /api/v1/commands/:cmd_id/kill
	CanTerminateCommand(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error

	// POST /api/v1/commands
	CanCreateCommand(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error

	// PATCH /commands/:cmd_id
	CanSetCommandsPriority(
		ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, priority int,
	) error
}

// AuthZProvider is the authz registry for commands.
var AuthZProvider authz.AuthZProviderType[CommandAuthZ]
