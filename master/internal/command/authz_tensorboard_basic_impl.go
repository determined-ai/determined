package command

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TensorboardAuthZBasic is basic OSS controls.
type TensorboardAuthZBasic struct{}

// CanGetNSC returns true and nil error unless the developer master config option
// security.authz._strict_ntsc_enabled is true then it returns a boolean if the user is
// an admin or if the user owns the task and a nil error.
func (t *TensorboardAuthZBasic) CanGetTensorboard(
	ctx context.Context, curUser *model.User, tb *tensorboardv1.Tensorboard,
) (canGetTensorboards bool, serverError error) {
	if !config.GetMasterConfig().Security.AuthZ.StrictNTSCEnabled {
		return true, nil
	}

	return curUser.Admin || int32(curUser.ID) == tb.UserId, nil
}

// FilterTensorboards always returns the same list.
func (t *TensorboardAuthZBasic) FilterTensorboards(
	ctx context.Context, curUser *model.User, tensorboards []*tensorboardv1.Tensorboard,
) ([]*tensorboardv1.Tensorboard, error) {
	return tensorboards, nil
}

// CanTerminateTensorboard always returns nil.
func (t *TensorboardAuthZBasic) CanTerminateTensorboard(
	ctx context.Context, curUser *model.User, tb *tensorboardv1.Tensorboard,
) error {
	return nil
}

// CanSetTensorboardPriority always returns nil.
func (t *TensorboardAuthZBasic) CanSetTensorboardPriority(
	ctx context.Context, curUser *model.User, tb *tensorboardv1.Tensorboard,
) error {
	return nil
}

func init() {
	TbAuthZProvider.Register("basic", &TensorboardAuthZBasic{})
}
