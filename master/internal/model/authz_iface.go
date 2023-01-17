package model

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

// ModelAuthZ describes authz methods for experiments.
type ModelAuthZ interface {
	// Nikita: Add the actual API url here.
	// GetModels
	CanGetModels(ctx context.Context, curUser model.User,
		workspaceID int32) (canGetModel bool, serverError error)
	// Get Checkpoint
	// GetModel
	// GetModel version
	CanGetModel(ctx context.Context, curUser model.User,
		m *modelv1.Model, workspaceID int32,
	) (canGetModel bool, serverError error)
	// Patch model
	// Patch model version
	// Post model version
	// Archive model
	// Unarchive model
	CanEditModel(ctx context.Context, curUser model.User,
		m *modelv1.Model, workspaceID int32,
	) error
	// Post model
	CanCreateModel(ctx context.Context,
		curUser model.User, workspaceID int32,
	) error
}

// AuthZProvider is the authz registry for models.
var AuthZProvider authz.AuthZProviderType[ModelAuthZ]
