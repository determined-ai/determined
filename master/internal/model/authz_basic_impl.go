package model

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

// ModelAuthZBasic is basic OSS controls.
type ModelAuthZBasic struct{}

// CanGetModels always returns true and a nil error.
func (a *ModelAuthZBasic) CanGetModels(ctx context.Context, curUser model.User, workspaceID int32,
) (canGetModel bool, serverError error) {
	return true, nil
}

// CanGetModel always returns true and a nil error.
func (a *ModelAuthZBasic) CanGetModel(ctx context.Context, curUser model.User,
	m *modelv1.Model, workspaceID int32,
) (canGetModel bool, serverError error) {
	return true, nil
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
