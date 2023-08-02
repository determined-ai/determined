package project

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// ProjectAuthZ is the interface for project authorization.
type ProjectAuthZ interface {
	// GET /api/v1/projects/:project_id
	CanGetProject(ctx context.Context, curUser model.User, project *projectv1.Project) error

	// POST /api/v1/workspaces/:workspace_id/projects
	CanCreateProject(
		ctx context.Context, curUser model.User, targetWorkspace *workspacev1.Workspace,
	) error

	// POST /api/v1/projects/:project_id/notes
	// PUT /api/v1/projects/:project_id/notes
	CanSetProjectNotes(ctx context.Context, curUser model.User, project *projectv1.Project) error

	// PATCH /api/v1/projects/:project_id
	CanSetProjectName(ctx context.Context, curUser model.User, project *projectv1.Project) error
	CanSetProjectDescription(
		ctx context.Context, curUser model.User, project *projectv1.Project,
	) error

	// DELETE /api/v1/projects/:project_id
	CanDeleteProject(
		ctx context.Context, curUser model.User, targetProject *projectv1.Project,
	) error

	// POST /api/v1/projects/:project_id/move
	CanMoveProject(ctx context.Context, curUser model.User, project *projectv1.Project, from,
		to *workspacev1.Workspace) error

	// POST /api/v1/experiments/:experiment_id/move
	CanMoveProjectExperiments(ctx context.Context, curUser model.User, exp *model.Experiment, from,
		to *projectv1.Project) error

	// POST /api/v1/projects/:project_id/archive
	CanArchiveProject(ctx context.Context, curUser model.User, project *projectv1.Project) error
	// POST /api/v1/projects/:project_id/unarchive
	CanUnarchiveProject(ctx context.Context, curUser model.User, project *projectv1.Project) error
}

// AuthZProvider providers ProjectAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[ProjectAuthZ]
