package model

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/modelv1"
)

// ModelAuthZ describes authz methods for experiments.
type ModelAuthZ interface {
	// GET /api/v1/models
	CanGetModels(ctx context.Context, curUser model.User, workspaceIDs []int32,
	) (workspaceIDsWithPermsFilter []int32, serverError error)
	// GET /api/v1/checkpoints/{checkpoint_uuid}
	// GET /api/v1/models/{model_name}
	// GET /api/v1/models/{model_name}/versions/{model_version_num}
	// GET /api/v1/models/{model_name}/versions
	CanGetModel(ctx context.Context, curUser model.User,
		m *modelv1.Model, workspaceID int32,
	) error
	// PATCH /api/v1/models/{model_name}
	// PATCH /api/v1/models/{model_name}/versions/{model_version_num}
	// POST /api/v1/models/{model_name}/versions
	// POST /api/v1/models/{model_name}/archive
	// POST /api/v1/models/{model_name}/unarchive
	CanEditModel(ctx context.Context, curUser model.User,
		m *modelv1.Model, workspaceID int32,
	) error
	// POST /api/v1/models
	CanCreateModel(ctx context.Context,
		curUser model.User, workspaceID int32,
	) error
	// DELETE /api/v1/models/{modelName}
	CanDeleteModel(ctx context.Context, curUser model.User,
		m *modelv1.Model, workspaceID int32,
	) error
	// DELETE /api/v1/models/{modelName}/versions/{modelVersionNum}
	CanDeleteModelVersion(ctx context.Context, curUser model.User,
		modelVersion *modelv1.ModelVersion, workspaceID int32) error
	// POST /api/v1/models/{model_name}/move
	CanMoveModel(ctx context.Context, curUser model.User, model *modelv1.Model,
		fromWorkspaceID int32, toWorkspaceID int32) error

	// GET /api/v1/models with filter to allow reading
	FilterReadableModelsQuery(
		ctx context.Context, curUser model.User, query *bun.SelectQuery,
	) (*bun.SelectQuery, error)
}

// AuthZProvider is the authz registry for models.
var AuthZProvider authz.AuthZProviderType[ModelAuthZ]
