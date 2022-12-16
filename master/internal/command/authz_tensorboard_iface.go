package command

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// NSCAuthZ describes authz methods for Notebooks, Shells, and Commands.
type TensorboardAuthZ interface {
	// GET /api/v1/tensorboards/:tb_id
	CanGetTensorboard(
		ctx context.Context, curUser *model.User, tb *tensorboardv1.Tensorboard,
	) (canGetTensorboard bool, serverError error)

	// GET /api/v1/tensorboards
	FilterTensorboards(
		ctx context.Context, curUser *model.User, tensorboards []*tensorboardv1.Tensorboard,
	) ([]*tensorboardv1.Tensorboard, error)

	// POST /api/v1/tensorboards/:tb_id/kill
	CanTerminateTensorboard(
		ctx context.Context, curUser *model.User, tb *tensorboardv1.Tensorboard,
	) error

	// POST /api/v1/tensorboards/:tb_id/set_priority
	CanSetTensorboardPriority(
		ctx context.Context, curUser *model.User, tb *tensorboardv1.Tensorboard,
	) error
}

// TbAuthZProvider is the authz registry for Tensorboards.
var TbAuthZProvider authz.AuthZProviderType[TensorboardAuthZ]
