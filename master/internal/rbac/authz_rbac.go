package rbac

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// RBACAuthZRBAC is RBAC controls.
type RBACAuthZRBAC struct{}

// CanGetRoles checks if a user can get all the roles specified.
func (a *RBACAuthZRBAC) CanGetRoles(ctx context.Context, curUser model.User,
	roleIDs []int32,
) error {
	err := db.DoPermissionsExist(ctx, curUser.ID, rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_ROLES)
	if err == nil {
		return nil
	} else if _, ok := err.(authz.PermissionDeniedError); !ok {
		return err
	}

	roles, err := GetAssignedRoles(ctx, curUser.ID)
	if err != nil {
		return err
	}

	rolesMap := make(map[int32]bool, len(roles))
	for _, roleID := range roles {
		rolesMap[roleID] = true
	}

	for _, roleID := range roleIDs {
		if _, ok := rolesMap[roleID]; !ok {
			return authz.PermissionDeniedError{
				RequiredPermissions: []rbacv1.PermissionType{
					rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES,
					rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_ROLES,
				},
			}
		}
	}

	return nil
}

// FilterRolesQuery filters for roles that the user has access to.
func (a *RBACAuthZRBAC) FilterRolesQuery(ctx context.Context, curUser model.User,
	query *bun.SelectQuery) (
	*bun.SelectQuery, error,
) {
	err := db.DoPermissionsExist(ctx, curUser.ID, rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_ROLES)
	if err == nil {
		return query, nil
	} else if _, ok := err.(authz.PermissionDeniedError); !ok {
		return query, err
	}

	roles, err := GetAssignedRoles(ctx, curUser.ID)
	if err != nil {
		return query, err
	}
	if len(roles) == 0 {
		return query.Where("false"), nil
	}

	return query.Where("id IN (?)", bun.In(roles)), nil
}

// CanGetUserRoles checks if the user can access a specific user's roles.
func (a *RBACAuthZRBAC) CanGetUserRoles(ctx context.Context, curUser model.User,
	userID int32,
) error {
	if int32(curUser.ID) == userID {
		return nil
	}
	return db.DoPermissionsExist(ctx, curUser.ID, rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES)
}

// CanGetGroupRoles checks if the user can access a specific group's roles.
func (a *RBACAuthZRBAC) CanGetGroupRoles(ctx context.Context, curUser model.User,
	groupID int32,
) error {
	err := db.DoPermissionsExist(ctx, curUser.ID, rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES)
	if err == nil {
		return nil
	} else if _, ok := err.(authz.PermissionDeniedError); !ok {
		return err
	}

	query := db.Bun().NewSelect().
		Table("permission_assignments").
		Join("JOIN role_assignments ra ON permission_assignments.role_id = ra.role_id").
		Join("JOIN user_group_membership ugm ON ra.group_id = ugm.group_id").
		Join("JOIN role_assignment_scopes ras ON ra.scope_id = ras.id").
		Where("ugm.user_id = ?", curUser.ID).
		Where("ra.group_id = ?", groupID)

	exists, err := query.Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return authz.PermissionDeniedError{RequiredPermissions: []rbacv1.PermissionType{
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_GROUP,
			rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES,
		}}
	}
	return nil
}

// CanSearchScope checks if a user can search for roles on a specific scope.
func (a *RBACAuthZRBAC) CanSearchScope(ctx context.Context, curUser model.User,
	workspaceID *int32,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_WORKSPACE)
}

// CanGetWorkspaceMembership checks if a user can get membership on a workspace.
func (a *RBACAuthZRBAC) CanGetWorkspaceMembership(
	ctx context.Context, curUser model.User, workspaceID int32,
) (bool, error) {
	if err := db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_WORKSPACE); err != nil {
		if _, ok := err.(authz.PermissionDeniedError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CanAssignRoles checks if a user can assign roles.
func (a *RBACAuthZRBAC) CanAssignRoles(
	ctx context.Context,
	curUser model.User,
	groupRoleAssignments []*rbacv1.GroupRoleAssignment,
	userRoleAssignments []*rbacv1.UserRoleAssignment,
) error {
	var workspaces []int32

	for _, v := range groupRoleAssignments {
		if v.RoleAssignment.ScopeWorkspaceId != nil {
			workspaces = append(workspaces, *v.RoleAssignment.ScopeWorkspaceId)
		} else {
			return db.DoesPermissionMatch(ctx, curUser.ID, nil,
				rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES)
		}
	}

	for _, v := range userRoleAssignments {
		if v.RoleAssignment.ScopeWorkspaceId != nil {
			workspaces = append(workspaces, *v.RoleAssignment.ScopeWorkspaceId)
		} else {
			return db.DoesPermissionMatch(ctx, curUser.ID, nil,
				rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES)
		}
	}

	return db.DoesPermissionMatchAll(ctx, curUser.ID,
		rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES, workspaces...)
}

// CanRemoveRoles checks if a user can remove roles.
func (a *RBACAuthZRBAC) CanRemoveRoles(
	ctx context.Context,
	curUser model.User,
	groupRoleAssignments []*rbacv1.GroupRoleAssignment,
	userRoleAssignments []*rbacv1.UserRoleAssignment,
) error {
	return a.CanAssignRoles(ctx, curUser, groupRoleAssignments, userRoleAssignments)
}

func init() {
	AuthZProvider.Register("rbac", &RBACAuthZRBAC{})
}
