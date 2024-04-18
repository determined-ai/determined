package workspace

import (
	"context"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// WorkspaceAuthZPermissive is the permission implementation.
type WorkspaceAuthZPermissive struct{}

// CanGetWorkspace calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanGetWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanGetWorkspace(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanGetWorkspace(ctx, curUser, workspace)
}

// CanGetWorkspaceID always return true and a nil error.
func (p *WorkspaceAuthZPermissive) CanGetWorkspaceID(
	ctx context.Context, curUser model.User, workspaceID int32,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanGetWorkspaceID(ctx, curUser, workspaceID)
	return (&WorkspaceAuthZBasic{}).CanGetWorkspaceID(ctx, curUser, workspaceID)
}

// CanModifyRPWorkspaceBindings requires user to be an admin.
func (p *WorkspaceAuthZPermissive) CanModifyRPWorkspaceBindings(
	ctx context.Context, curUser model.User, workspaceIDs []int32,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanModifyRPWorkspaceBindings(ctx, curUser, workspaceIDs)
	return (&WorkspaceAuthZBasic{}).CanModifyRPWorkspaceBindings(ctx, curUser, workspaceIDs)
}

// FilterWorkspaceProjects calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) FilterWorkspaceProjects(
	ctx context.Context, curUser model.User, projects []*projectv1.Project,
) ([]*projectv1.Project, error) {
	_, _ = (&WorkspaceAuthZRBAC{}).FilterWorkspaceProjects(ctx, curUser, projects)
	return (&WorkspaceAuthZBasic{}).FilterWorkspaceProjects(ctx, curUser, projects)
}

// FilterWorkspaces calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) FilterWorkspaces(
	ctx context.Context, curUser model.User, workspaces []*workspacev1.Workspace,
) ([]*workspacev1.Workspace, error) {
	_, _ = (&WorkspaceAuthZRBAC{}).FilterWorkspaces(ctx, curUser, workspaces)
	return (&WorkspaceAuthZBasic{}).FilterWorkspaces(ctx, curUser, workspaces)
}

// FilterWorkspaceIDs always returns provided list and a nil error.
func (p *WorkspaceAuthZPermissive) FilterWorkspaceIDs(
	ctx context.Context, curUser model.User, workspaceIDs []int32,
) ([]int32, error) {
	_, _ = (&WorkspaceAuthZRBAC{}).FilterWorkspaceIDs(ctx, curUser, workspaceIDs)
	return (&WorkspaceAuthZBasic{}).FilterWorkspaceIDs(ctx, curUser, workspaceIDs)
}

// CanCreateWorkspace calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanCreateWorkspace(
	ctx context.Context, curUser model.User,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanCreateWorkspace(ctx, curUser)
	return (&WorkspaceAuthZBasic{}).CanCreateWorkspace(ctx, curUser)
}

// CanCreateWorkspaceWithAgentUserGroup calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanCreateWorkspaceWithAgentUserGroup(
	ctx context.Context, curUser model.User,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanCreateWorkspaceWithAgentUserGroup(ctx, curUser)
	return (&WorkspaceAuthZBasic{}).CanCreateWorkspaceWithAgentUserGroup(ctx, curUser)
}

// CanSetWorkspacesName calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanSetWorkspacesName(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanSetWorkspacesName(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanSetWorkspacesName(ctx, curUser, workspace)
}

// CanSetWorkspacesAgentUserGroup calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanSetWorkspacesAgentUserGroup(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanSetWorkspacesAgentUserGroup(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanSetWorkspacesAgentUserGroup(ctx, curUser, workspace)
}

// CanDeleteWorkspace calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanDeleteWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanDeleteWorkspace(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanDeleteWorkspace(ctx, curUser, workspace)
}

// CanArchiveWorkspace calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanArchiveWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanArchiveWorkspace(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanArchiveWorkspace(ctx, curUser, workspace)
}

// CanUnarchiveWorkspace calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanUnarchiveWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanUnarchiveWorkspace(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanUnarchiveWorkspace(ctx, curUser, workspace)
}

// CanPinWorkspace calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanPinWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanPinWorkspace(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanPinWorkspace(ctx, curUser, workspace)
}

// CanUnpinWorkspace calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanUnpinWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanUnpinWorkspace(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanUnpinWorkspace(ctx, curUser, workspace)
}

// CanSetWorkspacesCheckpointStorageConfig calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanSetWorkspacesCheckpointStorageConfig(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanSetWorkspacesCheckpointStorageConfig(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanSetWorkspacesCheckpointStorageConfig(ctx, curUser, workspace)
}

// CanCreateWorkspaceWithCheckpointStorageConfig calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanCreateWorkspaceWithCheckpointStorageConfig(
	ctx context.Context, curUser model.User,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanCreateWorkspaceWithCheckpointStorageConfig(ctx, curUser)
	return (&WorkspaceAuthZBasic{}).CanCreateWorkspaceWithCheckpointStorageConfig(ctx, curUser)
}

// CanSetWorkspacesDefaultPools calls RBAC authz but enforces basic authz.
func (p *WorkspaceAuthZPermissive) CanSetWorkspacesDefaultPools(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	_ = (&WorkspaceAuthZRBAC{}).CanSetWorkspacesDefaultPools(ctx, curUser, workspace)
	return (&WorkspaceAuthZBasic{}).CanSetWorkspacesDefaultPools(ctx, curUser, workspace)
}

func init() {
	AuthZProvider.Register("permissive", &WorkspaceAuthZPermissive{})
}
