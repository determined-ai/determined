package db

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// DoesPermissionMatch checks for the existence of a permission in a workspace.
func DoesPermissionMatch(ctx context.Context, curUserID model.UserID, workspaceID *int32,
	permissionID rbacv1.PermissionType,
) error {
	query := Bun().NewSelect().
		Table("permission_assignments").
		Join("JOIN role_assignments ra ON permission_assignments.role_id = ra.role_id").
		Join("JOIN user_group_membership ugm ON ra.group_id = ugm.group_id").
		Join("JOIN role_assignment_scopes ras ON ra.scope_id = ras.id").
		Where("ugm.user_id = ?", curUserID).
		Where("permission_assignments.permission_id = ?", permissionID)

	if workspaceID == nil {
		query = query.Where("ras.scope_workspace_id IS NULL")
	} else {
		query = query.Where("ras.scope_workspace_id = ? OR ras.scope_workspace_id IS NULL",
			*workspaceID)
	}

	exists, err := query.Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return authz.PermissionDeniedError{RequiredPermissions: []rbacv1.PermissionType{permissionID}}
}

// DoesPermissionExist checks for the existence of a permission in any workspace.
func DoesPermissionExist(ctx context.Context, curUserID model.UserID,
	permissionID rbacv1.PermissionType,
) error {
	exists, err := Bun().NewSelect().
		Table("permission_assignments").
		Join("JOIN role_assignments ra ON permission_assignments.role_id = ra.role_id").
		Join("JOIN user_group_membership ugm ON ra.group_id = ugm.group_id").
		Join("JOIN role_assignment_scopes ras ON ra.scope_id = ras.id").
		Where("ugm.user_id = ?", curUserID).
		Where("permission_assignments.permission_id = ?", permissionID).Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return authz.PermissionDeniedError{RequiredPermissions: []rbacv1.PermissionType{permissionID}}
}
