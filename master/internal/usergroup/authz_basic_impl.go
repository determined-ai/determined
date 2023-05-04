package usergroup

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserGroupAuthZBasic is basic OSS controls.
type UserGroupAuthZBasic struct{}

// CanGetGroup always returns nil.
func (a *UserGroupAuthZBasic) CanGetGroup(ctx context.Context, curUser model.User, gid int) error {
	return nil
}

// FilterGroupsList returns the list it was given and a nil error.
func (a *UserGroupAuthZBasic) FilterGroupsList(ctx context.Context, curUser model.User,
	query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

// CanUpdateGroups always returns nil.
func (a *UserGroupAuthZBasic) CanUpdateGroups(ctx context.Context, curUser model.User) error {
	if curUser.Admin {
		return nil
	}
	return grpcutil.ErrPermissionDenied
}

func init() {
	AuthZProvider.Register("basic", &UserGroupAuthZBasic{})
}
