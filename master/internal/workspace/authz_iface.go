package workspace

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// WorkspaceAuthZ is the interface for workspace authorization.
type WorkspaceAuthZ interface {
	// GET /api/v1/workspaces/:workspace_id
	CanGetWorkspace(
		curUser model.User, workspace *workspacev1.Workspace,
	) (canGetWorkspace bool, serverError error)

	// GET /api/v1/workspaces/:workspace_id/projects
	FilterWorkspaceProjects(
		curUser model.User, projects []*projectv1.Project,
	) ([]*projectv1.Project, error)

	// GET /api/v1/workspaces
	FilterWorkspaces(
		curUser model.User, workspaces []*workspacev1.Workspace,
	) ([]*workspacev1.Workspace, error)

	// POST /api/v1/workspaces
	CanCreateWorkspace(curUser model.User) error

	// PATCH /api/v1/workspaces/:workspace_id
	CanSetWorkspacesName(curUser model.User, workspace *workspacev1.Workspace) error

	// DELETE /api/v1/workspaces/:workspace_id
	CanDeleteWorkspace(curUser model.User, workspace *workspacev1.Workspace) error

	// POST /api/v1/workspaces/:workspace_id/archive
	CanArchiveWorkspace(curUser model.User, workspace *workspacev1.Workspace) error
	// POST /api/v1/workspaces/:workspace_id/unarchive
	CanUnarchiveWorkspace(curUser model.User, workspace *workspacev1.Workspace) error

	// POST /api/v1/workspaces/:workspace_id/pin
	CanPinWorkspace(curUser model.User, workspace *workspacev1.Workspace) error
	// POST /api/v1/workspaces/:workspace_id/unpin
	CanUnpinWorkspace(curUser model.User, workspace *workspacev1.Workspace) error
}

// AuthZProvider providers WorkspaceAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[WorkspaceAuthZ]
