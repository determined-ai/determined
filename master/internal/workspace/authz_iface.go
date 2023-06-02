package workspace

import (
	"context"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// WorkspaceAuthZ is the interface for workspace authorization.
type WorkspaceAuthZ interface {
	// GET /api/v1/workspaces/:workspace_id
	CanGetWorkspace(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error

	// POST /api/v1/resource-pools/workspace-bind
	// POST /api/v1/resource-pools/workspace-unbind
	CanModifyRPWorkspaceBindings(
		ctx context.Context, curUser model.User, workspaceIDs []int32,
	) error

	// GET /api/v1/workspaces/:workspace_id/projects
	FilterWorkspaceProjects(
		ctx context.Context, curUser model.User, projects []*projectv1.Project,
	) ([]*projectv1.Project, error)

	// GET /api/v1/workspaces
	FilterWorkspaces(
		ctx context.Context, curUser model.User, workspaces []*workspacev1.Workspace,
	) ([]*workspacev1.Workspace, error)

	// POST /api/v1/workspaces
	CanCreateWorkspace(ctx context.Context, curUser model.User) error
	CanCreateWorkspaceWithAgentUserGroup(ctx context.Context, curUser model.User) error
	CanCreateWorkspaceWithCheckpointStorageConfig(ctx context.Context, curUser model.User) error

	// PATCH /api/v1/workspaces/:workspace_id
	CanSetWorkspacesName(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error
	CanSetWorkspacesAgentUserGroup(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error
	CanSetWorkspacesCheckpointStorageConfig(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error

	// DELETE /api/v1/workspaces/:workspace_id
	CanDeleteWorkspace(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error

	// POST /api/v1/workspaces/:workspace_id/archive
	CanArchiveWorkspace(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error
	// POST /api/v1/workspaces/:workspace_id/unarchive
	CanUnarchiveWorkspace(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error

	// POST /api/v1/workspaces/:workspace_id/pin
	CanPinWorkspace(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error
	// POST /api/v1/workspaces/:workspace_id/unpin
	CanUnpinWorkspace(
		ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
	) error
}

// AuthZProvider providers WorkspaceAuthZ implementations.
var AuthZProvider authz.AuthZProviderType[WorkspaceAuthZ]
