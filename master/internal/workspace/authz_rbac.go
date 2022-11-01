package workspace

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func init() {
	AuthZProvider.Register("rbac", &WorkspaceAuthZRBAC{})
}

// ErrorLookup is the error returned when a user's permissions couldn't be looked up.
var ErrorLookup = errors.New("error looking up user's permissions")

// WorkspaceAuthZRBAC is the RBAC implementation of WorkspaceAuthZ.
type WorkspaceAuthZRBAC struct{}

// FilterWorkspaceProjects filters a set of projects based on which workspaces a user has view
// permissions on.
func (r *WorkspaceAuthZRBAC) FilterWorkspaceProjects(
	ctx context.Context, curUser model.User, projects []*projectv1.Project,
) ([]*projectv1.Project, error) {
	workspaceIDs, err := workspacesUserHasPermissionOn(ctx, curUser.ID,
		workspaceIDsFromProjects(projects), rbacv1.PermissionType_PERMISSION_TYPE_VIEW_PROJECT)
	if err != nil {
		return nil, errors.Wrap(err, ErrorLookup.Error())
	}

	result := make([]*projectv1.Project, 0, len(projects))
	for _, p := range projects {
		if workspaceIDs[p.WorkspaceId] {
			result = append(result, p)
		}
	}

	return result, nil
}

// FilterWorkspaces filters workspaces based on which ones the user has view permissions on.
func (r *WorkspaceAuthZRBAC) FilterWorkspaces(
	ctx context.Context, curUser model.User, workspaces []*workspacev1.Workspace,
) ([]*workspacev1.Workspace, error) {
	workspaceIDs := idsFromWorkspaces(workspaces)

	ids, err := workspacesUserHasPermissionOn(ctx, curUser.ID, workspaceIDs,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_WORKSPACE)
	if err != nil {
		return nil, errors.Wrap(err, ErrorLookup.Error())
	}
	if len(ids) == len(workspaces) {
		return workspaces, nil
	}

	result := make([]*workspacev1.Workspace, 0, len(ids))
	for _, w := range workspaces {
		if ids[w.Id] {
			result = append(result, w)
		}
	}

	return result, nil
}

// CanGetWorkspace determines whether a user can view a workspace.
func (r *WorkspaceAuthZRBAC) CanGetWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) (canGetWorkspace bool, serverError error) {
	can, err := hasPermissionOnWorkspace(ctx, curUser.ID, workspace,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_WORKSPACE)
	if err != nil {
		return false, err
	}

	return can, nil
}

// CanCreateWorkspace determines whether a user can create workspaces.
func (r *WorkspaceAuthZRBAC) CanCreateWorkspace(ctx context.Context, curUser model.User) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_WORKSPACE)
}

// CanSetWorkspacesName determines whether a user can set a workspace's name.
func (r *WorkspaceAuthZRBAC) CanSetWorkspacesName(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, &workspace.Id,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_WORKSPACE)
}

// CanDeleteWorkspace determines whether a user can delete a workspace.
func (r *WorkspaceAuthZRBAC) CanDeleteWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, &workspace.Id,
		rbacv1.PermissionType_PERMISSION_TYPE_DELETE_WORKSPACE)
}

// CanArchiveWorkspace determines whether a user can archive a workspace.
func (r *WorkspaceAuthZRBAC) CanArchiveWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, &workspace.Id,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_WORKSPACE)
}

// CanUnarchiveWorkspace determines whether a user can unarchive a workspace.
func (r *WorkspaceAuthZRBAC) CanUnarchiveWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, &workspace.Id,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_WORKSPACE)
}

// CanPinWorkspace determines whether a user can pin a workspace.
func (r *WorkspaceAuthZRBAC) CanPinWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return nil
}

// CanUnpinWorkspace determines whether a user can unpin a workspace.
func (r *WorkspaceAuthZRBAC) CanUnpinWorkspace(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return nil
}

// CanCreateWorkspaceWithAgentUserGroup determines whether a user can set agent
// uid/gid on a new workspace.
func (r *WorkspaceAuthZRBAC) CanCreateWorkspaceWithAgentUserGroup(
	ctx context.Context, curUser model.User,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_SET_WORKSPACE_AGENT_USER_GROUP)
}

// CanSetWorkspacesAgentUserGroup determines whether a user can set agent uid/gid.
func (r *WorkspaceAuthZRBAC) CanSetWorkspacesAgentUserGroup(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, &workspace.Id,
		rbacv1.PermissionType_PERMISSION_TYPE_SET_WORKSPACE_AGENT_USER_GROUP)
}

// CanSetWorkspacesCheckpointStorageConfig determines if a user can set checkpoint storage access.
func (r *WorkspaceAuthZRBAC) CanSetWorkspacesCheckpointStorageConfig(
	ctx context.Context, curUser model.User, workspace *workspacev1.Workspace,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, &workspace.Id,
		rbacv1.PermissionType_PERMISSION_TYPE_SET_WORKSPACE_CHECKPOINT_STORAGE_CONFIG)
}

// CanCreateWorkspaceWithCheckpointStorageConfig determines if a user can set
// checkpoint storage access on a new workspace.
func (r *WorkspaceAuthZRBAC) CanCreateWorkspaceWithCheckpointStorageConfig(
	ctx context.Context, curUser model.User,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_WORKSPACE)
}

func hasPermissionOnWorkspace(ctx context.Context, uid model.UserID,
	workspace *workspacev1.Workspace, permID rbacv1.PermissionType,
) (bool, error) {
	var workspaceID *int32
	if workspace != nil {
		workspaceID = &workspace.Id
	}

	err := db.DoesPermissionMatch(ctx, uid, workspaceID, permID)
	if _, ok := err.(authz.PermissionDeniedError); ok {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, ErrorLookup.Error())
	}

	return true, nil
}

func workspacesUserHasPermissionOn(ctx context.Context, uid model.UserID,
	workspaceIDs []int32, permID rbacv1.PermissionType,
) (map[int32]bool, error) {
	// We'll want set intersection later, so let's set up for constant-time lookup
	inWorkspaceIDSet := make(map[int32]bool, len(workspaceIDs))
	for _, w := range workspaceIDs {
		inWorkspaceIDSet[w] = true
	}

	summary, err := rbac.GetPermissionSummary(ctx, uid)
	if err != nil {
		return nil, errors.Wrap(err, ErrorLookup.Error())
	}

	workspacesWithPermission := make(map[int32]bool)
	for role, assignments := range summary {
		// We only care about roles that contain the relevant permission
		ids := rbac.Permissions(role.Permissions).IDs()
		if !slices.Contains(ids, int(permID)) {
			continue
		}

		for _, assignment := range assignments {
			// If it's a global assignment, return the full set of ids
			if !assignment.Scope.WorkspaceID.Valid {
				return inWorkspaceIDSet, nil
			}

			// If this assignment is for a workspace in question, add it to the set
			if id := assignment.Scope.WorkspaceID.Int32; inWorkspaceIDSet[id] {
				workspacesWithPermission[id] = true
			}
		}
	}

	return workspacesWithPermission, nil
}

func idsFromWorkspaces(workspaces []*workspacev1.Workspace) []int32 {
	result := make([]int32, 0, len(workspaces))
	for _, w := range workspaces {
		if w == nil {
			continue
		}
		result = append(result, w.Id)
	}
	return result
}

func workspaceIDsFromProjects(projects []*projectv1.Project) []int32 {
	result := make([]int32, 0, len(projects))
	for _, p := range projects {
		if p == nil {
			continue
		}
		result = append(result, p.WorkspaceId)
	}
	return result
}
