package experiment

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// ExperimentAuthZRBAC is RBAC enabled controls.
type ExperimentAuthZRBAC struct{}

func getWorkspaceFromExperiment(ctx context.Context, e *model.Experiment,
) (int32, error) {
	var workspaceID int32
	err := db.Bun().NewRaw("SELECT workspace_id FROM projects WHERE id = ?",
		e.ProjectID).Scan(ctx, &workspaceID)
	return workspaceID, err
}

func addExpInfo(
	curUser model.User,
	e *model.Experiment,
	logFields log.Fields,
	permission rbacv1.PermissionType,
) {
	logFields["userID"] = curUser.ID
	logFields["username"] = curUser.Username
	logFields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{permission},
			SubjectType:     "experiment",
			SubjectIDs:      []string{fmt.Sprint(e.ID)},
		},
	}
}

// CanGetExperiment checks if a user has permission to view an experiment.
func (a *ExperimentAuthZRBAC) CanGetExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) (canGetExp bool, serverError error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, e, fields, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
	defer func() {
		fields["permissionGranted"] = canGetExp
		audit.Log(fields)
	}()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return false, err
	}

	if err = db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA); err != nil {
		if _, ok := err.(authz.PermissionDeniedError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CanGetExperimentArtifacts checks if a user has permission to view experiment artifacts.
func (a *ExperimentAuthZRBAC) CanGetExperimentArtifacts(
	ctx context.Context, curUser model.User, e *model.Experiment,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, e, fields, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS)
}

// CanDeleteExperiment checks if a user has permission to delete an experiment.
func (a *ExperimentAuthZRBAC) CanDeleteExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, e, fields, rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT)
}

// FilterExperimentsQuery filters a query for what experiments a user can view.
func (a *ExperimentAuthZRBAC) FilterExperimentsQuery(
	ctx context.Context, curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
	permissions []rbacv1.PermissionType,
) (selectQuery *bun.SelectQuery, err error) {
	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUser.ID
	fields["permissionRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: permissions,
			SubjectType:     "experiments",
		},
	}

	defer func() {
		audit.LogFromErr(fields, err)
	}()

	assignmentsMap, err := rbac.GetPermissionSummary(ctx, curUser.ID)
	if err != nil {
		return query, err
	}

	var workspaces []int32
	neededPermissionsGlobal := []int{}
	paramPermissions := []int{}
	for _, p := range permissions {
		paramPermissions = append(paramPermissions, int(p))
	}
	copy(paramPermissions, neededPermissionsGlobal)

	for role, roleAssignments := range assignmentsMap {
		for _, assignment := range roleAssignments {
			neededPermissionsLocal := []int{}
			copy(paramPermissions, neededPermissionsLocal)

			for _, heldPermission := range role.Permissions {
				if idx := slices.Index(neededPermissionsLocal, heldPermission.ID); idx > -1 {
					if assignment.Scope.WorkspaceID.Valid {
						neededPermissionsLocal = append(neededPermissionsLocal[:idx],
							neededPermissionsLocal[idx+1:]...)
					} else if globalIdx := slices.Index(neededPermissionsGlobal,
						heldPermission.ID); globalIdx > -1 {
						neededPermissionsGlobal = append(neededPermissionsGlobal[:globalIdx],
							neededPermissionsGlobal[globalIdx+1:]...)
					}
				}
			}

			if len(neededPermissionsGlobal) == 0 {
				return query, nil
			} else if len(neededPermissionsLocal) == 0 {
				workspaces = append(workspaces, assignment.Scope.WorkspaceID.Int32)
			}
		}
	}

	if len(workspaces) == 0 {
		return query.Where("workspace_id = -1"), nil
	}

	query = query.Where("workspace_id IN (?)", bun.In(workspaces))

	return query, nil
}

// FilterExperimentLabelsQuery filters a query for what experiment metadata a user can view.
func (a *ExperimentAuthZRBAC) FilterExperimentLabelsQuery(
	ctx context.Context, curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
) (selectQuery *bun.SelectQuery, err error) {
	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUser.ID
	fields["permissionRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{
				rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA,
			},
			SubjectType: "experiment",
		},
	}

	defer func() {
		audit.LogFromErr(fields, err)
	}()

	if proj != nil {
		// if proj is not nil, there is already a filter in place
		return query, nil
	}

	assignmentsMap, err := rbac.GetPermissionSummary(ctx, curUser.ID)
	if err != nil {
		return query, err
	}

	var workspaces []int32

	for role, roleAssignments := range assignmentsMap {
		for _, permission := range role.Permissions {
			if permission.ID == int(
				rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA) {
				for _, assignment := range roleAssignments {
					if assignment.Scope.WorkspaceID.Valid {
						workspaces = append(workspaces, assignment.Scope.WorkspaceID.Int32)
					} else {
						// if permission is global, return without filtering
						return query, nil
					}
				}
			}
		}
	}

	if len(workspaces) == 0 {
		return query.Where("project_id = -1"), nil
	}

	var projectIDs []int32
	err = db.Bun().NewRaw("SELECT id FROM projects WHERE workspace_id IN (?)",
		bun.In(workspaces)).Scan(ctx, &projectIDs)
	if err != nil {
		return query, err
	}

	query = query.Where("project_id IN (?)", bun.In(projectIDs))

	return query, nil
}

// CanPreviewHPSearch always returns a nil error.
func (a *ExperimentAuthZRBAC) CanPreviewHPSearch(ctx context.Context, curUser model.User,
) (err error) {
	// TODO: does this require any specific permission if you already have the config?
	// Maybe permission to submit the experiment?
	fields := audit.ExtractLogFields(ctx)
	fields["userID"] = curUser.ID
	fields["permissionsRequired"] = []audit.PermissionWithSubject{
		{
			PermissionTypes: []rbacv1.PermissionType{},
			SubjectType:     "preview HP Search",
		},
	}

	defer func() {
		audit.LogFromErr(fields, err)
	}()

	return nil
}

// CanEditExperiment checks if a user can edit an experiment.
func (a *ExperimentAuthZRBAC) CanEditExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, e, fields, rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
}

// CanEditExperimentsMetadata checks if a user can edit an experiment's metadata.
func (a *ExperimentAuthZRBAC) CanEditExperimentsMetadata(
	ctx context.Context, curUser model.User, e *model.Experiment,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, e, fields, rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA)
}

// CanCreateExperiment checks if a user can create an experiment.
func (a *ExperimentAuthZRBAC) CanCreateExperiment(
	ctx context.Context, curUser model.User, proj *projectv1.Project, e *model.Experiment,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, e, fields, rbacv1.PermissionType_PERMISSION_TYPE_CREATE_EXPERIMENT)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_EXPERIMENT)
}

// CanForkFromExperiment checks if a user can create an experiment.
func (a *ExperimentAuthZRBAC) CanForkFromExperiment(
	ctx context.Context, curUser model.User, e *model.Experiment,
) (err error) {
	fields := audit.ExtractLogFields(ctx)
	addExpInfo(curUser, e, fields, rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
	defer func() {
		audit.LogFromErr(fields, err)
	}()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return db.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
}

// CanSetExperimentsMaxSlots checks if a user can update an experiment's max slots.
func (a *ExperimentAuthZRBAC) CanSetExperimentsMaxSlots(
	ctx context.Context, curUser model.User, e *model.Experiment, slots int,
) error {
	return a.CanEditExperiment(ctx, curUser, e)
}

// CanSetExperimentsWeight checks if a user can update an experiment's weight.
func (a *ExperimentAuthZRBAC) CanSetExperimentsWeight(
	ctx context.Context, curUser model.User, e *model.Experiment, weight float64,
) error {
	return a.CanEditExperiment(ctx, curUser, e)
}

// CanSetExperimentsPriority checks if a user can update an experiment's priority.
func (a *ExperimentAuthZRBAC) CanSetExperimentsPriority(
	ctx context.Context, curUser model.User, e *model.Experiment, priority int,
) error {
	return a.CanEditExperiment(ctx, curUser, e)
}

// CanSetExperimentsCheckpointGCPolicy checks if a user can update the checkpoint gc policy.
func (a *ExperimentAuthZRBAC) CanSetExperimentsCheckpointGCPolicy(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return a.CanEditExperiment(ctx, curUser, e)
}

// CanRunCustomSearch checks if a user has permission to run customer search.
func (a *ExperimentAuthZRBAC) CanRunCustomSearch(
	ctx context.Context, curUser model.User, e *model.Experiment,
) error {
	return a.CanEditExperiment(ctx, curUser, e) // TODO verify with custom search project.
}

func init() {
	AuthZProvider.Register("rbac", &ExperimentAuthZRBAC{})
}
