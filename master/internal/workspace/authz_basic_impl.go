package workspace

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

// WorkspaceAuthZBasic is classic OSS Determined authentication for workspaces.
type WorkspaceAuthZBasic struct{}

// CanGetWorkspace always return true and a nil error.
func (a *WorkspaceAuthZBasic) CanGetWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return nil
}

// CanBindRPWorkspace requires user to be an admin.
func (a *WorkspaceAuthZBasic) CanBindRPWorkspace(
	ctx context.Context, curUser model.User, workspaceIDs []int32,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can bind resource pool to a workspace")
	}
	return nil
}

// CanUnBindRPWorkspace requires user to be an admin.
func (a *WorkspaceAuthZBasic) CanUnBindRPWorkspace(
	ctx context.Context, curUser model.User, workspaceIDs []int32,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can unbind resource pool to a workspace")
	}
	return nil
}

// FilterWorkspaceProjects always returns the list provided and a nil error.
func (a *WorkspaceAuthZBasic) FilterWorkspaceProjects(
	ctx context.Context, curUser model.User, projects []*projectv1.Project,
) ([]*projectv1.Project, error) {
	return projects, nil
}

// FilterWorkspaces always returns provided list and a nil errir.
func (a *WorkspaceAuthZBasic) FilterWorkspaces(
	ctx context.Context, curUser model.User, workspaces []*workspacev1.Workspace,
) ([]*workspacev1.Workspace, error) {
	return workspaces, nil
}

// CanCreateWorkspace always returns a nil error.
func (a *WorkspaceAuthZBasic) CanCreateWorkspace(ctx context.Context, curUser model.User) error {
	return nil
}

// CanCreateWorkspaceWithAgentUserGroup requires user to be an admin.
func (a *WorkspaceAuthZBasic) CanCreateWorkspaceWithAgentUserGroup(
	ctx context.Context, curUser model.User,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can set workspace agent user groups")
	}
	return nil
}

// CanSetWorkspacesName returns an error if the user is not an admin
// or not the owner of the workspace.
func (a *WorkspaceAuthZBasic) CanSetWorkspacesName(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may set other user's workspaces names")
	}
	return nil
}

// CanSetWorkspacesAgentUserGroup can only be done by admins.
func (a *WorkspaceAuthZBasic) CanSetWorkspacesAgentUserGroup(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can set workspace agent user groups")
	}
	return nil
}

// CanDeleteWorkspace returns an error if the user is not an admin
// or not the owner of the workspace.
func (a *WorkspaceAuthZBasic) CanDeleteWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may delete other user's workspaces")
	}
	return nil
}

// CanArchiveWorkspace returns an error if the user is not an admin
// or not the owner of the workspace.
func (a *WorkspaceAuthZBasic) CanArchiveWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may archive other user's workspaces")
	}
	return nil
}

// CanUnarchiveWorkspace returns an error if the user is not an admin
// or not the owner of the workspace.
func (a *WorkspaceAuthZBasic) CanUnarchiveWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may unarchive other user's workspaces")
	}
	return nil
}

// CanPinWorkspace always returns a nil error.
func (a *WorkspaceAuthZBasic) CanPinWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return nil
}

// CanUnpinWorkspace always returns a nil error.
func (a *WorkspaceAuthZBasic) CanUnpinWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return nil
}

// CanSetWorkspacesCheckpointStorageConfig returns an error if the user is not an admin
// or owner of the workspace.
func (a *WorkspaceAuthZBasic) CanSetWorkspacesCheckpointStorageConfig(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	if !curUser.Admin && curUser.ID != model.UserID(workspace.UserId) {
		return fmt.Errorf("only admins may set checkpoint storage config on other user's workspaces")
	}
	return nil
}

// CanCreateWorkspaceWithCheckpointStorageConfig returns an nil error.
func (a *WorkspaceAuthZBasic) CanCreateWorkspaceWithCheckpointStorageConfig(
	ctx context.Context, curUser model.User,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &WorkspaceAuthZBasic{})
}
