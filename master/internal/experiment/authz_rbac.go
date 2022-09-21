package experiment

import (
	"context"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rbac"
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

// CanGetExperiment checks if a user has permission to view an experiment.
func (a *ExperimentAuthZRBAC) CanGetExperiment(
	curUser model.User, e *model.Experiment,
) (canGetExp bool, serverError error) {
	ctx := context.Background()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return false, err
	}

	if err = rbac.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA); err != nil {
		if errors.Is(err, grpcutil.ErrPermissionDenied) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CanGetExperimentArtifacts checks if a user has permission to view experiment artifacts.
func (a *ExperimentAuthZRBAC) CanGetExperimentArtifacts(
	curUser model.User, e *model.Experiment,
) error {
	ctx := context.Background()
	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return rbac.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS)
}

// CanDeleteExperiment checks if a user has permission to delete an experiment.
func (a *ExperimentAuthZRBAC) CanDeleteExperiment(curUser model.User, e *model.Experiment) error {
	ctx := context.Background()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return rbac.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT)
}

// FilterExperimentsQuery filters a query for what experiments a user can view.
func (a *ExperimentAuthZRBAC) FilterExperimentsQuery(
	curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	ctx := context.Background()
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
		return query.Where("workspace_id = -1"), nil
	}

	query = query.Where("workspace_id IN (?)", bun.In(workspaces))

	return query, nil
}

// FilterExperimentLabelsQuery filters a query for what experiment metadata a user can view.
func (a *ExperimentAuthZRBAC) FilterExperimentLabelsQuery(
	curUser model.User, proj *projectv1.Project, query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	if proj != nil {
		// if proj is not nil, there is already a filter in place
		return query, nil
	}
	ctx := context.Background()
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
func (a *ExperimentAuthZRBAC) CanPreviewHPSearch(curUser model.User) error {
	// TODO: does this require any specific permission if you already have the config?
	// Maybe permission to submit the experiment?
	return nil
}

// CanEditExperiment checks if a user can edit an experiment.
func (a *ExperimentAuthZRBAC) CanEditExperiment(curUser model.User, e *model.Experiment) error {
	ctx := context.Background()
	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return rbac.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
}

// CanEditExperimentsMetadata checks if a user can edit an experiment's metadata.
func (a *ExperimentAuthZRBAC) CanEditExperimentsMetadata(
	curUser model.User, e *model.Experiment,
) error {
	ctx := context.Background()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return rbac.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA)
}

// CanCreateExperiment checks if a user can create an experiment.
func (a *ExperimentAuthZRBAC) CanCreateExperiment(
	curUser model.User, proj *projectv1.Project, e *model.Experiment,
) error {
	ctx := context.Background()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return rbac.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_CREATE_EXPERIMENT)
}

// CanForkFromExperiment checks if a user can create an experiment.
func (a *ExperimentAuthZRBAC) CanForkFromExperiment(
	curUser model.User, e *model.Experiment,
) error {
	ctx := context.Background()

	workspaceID, err := getWorkspaceFromExperiment(ctx, e)
	if err != nil {
		return err
	}

	return rbac.DoesPermissionMatch(ctx, curUser.ID, &workspaceID,
		rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
}

// CanSetExperimentsMaxSlots checks if a user can update an experiment's max slots.
func (a *ExperimentAuthZRBAC) CanSetExperimentsMaxSlots(
	curUser model.User, e *model.Experiment, slots int,
) error {
	return a.CanEditExperiment(curUser, e)
}

// CanSetExperimentsWeight checks if a user can update an experiment's weight.
func (a *ExperimentAuthZRBAC) CanSetExperimentsWeight(
	curUser model.User, e *model.Experiment, weight float64,
) error {
	return a.CanEditExperiment(curUser, e)
}

// CanSetExperimentsPriority checks if a user can update an experiment's priority.
func (a *ExperimentAuthZRBAC) CanSetExperimentsPriority(
	curUser model.User, e *model.Experiment, priority int,
) error {
	return a.CanEditExperiment(curUser, e)
}

// CanSetExperimentsCheckpointGCPolicy checks if a user can update the checkpoint gc policy.
func (a *ExperimentAuthZRBAC) CanSetExperimentsCheckpointGCPolicy(
	curUser model.User, e *model.Experiment,
) error {
	return a.CanEditExperiment(curUser, e)
}

func init() {
	AuthZProvider.Register("rbac", &ExperimentAuthZRBAC{})
}
