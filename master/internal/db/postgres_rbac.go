package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"

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

// DoPermissionsExist checks for the existence of a permission in any workspace.
func DoPermissionsExist(ctx context.Context, curUserID model.UserID,
	permissionIDs ...rbacv1.PermissionType,
) error {
	exists, err := Bun().NewSelect().
		Table("permission_assignments").
		Join("JOIN role_assignments ra ON permission_assignments.role_id = ra.role_id").
		Join("JOIN user_group_membership ugm ON ra.group_id = ugm.group_id").
		Join("JOIN role_assignment_scopes ras ON ra.scope_id = ras.id").
		Where("ugm.user_id = ?", curUserID).
		Where("permission_assignments.permission_id IN (?)", bun.In(permissionIDs)).Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if len(permissionIDs) > 1 {
		return authz.PermissionDeniedError{RequiredPermissions: permissionIDs, OneOf: true}
	}
	return authz.PermissionDeniedError{RequiredPermissions: permissionIDs}
}

// DoesPermissionMatchAll checks for the existence of a permission in all specified workspaces.
func DoesPermissionMatchAll(ctx context.Context, curUserID model.UserID,
	permissionID rbacv1.PermissionType, workspaceIds ...int32,
) error {
	type workspaceScope struct {
		ID          int           `bun:"id,pk,autoincrement" json:"id"`
		WorkspaceID sql.NullInt32 `bun:"scope_workspace_id"  json:"workspace_id"`
	}
	var scopes []workspaceScope
	scopesMap := map[int32]bool{}

	err := Bun().NewSelect().
		TableExpr("role_assignment_scopes as ras").
		Column("scope_workspace_id").
		Join("JOIN role_assignments ra ON ra.scope_id = ras.id").
		Join("JOIN permission_assignments pa ON ra.role_id = pa.role_id").
		Join("JOIN user_group_membership ugm ON ra.group_id = ugm.group_id").
		Where("ugm.user_id = ?", curUserID).
		Where("pa.permission_id = ?", permissionID).
		Where("ras.scope_workspace_id IS NULL OR ras.scope_workspace_id IN (?)",
			bun.In(workspaceIds)).
		Scan(ctx, &scopes)
	if err != nil {
		return err
	}

	for _, v := range scopes {
		if !v.WorkspaceID.Valid {
			return nil
		}
		scopesMap[v.WorkspaceID.Int32] = true
	}

	for _, v := range workspaceIds {
		if ok := scopesMap[v]; !ok {
			return authz.PermissionDeniedError{
				RequiredPermissions: []rbacv1.PermissionType{permissionID},
			}
		}
	}
	return nil
}

// GetNonGlobalWorkspacesWithPermission returns all workspaces the user has permissionID on.
func GetNonGlobalWorkspacesWithPermission(ctx context.Context, curUserID model.UserID,
	permissionID rbacv1.PermissionType,
) ([]int, error) {
	var workspaces []int

	err := Bun().NewSelect().
		TableExpr("role_assignment_scopes as ras").
		Column("scope_workspace_id").
		Join("JOIN role_assignments ra ON ra.scope_id = ras.id").
		Join("JOIN permission_assignments pa ON ra.role_id = pa.role_id").
		Join("JOIN user_group_membership ugm ON ra.group_id = ugm.group_id").
		Where("ugm.user_id = ?", curUserID).
		Where("pa.permission_id = ?", permissionID).
		Scan(ctx, &workspaces)
	if err != nil {
		return workspaces, err
	}

	return workspaces, nil
}

// ExperimentIDsToWorkspaceIDs returns a slice of workspaces that the given experiments belong to.
func ExperimentIDsToWorkspaceIDs(ctx context.Context, experimentIDs []int32) (
	[]model.AccessScopeID, error,
) {
	if len(experimentIDs) == 0 {
		return []model.AccessScopeID{}, nil
	}

	var rows []map[string]interface{}
	err := Bun().NewSelect().TableExpr("workspaces AS w").
		ColumnExpr("w.id AS workspace_id").
		ColumnExpr("e.id AS exp_id").
		Join("JOIN projects p ON w.id = p.workspace_id").
		Join("JOIN experiments e ON p.id = e.project_id").
		Where("e.id IN (?)", bun.In(experimentIDs)).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	workspaceSet := map[int]bool{}
	var workspaceIDs []model.AccessScopeID
	for _, row := range rows {
		workspaceID, ok := row["workspace_id"].(int64)
		if !ok {
			return nil, fmt.Errorf("workspaceID is not an int64")
		}
		workspaceSet[int(workspaceID)] = true
	}

	for wID := range workspaceSet {
		workspaceIDs = append(workspaceIDs, model.AccessScopeID(wID))
	}

	return workspaceIDs, nil
}

// TrialIDsToWorkspaceIDs returns a slice of workspaces that the given trials belong to.
func TrialIDsToWorkspaceIDs(ctx context.Context, trialIDs []int32) (
	[]model.AccessScopeID, error,
) {
	if len(trialIDs) == 0 {
		return []model.AccessScopeID{}, nil
	}

	var rows []map[string]interface{}
	err := Bun().NewSelect().TableExpr("workspaces AS w").
		ColumnExpr("w.id AS workspace_id").
		ColumnExpr("t.id AS trial_id").
		Join("JOIN projects p ON w.id = p.workspace_id").
		Join("JOIN experiments e ON p.id = e.project_id").
		Join("JOIN trials t ON e.id = t.experiment_id").
		Where("trial_id IN (?)", bun.In(trialIDs)).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	workspaceSet := map[int]bool{}
	var workspaceIDs []model.AccessScopeID

	for _, row := range rows {
		workspaceID, ok := row["workspace_id"].(int64)
		if !ok {
			return nil, fmt.Errorf("workspaceID is not an int64")
		}
		workspaceSet[int(workspaceID)] = true
	}

	for wID := range workspaceSet {
		workspaceIDs = append(workspaceIDs, model.AccessScopeID(wID))
	}

	return workspaceIDs, nil
}
