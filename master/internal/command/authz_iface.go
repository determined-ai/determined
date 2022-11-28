package command

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// CommandAuthZ describes authz methods for commands.
type CommandAuthZ interface {
	// GET /api/v1/commands/:cmd_id
	// GET /tasks
	CanGetCommand(
		ctx context.Context, curUser model.User, c *command,
	) (canGetCmd bool, serverError error)

	// DELETE /api/v1/commands/:cmd_id
	CanDeleteCommand(ctx context.Context, curUser model.User, c *command) error

	// GET /api/v1/commands
	// "workspace" being nil indicates getting commands from all workspaces.
	FilterCommandsQuery(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace, query *bun.SelectQuery,
	) (*bun.SelectQuery, error)

	// POST /api/v1/commands/:cmd_id/activate
	// POST /api/v1/commands
	// POST /api/v1/commands/:cmd_id/pause
	// POST /api/v1/commands/:cmd_id/kill
	// POST /api/v1/commands/:cmd_id/hyperparameter-importance
	// POST /api/v1/commands/:cmd_id/cancel
	CanEditCommand(ctx context.Context, curUser model.User, c *command) error

	// POST /api/v1/commands/:cmd_id/archive
	// POST /api/v1/commands/:cmd_id/unarchive
	// PATCH /api/v1/commands/:cmd_id/
	CanEditCommandsMetadata(ctx context.Context, curUser model.User, c *command) error

	// POST /api/v1/commands
	CanCreateCommand(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace, c *command,
	) error

	// PATCH /commands/:cmd_id
	CanSetCommandsMaxSlots(
		ctx context.Context, curUser model.User, c *command, slots int,
	) error
	CanSetCommandsWeight(
		ctx context.Context, curUser model.User, c *command, weight float64,
	) error
	CanSetCommandsPriority(
		ctx context.Context, curUser model.User, c *command, priority int,
	) error
}

// AuthZProvider is the authz registry for commands.
var AuthZProvider authz.AuthZProviderType[CommandAuthZ]
