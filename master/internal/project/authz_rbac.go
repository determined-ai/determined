package project

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func init() {
	AuthZProvider.Register("rbac", &ProjectAuthZRBAC{})
}

// ProjectAuthZRBAC is the RBAC implementation of ProjectAuthZ.
type ProjectAuthZRBAC struct{}

func logEntryWithProjectTarget(
	fields log.Fields, curUser model.User, permission rbacv1.PermissionType, projectIDs ...int32,
) {
	pids := make([]string, 0, len(projectIDs))
	for _, pid := range projectIDs {
		pids = append(pids, fmt.Sprint(pid))
	}

	fields["userID"] = curUser.ID
	fields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{permission},
			SubjectType:     "project",
			SubjectIDs:      pids,
		},
	}
}

func permCheck(
	ctx context.Context, curUser model.User, workspaceID int32, perm rbacv1.PermissionType,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID, perm)
}

// CanGetProject returns true if user has "VIEW_PROJECT" globally
// or on a given workspace scope scope and false if not along
// with a serverError in case of a database failure.
func (a *ProjectAuthZRBAC) CanGetProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (canGetProject bool, serverError error) {
	fields := audit.ExtractLogFields(ctx)
	logEntryWithProjectTarget(fields, curUser, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_PROJECT,
		project.Id)
	defer func() {
		fields["permissionGranted"] = canGetProject
		audit.Log(fields)
	}()

	if err := permCheck(ctx, curUser, project.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_PROJECT); err != nil {
		if _, ok := err.(authz.PermissionDeniedError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CanCreateProject returns an error if a user doesn't have "CREATE_PROJECT" globally
// or on the target workspace.
func (a *ProjectAuthZRBAC) CanCreateProject(
	ctx context.Context, curUser model.User, willBeInWorkspace *workspacev1.Workspace,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logEntryWithProjectTarget(fields, curUser, rbacv1.PermissionType_PERMISSION_TYPE_CREATE_PROJECT)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return permCheck(ctx, curUser, willBeInWorkspace.Id,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_PROJECT)
}

// CanSetProjectNotes returns an error if a user doesn't have "UPDATE_PROJECT" globally
// or on the target project's workspace.
func (a *ProjectAuthZRBAC) CanSetProjectNotes(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logEntryWithProjectTarget(fields, curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT, project.Id)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return permCheck(ctx, curUser, project.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT)
}

// CanSetProjectName returns an error if a user doesn't have "UPDATE_PROJECT" globally
// or on the target project's workspace.
func (a *ProjectAuthZRBAC) CanSetProjectName(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logEntryWithProjectTarget(fields, curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT, project.Id)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return permCheck(ctx, curUser, project.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT)
}

// CanSetProjectDescription returns an error if a user doesn't have "UPDATE_PROJECT" globally
// or on the target project's workspace.
func (a *ProjectAuthZRBAC) CanSetProjectDescription(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logEntryWithProjectTarget(fields, curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT, project.Id)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return permCheck(ctx, curUser, project.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT)
}

// CanDeleteProject returns an error if a user doesn't have "DELETE_PROJECT" globally
// or on the target project's workspace.
func (a *ProjectAuthZRBAC) CanDeleteProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logEntryWithProjectTarget(fields, curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_DELETE_PROJECT, project.Id)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return permCheck(ctx, curUser, project.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_DELETE_PROJECT)
}

// CanMoveProject returns an error if a user doesn't have "DELETE_PROJECT" globally
// or on the from workspace and if
// a user also doesn't have "CREATE_PROJECT" globally or on the to workspace.
func (a *ProjectAuthZRBAC) CanMoveProject(
	ctx context.Context,
	curUser model.User,
	project *projectv1.Project,
	from, to *workspacev1.Workspace,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUser.ID
	fields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{
				rbacv1.PermissionType_PERMISSION_TYPE_DELETE_PROJECT,
			},
			SubjectType: "workspace",
			SubjectIDs:  []string{fmt.Sprint(from.Id)},
		},
		{
			PermissionTypes: []rbacv1.PermissionType{
				rbacv1.PermissionType_PERMISSION_TYPE_CREATE_PROJECT,
			},
			SubjectType: "workspace",
			SubjectIDs:  []string{fmt.Sprint(to.Id)},
		},
	}
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	if err := permCheck(ctx, curUser, from.Id,
		rbacv1.PermissionType_PERMISSION_TYPE_DELETE_PROJECT); err != nil {
		return err
	}
	return permCheck(ctx, curUser, to.Id, rbacv1.PermissionType_PERMISSION_TYPE_CREATE_PROJECT)
}

// CanMoveProjectExperiments returns an error if a user doesn't have "DELETE_EXPERIMENT" globally
// or on the from workspace and if a user also
// doesn't have "CREATE_EXPERIMENT" globally or on the to workspace.
func (a *ProjectAuthZRBAC) CanMoveProjectExperiments(
	ctx context.Context, curUser model.User, exp *model.Experiment, from, to *projectv1.Project,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUser.ID
	fields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{
				rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT,
			},
			SubjectType: "project",
			SubjectIDs:  []string{fmt.Sprint(from.Id)},
		},
		{
			PermissionTypes: []rbacv1.PermissionType{
				rbacv1.PermissionType_PERMISSION_TYPE_CREATE_EXPERIMENT,
			},
			SubjectType: "project",
			SubjectIDs:  []string{fmt.Sprint(to.Id)},
		},
	}
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	if err := permCheck(ctx, curUser, from.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT); err != nil {
		return err
	}
	return permCheck(ctx, curUser, to.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_EXPERIMENT)
}

// CanArchiveProject returns an error if a user doesn't have "UPDATE_PROJECT" globally
// or on the target project's workspace.
func (a *ProjectAuthZRBAC) CanArchiveProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logEntryWithProjectTarget(fields, curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT, project.Id)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return permCheck(ctx, curUser, project.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT)
}

// CanUnarchiveProject returns an error if a user doesn't have "UPDATE_PROJECT" globally
// or on the target project's workspace.
func (a *ProjectAuthZRBAC) CanUnarchiveProject(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	logEntryWithProjectTarget(fields, curUser,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT, project.Id)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return permCheck(ctx, curUser, project.WorkspaceId,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_PROJECT)
}
