package command

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// NSCAuthZ describes authz methods for Notebooks, Shells, and Commands.
type NSCAuthZ interface {
	// GET /api/v1/commands/:nsc_id
	// GET /tasks
	CanGetNSC(
		ctx context.Context, curUser model.User, ownerID model.UserID, workspaceID model.AccessScopeID,
	) (canGetNsc bool, serverError error)

	// GET /api/v1/tasks/count
	CanGetActiveTasksCount(ctx context.Context, curUser model.User) error

	// POST /api/v1/commands/:nsc_id/kill
	CanTerminateNSC(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error

	// POST /api/v1/commands
	CanCreateNSC(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error

	// PATCH /commands/:nsc_id
	CanSetNSCsPriority(
		ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, priority int,
	) error
}

// AuthZProvider is the authz registry for Notebooks, Shells, and Commands.
var AuthZProvider authz.AuthZProviderType[NSCAuthZ]
