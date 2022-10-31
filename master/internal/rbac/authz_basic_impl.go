package rbac

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// RBACAuthZBasic is basic OSS controls.
type RBACAuthZBasic struct{}

// CanGetRoles always returns nil.
func (a *RBACAuthZBasic) CanGetRoles(ctx context.Context, curUser model.User,
	roleIDs []int32,
) error {
	return nil
}

// FilterRolesQuery always returns the original query and a nil error.
func (a *RBACAuthZBasic) FilterRolesQuery(ctx context.Context, curUser model.User,
	query *bun.SelectQuery) (
	*bun.SelectQuery, error,
) {
	return query, nil
}

// CanGetUserRoles always returns nil.
func (a *RBACAuthZBasic) CanGetUserRoles(ctx context.Context, curUser model.User,
	userID int32,
) error {
	return nil
}

// CanGetGroupRoles always returns nil.
func (a *RBACAuthZBasic) CanGetGroupRoles(ctx context.Context, curUser model.User,
	groupID int32,
) error {
	return nil
}

// CanSearchScope always returns nil.
func (a *RBACAuthZBasic) CanSearchScope(ctx context.Context, curUser model.User,
	workspaceID *int32,
) error {
	return nil
}

// CanGetWorkspaceMembership always returns true and a nil error.
func (a *RBACAuthZBasic) CanGetWorkspaceMembership(
	ctx context.Context, curUser model.User, workspaceID int32,
) (bool, error) {
	return true, nil
}

// CanAssignRoles returns nil if a user has admin privileges.
func (a *RBACAuthZBasic) CanAssignRoles(
	ctx context.Context,
	curUser model.User,
	groupRoleAssignments []*rbacv1.GroupRoleAssignment,
	userRoleAssignments []*rbacv1.UserRoleAssignment,
) error {
	if curUser.Admin {
		return nil
	}
	return authz.PermissionDeniedError{}
}

// CanRemoveRoles always returns nil.
func (a *RBACAuthZBasic) CanRemoveRoles(
	ctx context.Context,
	curUser model.User,
	groupRoleAssignments []*rbacv1.GroupRoleAssignment,
	userRoleAssignments []*rbacv1.UserRoleAssignment,
) error {
	return a.CanAssignRoles(ctx, curUser, groupRoleAssignments, userRoleAssignments)
}

func init() {
	AuthZProvider.Register("basic", &RBACAuthZBasic{})
}
