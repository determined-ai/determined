package model

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/internal/rbac/audit"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// ModelAuthZRBAC RBAC enabled controls.
type ModelAuthZRBAC struct{}

func addExpInfo(
	curUser model.User,
	logFields log.Fields, subjectID string,
	permissions []rbacv1.PermissionType,
) {
	logFields["userID"] = curUser.ID
	logFields["username"] = curUser.Username
	logFields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: permissions,
			SubjectType:     "model",
			SubjectIDs:      []string{subjectID},
		},
	}
}

// CanGetModels always returns true and a nil error.
func (a *ModelAuthZRBAC) CanGetModels(ctx context.Context, curUser model.User, workspaceID int32,
) (canGetModel bool, serverError error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, fields, fmt.Sprintf("all models in %d", workspaceID),
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MODEL_REGISTRY})
	defer func() {
		fields["permissionGranted"] = canGetModel
		audit.Log(fields)
	}()

	if err := db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MODEL_REGISTRY); err != nil {
		if _, ok := err.(authz.PermissionDeniedError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CanGetModel always returns true and a nil error.
func (a *ModelAuthZRBAC) CanGetModel(ctx context.Context, curUser model.User,
	m *modelv1.Model, workspaceID int32,
) (canGetModel bool, serverError error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, fields, string(m.Id),
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MODEL_REGISTRY})
	defer func() {
		fields["permissionGranted"] = canGetModel
		audit.Log(fields)
	}()

	if err := db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MODEL_REGISTRY); err != nil {
		if _, ok := err.(authz.PermissionDeniedError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CanEditModel always returns true and a nil error.
func (a *ModelAuthZRBAC) CanEditModel(ctx context.Context, curUser model.User,
	m *modelv1.Model, workspaceID int32,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, fields, string(m.Id),
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_EDIT_MODEL_REGISTRY})
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_EDIT_MODEL_REGISTRY)
}

// CanCreateModel always returns true and a nil error.
func (a *ModelAuthZRBAC) CanCreateModel(ctx context.Context,
	curUser model.User, workspaceID int32,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, fields, fmt.Sprintf("creating a model in %d", workspaceID),
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_CREATE_MODEL_REGISTRY})
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_MODEL_REGISTRY)
}

// CanMoveModel checks for edit permission in origin and create permission in destination.
func (a *ModelAuthZRBAC) CanMoveModel(ctx context.Context,
	curUser model.User, _ *modelv1.Model, origin int32, destination int32,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, fields, fmt.Sprintf("moving model from workspace %d to %d", origin,
		destination),
		[]rbacv1.PermissionType{
			rbacv1.PermissionType_PERMISSION_TYPE_EDIT_MODEL_REGISTRY,
			rbacv1.PermissionType_PERMISSION_TYPE_CREATE_MODEL_REGISTRY,
		})
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	origErr := db.DoesPermissionMatch(ctx, curUser.ID, &origin,
		rbacv1.PermissionType_PERMISSION_TYPE_EDIT_MODEL_REGISTRY)
	if origErr != nil {
		return origErr
	}

	return db.DoesPermissionMatch(ctx, curUser.ID, &destination,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_MODEL_REGISTRY)
}

// FilterReadableModelsQuery returns query in relevant workspaces and a nil error.
func (a *ModelAuthZRBAC) FilterReadableModelsQuery(
	ctx context.Context, curUser model.User, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUser.ID
	fields["permissionRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{
				rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MODEL_REGISTRY,
			},
			SubjectType: "models",
		},
	}

	var err error
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	assignmentsMap, err := rbac.GetPermissionSummary(ctx, curUser.ID)
	if err != nil {
		return query, err
	}

	var workspaces []int32

	for role, roleAssignments := range assignmentsMap {
		for _, permission := range role.Permissions {
			if permission.ID == int(
				rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MODEL_REGISTRY) {
				for _, assignment := range roleAssignments {
					if assignment.Scope.WorkspaceID.Valid {
						workspaces = append(workspaces, assignment.Scope.WorkspaceID.Int32)
					} else {
						// if permission is global, return without filtering
						return query, nil
					}
				}
			}
		}
	}

	if len(workspaces) == 0 {
		return query.Where("false"), nil
	}

	query = query.Where("workspace_id IN (?)", bun.In(workspaces))

	return query, nil
}

func init() {
	AuthZProvider.Register("rbac", &ModelAuthZRBAC{})
}
