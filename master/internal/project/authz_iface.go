package project

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// ProjectAuthZ is the interface for project authorization.
type ProjectAuthZ interface {
	// GET /api/v1/projects/:project_id
	CanGetProject(curUser model.User, project *projectv1.Project) (
		canGetProject bool, serverError error,
	)

	// POST /api/v1/workspaces/:workspace_id/projects
	CanCreateProject(curUser model.User, targetWorkspace *workspacev1.Workspace) error

	// POST /api/v1/projects/:project_id/notes
	// PUT /api/v1/projects/:project_id/notes
	CanSetProjectNotes(curUser model.User, project *projectv1.Project) error

	// PATCH /api/v1/projects/:project_id
	CanSetProjectName(curUser model.User, project *projectv1.Project) error
	CanSetProjectDescription(curUser model.User, project *projectv1.Project) error

	// DELETE /api/v1/projects/:project_id
	CanDeleteProject(curUser model.User, targetProject *projectv1.Project) error

	// POST /api/v1/projects/:project_id/move
	CanMoveProject(
		curUser model.User, project *projectv1.Project, from, to *workspacev1.Workspace,
	) error

	// POST /api/v1/experiments/:experiment_id/move
	CanMoveProjectExperiments(
		curUser model.User, exp *experimentv1.Experiment, from, to *projectv1.Project,
	) error

	// POST /api/v1/projects/:project_id/archive
	CanArchiveProject(curUser model.User, project *projectv1.Project) error
	// POST /api/v1/projects/:project_id/unarchive
	CanUnarchiveProject(curUser model.User, project *projectv1.Project) error
}

// AuthZProvider providers ProjectAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[ProjectAuthZ]
