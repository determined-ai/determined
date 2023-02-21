package model

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

// ModelAuthZPermissive is the permission implementation.
type ModelAuthZPermissive struct{}

// CanGetModels always returns true and a nil error.
func (a *ModelAuthZPermissive) CanGetModels(ctx context.Context, curUser model.User,
	workspaceID int32,
) (canGetModel bool, serverError error) {
	_, _ = (&ModelAuthZRBAC{}).CanGetModels(ctx, curUser, workspaceID)
	return (&ModelAuthZBasic{}).CanGetModels(ctx, curUser, workspaceID)
}

// CanGetModel always returns true and a nil error.
func (a *ModelAuthZPermissive) CanGetModel(ctx context.Context, curUser model.User,
	m *modelv1.Model, workspaceID int32,
) (canGetModel bool, serverError error) {
	_, _ = (&ModelAuthZRBAC{}).CanGetModel(ctx, curUser, m, workspaceID)
	return (&ModelAuthZBasic{}).CanGetModel(ctx, curUser, m, workspaceID)
}

// CanEditModel always returns true and a nil error.
func (a *ModelAuthZPermissive) CanEditModel(ctx context.Context, curUser model.User,
	m *modelv1.Model, workspaceID int32,
) error {
	_ = (&ModelAuthZRBAC{}).CanEditModel(ctx, curUser, m, workspaceID)
	return (&ModelAuthZBasic{}).CanEditModel(ctx, curUser, m, workspaceID)
}

// CanCreateModel always returns true and a nil error.
func (a *ModelAuthZPermissive) CanCreateModel(ctx context.Context,
	curUser model.User, workspaceID int32,
) error {
	_ = (&ModelAuthZRBAC{}).CanCreateModel(ctx, curUser, workspaceID)
	return (&ModelAuthZBasic{}).CanCreateModel(ctx, curUser, workspaceID)
}

// CanMoveModel always returns true.
func (a *ModelAuthZPermissive) CanMoveModel(ctx context.Context,
	curUser model.User, m *modelv1.Model, origin int32, destination int32,
) error {
	_ = (&ModelAuthZRBAC{}).CanMoveModel(ctx, curUser, m, origin, destination)
	return (&ModelAuthZBasic{}).CanMoveModel(ctx, curUser, m, origin, destination)
}

// FilterReadableModelsQuery returns query and a nil error.
func (a *ModelAuthZPermissive) FilterReadableModelsQuery(
	ctx context.Context, curUser model.User, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	_, _ = (&ModelAuthZRBAC{}).FilterReadableModelsQuery(ctx, curUser, query)
	return (&ModelAuthZBasic{}).FilterReadableModelsQuery(ctx, curUser, query)
}

func init() {
	AuthZProvider.Register("permissive", &ModelAuthZPermissive{})
}
