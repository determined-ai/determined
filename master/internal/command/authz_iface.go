package command

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// CommandAuthZ describes authz methods for commands.
type CommandAuthZ interface {
	// GET /api/v1/commands/:cmd_id
	// GET /tasks
	CanGetCommand(
		ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec,
	) (canGetCmd bool, serverError error)

	// GET /api/v1/commands
	// "workspace" being nil indicates getting commands from all workspaces.
	FilterCommandsQuery(
		ctx context.Context, curUser model.User, workspace *model.Workspace, query *bun.SelectQuery,
	) (*bun.SelectQuery, error)

	// POST /api/v1/commands
	// POST /api/v1/commands/:cmd_id/kill
	// POST /api/v1/commands/:cmd_id/cancel
	CanEditCommand(ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec) error

	// PATCH /api/v1/commands/:cmd_id/
	CanEditCommandsMetadata(ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec) error

	// POST /api/v1/commands
	CanCreateCommand(
		ctx context.Context, curUser model.User, workspace *model.Workspace, c *tasks.GenericCommandSpec,
	) error

	// PATCH /commands/:cmd_id
	CanSetCommandsMaxSlots(
		ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, slots int,
	) error
	CanSetCommandsWeight(
		ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, weight float64,
	) error
	CanSetCommandsPriority(
		ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, priority int,
	) error
}

// AuthZProvider is the authz registry for commands.
var AuthZProvider authz.AuthZProviderType[CommandAuthZ]
