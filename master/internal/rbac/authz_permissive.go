package rbac

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// RBACAuthZPermissive is the permission implementation.
type RBACAuthZPermissive struct{}

// CanGetRoles calls RBAC authz but enforces basic authz.
func (p *RBACAuthZPermissive) CanGetRoles(
	ctx context.Context, curUser model.User, roleIDs []int32,
) error {
	_ = (&RBACAuthZRBAC{}).CanGetRoles(ctx, curUser, roleIDs)
	return (&RBACAuthZBasic{}).CanGetRoles(ctx, curUser, roleIDs)
}

// FilterRolesQuery calls RBAC authz but enforces basic authz.
func (p *RBACAuthZPermissive) FilterRolesQuery(
	ctx context.Context, curUser model.User, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	_, _ = (&RBACAuthZRBAC{}).FilterRolesQuery(ctx, curUser, query)
	return (&RBACAuthZBasic{}).FilterRolesQuery(ctx, curUser, query)
}

// CanGetUserRoles calls RBAC authz but enforces basic authz.
func (p *RBACAuthZPermissive) CanGetUserRoles(
	ctx context.Context, curUser model.User, userID int32,
) error {
	_ = (&RBACAuthZRBAC{}).CanGetUserRoles(ctx, curUser, userID)
	return (&RBACAuthZBasic{}).CanGetUserRoles(ctx, curUser, userID)
}

// CanGetGroupRoles calls RBAC authz but enforces basic authz.
func (p *RBACAuthZPermissive) CanGetGroupRoles(
	ctx context.Context, curUser model.User, groupID int32,
) error {
	_ = (&RBACAuthZRBAC{}).CanGetGroupRoles(ctx, curUser, groupID)
	return (&RBACAuthZBasic{}).CanGetGroupRoles(ctx, curUser, groupID)
}

// CanSearchScope calls RBAC authz but enforces basic authz.
func (p *RBACAuthZPermissive) CanSearchScope(
	ctx context.Context, curUser model.User, workspaceID *int32,
) error {
	_ = (&RBACAuthZRBAC{}).CanSearchScope(ctx, curUser, workspaceID)
	return (&RBACAuthZBasic{}).CanSearchScope(ctx, curUser, workspaceID)
}

// CanGetWorkspaceMembership calls RBAC authz but enforces basic authz.
func (p *RBACAuthZPermissive) CanGetWorkspaceMembership(
	ctx context.Context, curUser model.User, workspaceID int32,
) (bool, error) {
	_, _ = (&RBACAuthZRBAC{}).CanGetWorkspaceMembership(ctx, curUser, workspaceID)
	return (&RBACAuthZBasic{}).CanGetWorkspaceMembership(ctx, curUser, workspaceID)
}

// CanAssignRoles calls RBAC authz but enforces basic authz.
func (p *RBACAuthZPermissive) CanAssignRoles(
	ctx context.Context, curUser model.User, groupRoleAssignments []*rbacv1.GroupRoleAssignment,
	userRoleAssignments []*rbacv1.UserRoleAssignment,
) error {
	_ = (&RBACAuthZRBAC{}).CanAssignRoles(ctx, curUser, groupRoleAssignments, userRoleAssignments)
	return (&RBACAuthZBasic{}).CanAssignRoles(ctx, curUser, groupRoleAssignments, userRoleAssignments)
}

// CanRemoveRoles calls RBAC authz but enforces basic authz.
func (p *RBACAuthZPermissive) CanRemoveRoles(
	ctx context.Context, curUser model.User, groupRoleAssignments []*rbacv1.GroupRoleAssignment,
	userRoleAssignments []*rbacv1.UserRoleAssignment,
) error {
	_ = (&RBACAuthZRBAC{}).CanRemoveRoles(ctx, curUser, groupRoleAssignments, userRoleAssignments)
	return (&RBACAuthZBasic{}).CanRemoveRoles(ctx, curUser, groupRoleAssignments, userRoleAssignments)
}

func init() {
	AuthZProvider.Register("permissive", &RBACAuthZPermissive{})
}
