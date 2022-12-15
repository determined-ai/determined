package command

import (
	"context"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// NSCAuthZ describes authz methods for Notebooks, Shells, and Commands.
type TensorboardAuthZ interface {
	// GET /api/v1/tensorboards
	// GET /api/v1/tensorboards/:tb_id
	CanGetTensorboards(
		ctx context.Context, curUser model.User, tensorboards []*tensorboardv1.Tensorboard,
	) (canGetTensorboards bool, serverError error)

	// POST /api/v1/tensorboards/:tb_id/kill
	CanTerminateTensorboard(
		ctx context.Context, curUser model.User, workspaceID model.AccessScopeID,
	) error

	// POST /api/v1/tensorboards/:tb_id/set_priority
	CanSetTensorboardPriority(
		ctx context.Context, curUser model.User, c *tasks.GenericCommandSpec, priority int,
	) error
}

// TbAuthZProvider is the authz registry for Tensorboards.
var TbAuthZProvider authz.AuthZProviderType[TensorboardAuthZ]
