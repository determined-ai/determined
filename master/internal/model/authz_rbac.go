package model

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
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
	permission rbacv1.PermissionType,
) {
	logFields["userID"] = curUser.ID
	logFields["username"] = curUser.Username
	logFields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{permission},
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
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MODEL_REGISTRY)
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
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_MODEL_REGISTRY)
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
		rbacv1.PermissionType_PERMISSION_TYPE_EDIT_MODEL_REGISTRY)
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
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_MODEL_REGISTRY)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_MODEL_REGISTRY)
}

func init() {
	AuthZProvider.Register("rbac", &ModelAuthZRBAC{})
}
