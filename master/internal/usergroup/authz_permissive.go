package usergroup

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserGroupAuthZPermissive is the permission implementation.
type UserGroupAuthZPermissive struct{}

// CanGetGroup calls RBAC authz but enforces basic authz.
func (p *UserGroupAuthZPermissive) CanGetGroup(
	ctx context.Context, curUser model.User, gid int,
) (bool, error) {
	_, _ = (&UserGroupAuthZRBAC{}).CanGetGroup(ctx, curUser, gid)
	return (&UserGroupAuthZBasic{}).CanGetGroup(ctx, curUser, gid)
}

// FilterGroupsList calls RBAC authz but enforces basic authz.
func (p *UserGroupAuthZPermissive) FilterGroupsList(
	ctx context.Context, curUser model.User, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	_, _ = (&UserGroupAuthZRBAC{}).FilterGroupsList(ctx, curUser, query)
	return (&UserGroupAuthZBasic{}).FilterGroupsList(ctx, curUser, query)
}

// CanUpdateGroups calls RBAC authz but enforces basic authz.
func (p *UserGroupAuthZPermissive) CanUpdateGroups(
	ctx context.Context, curUser model.User,
) error {
	_ = (&UserGroupAuthZRBAC{}).CanUpdateGroups(ctx, curUser)
	return (&UserGroupAuthZBasic{}).CanUpdateGroups(ctx, curUser)
}

func init() {
	AuthZProvider.Register("permissive", &UserGroupAuthZPermissive{})
}
