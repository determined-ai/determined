package command

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// NSCAuthZ describes authz methods for Notebooks, Shells, and Commands.
type NSCAuthZ interface {
	// GET /api/v1/NSCs/:nsc_id
	// GET /tasks
	CanGetNSC(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) (canGetNsc bool, serverError error)

	// GET /api/v1/NSCs
	FilterNSCWorkspaces( // DELETEME.
		ctx context.Context, curUser model.User, workspaceSet model.AccessScopeSet,
	) (model.AccessScopeSet, error)

	// GET /api/v1/NSCs
	AccessibleScopes(
		ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
	) (*model.AccessScopeSet, error)

	// GET /api/v1/tasks/count
	CanGetActiveTasksCount(ctx context.Context, curUser model.User) error

	// POST /api/v1/NSCs/:nsc_id/kill
	CanTerminateNSC(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error

	// POST /api/v1/NSCs
	CanCreateNSC(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error

	// PATCH /NSCs/:nsc_id
	CanSetNSCsPriority(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, priority int,
	) error
}

// AuthZProvider is the authz registry for Notebooks, Shells, and Commands.
var AuthZProvider authz.AuthZProviderType[NSCAuthZ]
