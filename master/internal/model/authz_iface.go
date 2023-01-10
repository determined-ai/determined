package model

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

type ModelAuthZ interface {
	// Get Checkpoint
	// GET Model
	// Get Model version
	CanGetModel(ctx context.Context, curUser model.User, m *model.Model) (canGetModel bool, serverError error)
	// Patch model
	// Patch model version
	// Post model version
	// Archive model
	// Unarchive model
	CanEditModel(ctx context.Context, curUser model.User, m *model.Model) error
	// Post model
	CanCreateModel(ctx context.Context, curUser model.User, m *model.Model) error
}

var AuthZProvider authz.AuthZProviderType[ModelAuthZ]
