package command

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// NSCAuthZ describes authz methods for Notebooks, Shells, and Commands.
type NSCAuthZ interface {
	// NSC functions
	// GET /api/v1/NSCs/:nsc_id
	// GET /tasks
	CanGetNSC(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error

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
	// POST /api/v1/tensorboards/:tb_id/set_priority
	CanSetNSCsPriority(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID, priority int,
	) error

	// GET /api/v1/NSCs
	AccessibleScopes(
		ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
	) (model.AccessScopeSet, error)

	FilterTensorboards(
		ctx context.Context, curUser model.User, requestedScope model.AccessScopeID,
		tensorboards []*tensorboardv1.Tensorboard,
	) ([]*tensorboardv1.Tensorboard, error)

	// Tensorboard functions
	// GET /api/v1/tensorboards/:tb_id
	CanGetTensorboard(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
		experimentIDs []int32, trialIDs []int32,
	) error

	// POST /api/v1/tensorboards/:tb_id/kill
	CanTerminateTensorboard(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error
}

// AuthZProvider is the authz registry for Notebooks, Shells, and Commands.
var AuthZProvider authz.AuthZProviderType[NSCAuthZ]
