package command

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// NSCAuthZ describes authz methods for Notebooks, Shells, and Commands.
type NSCAuthZ interface {
	// NSC functions
	// GET /api/v1/NSCs/:nsc_id
	// GET /tasks
	CanGetNSC(
		ctx context.Context, curUser model.User, ownerID model.UserID, workspaceID model.AccessScopeID,
	) (canGetNsc bool, serverError error)

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


	AccessibleScopes(
		ctx context.Context, curUser model.UserID, scopes []model.AccessScopeID,
	) ([]model.AccessScopeID, error)

	// Tensorboard functions
	// GET /api/v1/tensorboards/:tb_id
	CanGetTensorboard(
		ctx context.Context, curUser *model.User, ownerID model.UserID, workspaceID model.AccessScopeID,
	) (canGetTensorboard bool, serverError error)

	// GET /api/v1/tensorboards
	FilterTensorboards(
		ctx context.Context, curUser *model.User, tensorboards []*tensorboardv1.Tensorboard,
	) ([]*tensorboardv1.Tensorboard, error)

	// POST /api/v1/tensorboards/:tb_id/kill
	CanTerminateTensorboard(
		ctx context.Context, curUser *model.User, tb *tensorboardv1.Tensorboard,
	) error
}

// AuthZProvider is the authz registry for Notebooks, Shells, and Commands.
var AuthZProvider authz.AuthZProviderType[NSCAuthZ]
