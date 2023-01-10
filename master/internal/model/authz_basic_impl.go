package model

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
)

// ModelAuthZBasic is basic OSS controls.
type ModelAuthZBasic struct{}

// CanGetModel always returns true and a nill error.
func (a *ModelAuthZBasic) CanGetModel(ctx context.Context, curUser model.User,
	m *model.Model,
) (canGetModel bool, serverError error) {
	return true, nil
}

// CanEditModel always returns true and a nil error.
func (a *ModelAuthZBasic) CanEditModel(ctx context.Context, curUser model.User,
	m *model.Model,
) error {
	return nil
}

// CanCreateModel always returns true and a nil error.
func (a *ModelAuthZBasic) CanCreateModel(ctx context.Context,
	curUser model.User, m *model.Model,
) error {
	return nil
}
