package model

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

// ModelAuthZBasic is basic OSS controls.
type ModelAuthZBasic struct{}

// CanGetModels always returns true and a nil error.
func (a *ModelAuthZBasic) CanGetModels(ctx context.Context,
	curUser model.User, workspaceIDs []int32,
) (workspaceIDsWithPermsFilter []int32, serverError error) {
	return workspaceIDs, nil
}

// CanGetModel always returns true and a nil error.
func (a *ModelAuthZBasic) CanGetModel(ctx context.Context, curUser model.User,
	m *modelv1.Model, workspaceID int32,
) error {
	return nil
}

// CanEditModel always returns true and a nil error.
func (a *ModelAuthZBasic) CanEditModel(ctx context.Context, curUser model.User,
	m *modelv1.Model, workspaceID int32,
) error {
	return nil
}

// CanCreateModel always returns true and a nil error.
func (a *ModelAuthZBasic) CanCreateModel(ctx context.Context,
	curUser model.User, workspaceID int32,
) error {
	return nil
}

// CanDeleteModel returns an error if the model
// is not owned by the current user and the current user is not an admin.
func (a *ModelAuthZBasic) CanDeleteModel(ctx context.Context, curUser model.User,
	m *modelv1.Model, workspaceID int32,
) error {
	// TODO: Modify model UserID to use UserID
	curUserIsOwner := m.UserId == int32(curUser.ID)
	if !curUser.Admin && !curUserIsOwner {
		return authz.PermissionDeniedError{}.WithPrefix(
			"non-admin users may not delete other users' models",
		)
	}
	return nil
}

// CanDeleteModelVersion returns an error if the model/model version
// is not owned by the current user and the current user is not an admin.
func (a *ModelAuthZBasic) CanDeleteModelVersion(ctx context.Context, curUser model.User,
	modelVersion *modelv1.ModelVersion, workspaceID int32,
) error {
	curUserIsOwner := modelVersion.UserId == int32(curUser.ID) ||
		modelVersion.Model.UserId == int32(curUser.ID)
	if !curUser.Admin && !curUserIsOwner {
		return authz.PermissionDeniedError{}.WithPrefix(
			"non-admin users may not delete other users' model versions",
		)
	}
	return nil
}

// CanMoveModel always returns true and a nil error.
func (a *ModelAuthZBasic) CanMoveModel(
	ctx context.Context,
	curUser model.User,
	modelRegister *modelv1.Model,
	fromWorkspaceID int32,
	toWorkspaceID int32,
) error {
	return nil
}

// FilterReadableModelsQuery returns the query unmodified and a nil error.
func (a *ModelAuthZBasic) FilterReadableModelsQuery(
	ctx context.Context, curUser model.User, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	return query, nil
}

func init() {
	AuthZProvider.Register("basic", &ModelAuthZBasic{})
}
