package command

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// ResourceContext is a context that contains information about an individual NTSC resource.
type ResourceContext struct {
	Spec    *tasks.GenericCommandSpec
	OwnerID model.UserID
}

// AccessContext is the request context for checking access to a resource.
type AccessContext struct {
	Ctx context.Context
	// User is the user making the request.
	User model.User
	// Workspace is the workspace the user is making the request in.
	Workspace *model.Workspace
}

// CommandAuthZ describes authz methods for commands.
// DISCUSS should we start moving to using NTSC in code, anything other than "command" is more clear IMO.
type CommandAuthZ interface {
	// TODO request ownerID for some checks.
	// GET /api/v1/commands/:cmd_id
	// GET /tasks
	CanGetCommand(
		aCtx AccessContext, rCtx ResourceContext,
	) (canGetCmd bool, serverError error)

	CanAccessNTSCTask(ctx context.Context, curUser model.User, ownerID model.UserID) (
		canView bool, serverError error)

	// GET /api/v1/tasks/count
	// TODO(nick) move this when we add an AuthZ for notebooks.
	CanGetActiveTasksCount(ctx context.Context, curUser model.User) error

	// GET /api/v1/commands
	// "workspace" being nil indicates getting commands from all workspaces.
	FilterCommandsQuery(
		aCtx AccessContext, query *bun.SelectQuery,
	) (*bun.SelectQuery, error)

	// POST /api/v1/commands/:cmd_id/kill
	CanTerminateCommand(
		aCtx AccessContext, rCtx ResourceContext,
	) error

	// POST /api/v1/commands
	CanCreateCommand(
		aCtx AccessContext, rCtx ResourceContext,
	) error

	// PATCH /commands/:cmd_id
	CanSetCommandsMaxSlots(
		aCtx AccessContext, rCtx ResourceContext, slots int,
	) error
	CanSetCommandsWeight(
		aCtx AccessContext, rCtx ResourceContext, weight float64,
	) error
	CanSetCommandsPriority(
		aCtx AccessContext, rCtx ResourceContext, priority int,
	) error
}

// AuthZProvider is the authz registry for commands.
var AuthZProvider authz.AuthZProviderType[CommandAuthZ]
