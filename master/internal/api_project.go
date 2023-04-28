package internal

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/db"
	exputil "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func (a *apiServer) GetProjectByID(
	ctx context.Context, id int32, curUser model.User,
) (*projectv1.Project, error) {
	notFoundErr := status.Errorf(codes.NotFound, "project (%d) not found", id)
	p := &projectv1.Project{}
	if err := a.m.db.QueryProto("get_project", p, id); errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching project (%d) from database", id)
	}

	if ok, err := project.AuthZProvider.Get().CanGetProject(ctx, curUser, p); err != nil {
		return nil, err
	} else if !ok {
		return nil, notFoundErr
	}
	return p, nil
}

func (a *apiServer) getProjectColumnsByID(
	ctx context.Context, id int32, curUser model.User,
) (*apiv1.GetProjectColumnsResponse, error) {
	p, err := a.GetProjectByID(ctx, id, curUser)
	if err != nil {
		return nil, err
	}

	columns := []*projectv1.ProjectColumn{
		{
			Column:      "id",
			DisplayName: "ID",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "name",
			DisplayName: "Name",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "description",
			DisplayName: "Description",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "tags",
			DisplayName: "Tags",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "forkedFrom",
			DisplayName: "Forked",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "startTime",
			DisplayName: "Start time",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_DATE,
		},
		{
			Column:      "duration",
			DisplayName: "Duration",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "numTrials",
			DisplayName: "Trial count",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "state",
			DisplayName: "State",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "searcherType",
			DisplayName: "Searcher type",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "resourcePool",
			DisplayName: "Resource pool",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "progress",
			DisplayName: "Progress",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "checkpointSize",
			DisplayName: "Checkpoint size",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "checkpointCount",
			DisplayName: "Checkpoint count",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "user",
			DisplayName: "User",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
	}

	hyperparameters := []struct {
		WorkspaceID     int
		Hyperparameters expconf.HyperparametersV0
	}{}

	// get all experiments in project
	experimentQuery := db.Bun().NewSelect().
		ColumnExpr("?::int as workspace_id", p.WorkspaceId).
		ColumnExpr("config->'hyperparameters' as hyperparameters").
		TableExpr("experiments").
		Where("config->>'hyperparameters' IS NOT NULL").
		Where("project_id = ?", id).
		Order("id")

	experimentQuery, err = exputil.AuthZProvider.Get().FilterExperimentsQuery(
		ctx,
		curUser,
		p,
		experimentQuery,
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA},
	)
	if err != nil {
		return nil, err
	}

	err = experimentQuery.Scan(ctx, &hyperparameters)
	if err != nil {
		return nil, err
	}
	hparamSet := make(map[string]struct{})
	for _, hparam := range hyperparameters {
		flatHparam := expconf.FlattenHPs(hparam.Hyperparameters)

		// ensure we're iterating in order
		paramKeys := make([]string, len(flatHparam))
		for key := range flatHparam {
			paramKeys = append(paramKeys, key)
		}
		sort.Strings(paramKeys)

		for _, key := range paramKeys {
			value := flatHparam[key]
			_, seen := hparamSet[key]
			if !seen {
				hparamSet[key] = struct{}{}
				var columnType projectv1.ColumnType
				switch {
				case value.RawIntHyperparameter != nil ||
					value.RawDoubleHyperparameter != nil ||
					value.RawLogHyperparameter != nil:
					columnType = projectv1.ColumnType_COLUMN_TYPE_NUMBER
				case value.RawConstHyperparameter != nil:
					switch value.RawConstHyperparameter.RawVal.(type) {
					case float64:
						columnType = projectv1.ColumnType_COLUMN_TYPE_NUMBER
					case string:
						columnType = projectv1.ColumnType_COLUMN_TYPE_TEXT
					default:
						columnType = projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED
					}
				default:
					columnType = projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED
				}
				columns = append(columns, &projectv1.ProjectColumn{
					Column:   fmt.Sprintf("hp.%s", key),
					Location: projectv1.LocationType_LOCATION_TYPE_HYPERPARAMETERS,
					Type:     columnType,
				})
			}
		}
	}

	// Get metrics columns
	metricNames := []struct {
		Vname       []string
		WorkspaceID int
	}{}
	metricQuery := db.Bun().
		NewSelect().
		TableExpr("exp_metrics_name").
		TableExpr("LATERAL json_array_elements_text(vname) AS vnames").
		ColumnExpr("array_to_json(array_agg(DISTINCT vnames)) AS vname").
		ColumnExpr("?::int as workspace_id", p.WorkspaceId).
		Where("project_id = ?", id)

	metricQuery, err = exputil.AuthZProvider.Get().FilterExperimentsQuery(
		ctx,
		curUser,
		p,
		metricQuery,
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_ARTIFACTS},
	)
	if err != nil {
		return nil, err
	}

	err = metricQuery.Scan(ctx, &metricNames)
	if err != nil {
		return nil, err
	}
	for _, mn := range metricNames {
		for _, mnv := range mn.Vname {
			columns = append(columns, &projectv1.ProjectColumn{
				Column:   fmt.Sprintf("validation.%s", mnv),
				Location: projectv1.LocationType_LOCATION_TYPE_VALIDATIONS,
				Type:     projectv1.ColumnType_COLUMN_TYPE_NUMBER,
			})
		}
	}

	return &apiv1.GetProjectColumnsResponse{
		Columns: columns,
	}, nil
}

func (a *apiServer) getProjectAndCheckCanDoActions(
	ctx context.Context, projectID int32,
	canDoActions ...func(context.Context, model.User, *projectv1.Project) error,
) (*projectv1.Project, model.User, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, model.User{}, err
	}
	p, err := a.GetProjectByID(ctx, projectID, *curUser)
	if err != nil {
		return nil, model.User{}, err
	}

	for _, canDoAction := range canDoActions {
		if err = canDoAction(ctx, *curUser, p); err != nil {
			return nil, model.User{}, status.Error(codes.PermissionDenied, err.Error())
		}
	}
	return p, *curUser, nil
}

func (a *apiServer) CheckParentWorkspaceUnarchived(project *projectv1.Project) error {
	w := &workspacev1.Workspace{}
	err := a.m.db.QueryProto("get_workspace_from_project", w, project.Id)
	if err != nil {
		return errors.Wrapf(err,
			"error fetching project (%v)'s workspace from database", project.Id)
	}

	if w.Archived {
		return errors.Errorf("This project belongs to an archived workspace. " +
			"To make changes, first unarchive the workspace.")
	}
	return nil
}

func (a *apiServer) GetProject(
	ctx context.Context, req *apiv1.GetProjectRequest,
) (*apiv1.GetProjectResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	p, err := a.GetProjectByID(ctx, req.Id, *curUser)
	return &apiv1.GetProjectResponse{Project: p}, err
}

func (a *apiServer) GetProjectColumns(
	ctx context.Context, req *apiv1.GetProjectColumnsRequest,
) (*apiv1.GetProjectColumnsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	return a.getProjectColumnsByID(ctx, req.Id, *curUser)
}

func (a *apiServer) PostProject(
	ctx context.Context, req *apiv1.PostProjectRequest,
) (*apiv1.PostProjectResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	w, err := a.GetWorkspaceByID(ctx, req.WorkspaceId, *curUser, true)
	if err != nil {
		return nil, err
	}
	if err = project.AuthZProvider.Get().CanCreateProject(ctx, *curUser, w); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	p := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project", p, req.Name, req.Description,
		req.WorkspaceId, curUser.ID)

	return &apiv1.PostProjectResponse{Project: p},
		errors.Wrapf(err, "error creating project %s in database", req.Name)
}

func (a *apiServer) AddProjectNote(
	ctx context.Context, req *apiv1.AddProjectNoteRequest,
) (*apiv1.AddProjectNoteResponse, error) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId,
		project.AuthZProvider.Get().CanSetProjectNotes)
	if err != nil {
		return nil, err
	}

	notes := p.Notes
	notes = append(notes, &projectv1.Note{
		Name:     req.Note.Name,
		Contents: req.Note.Contents,
	})

	newp := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project_note", newp, req.ProjectId, notes)
	return &apiv1.AddProjectNoteResponse{Notes: newp.Notes},
		errors.Wrapf(err, "error adding project note")
}

func (a *apiServer) PutProjectNotes(
	ctx context.Context, req *apiv1.PutProjectNotesRequest,
) (*apiv1.PutProjectNotesResponse, error) {
	_, _, err := a.getProjectAndCheckCanDoActions(ctx, req.ProjectId,
		project.AuthZProvider.Get().CanSetProjectNotes)
	if err != nil {
		return nil, err
	}

	newp := &projectv1.Project{}
	err = a.m.db.QueryProto("insert_project_note", newp, req.ProjectId, req.Notes)
	return &apiv1.PutProjectNotesResponse{Notes: newp.Notes},
		errors.Wrapf(err, "error putting project notes")
}

func (a *apiServer) PatchProject(
	ctx context.Context, req *apiv1.PatchProjectRequest,
) (*apiv1.PatchProjectResponse, error) {
	currProject, currUser, err := a.getProjectAndCheckCanDoActions(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if currProject.Archived {
		return nil, errors.Errorf("project (%d) is archived and cannot have attributes updated.",
			currProject.Id)
	}
	if currProject.Immutable {
		return nil, errors.Errorf("project (%v) is immutable and cannot have attributes updated.",
			currProject.Id)
	}

	madeChanges := false
	if req.Project.Name != nil && req.Project.Name.Value != currProject.Name {
		if err = project.AuthZProvider.Get().CanSetProjectName(ctx, currUser, currProject); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		log.Infof("project (%d) name changing from \"%s\" to \"%s\"",
			currProject.Id, currProject.Name, req.Project.Name.Value)
		madeChanges = true
		currProject.Name = req.Project.Name.Value
	}

	if req.Project.Description != nil && req.Project.Description.Value != currProject.Description {
		if err = project.AuthZProvider.Get().
			CanSetProjectDescription(ctx, currUser, currProject); err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		log.Infof("project (%d) description changing from \"%s\" to \"%s\"",
			currProject.Id, currProject.Description, req.Project.Description.Value)
		madeChanges = true
		currProject.Description = req.Project.Description.Value
	}

	if !madeChanges {
		return &apiv1.PatchProjectResponse{Project: currProject}, nil
	}

	finalProject := &projectv1.Project{}
	err = a.m.db.QueryProto("update_project",
		finalProject, currProject.Id, currProject.Name, currProject.Description)

	return &apiv1.PatchProjectResponse{Project: finalProject},
		errors.Wrapf(err, "error updating project (%d) in database", currProject.Id)
}

func (a *apiServer) deleteProject(ctx context.Context, projectID int32,
	expList []*model.Experiment,
) (err error) {
	holder := &projectv1.Project{}
	_, _, err = grpcutil.GetUser(ctx)
	if err != nil {
		log.WithError(err).Errorf("failed to access user and delete project %d", projectID)
		_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
		return err
	}

	log.Debugf("deleting project %d experiments", projectID)
	// sema := make(chan struct{}, 1)
	// if _, err = a.deleteExperiments(expList, user, sema); err != nil {
	// 	log.WithError(err).Errorf("failed to delete experiments")
	// 	_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
	// 	return err
	// }
	log.Debugf("project %d experiments deleted successfully", projectID)
	err = a.m.db.QueryProto("delete_project", holder, projectID)
	if err != nil {
		log.WithError(err).Errorf("failed to delete project %d", projectID)
		_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
		return err
	}
	log.Debugf("project %d deleted successfully", projectID)
	return nil
}

func (a *apiServer) DeleteProject(
	ctx context.Context, req *apiv1.DeleteProjectRequest) (*apiv1.DeleteProjectResponse,
	error,
) {
	_, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanDeleteProject)
	if err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("deletable_project", holder, req.Id)
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) does not exist or not deletable by this user",
			req.Id)
	}

	expList, err := a.m.db.ProjectExperiments(int(req.Id))
	if err != nil {
		return nil, err
	}

	if len(expList) == 0 {
		err = a.m.db.QueryProto("delete_project", holder, req.Id)
		return &apiv1.DeleteProjectResponse{Completed: (err == nil)},
			errors.Wrapf(err, "error deleting project (%d)", req.Id)
	}
	go func() {
		_ = a.deleteProject(ctx, req.Id, expList)
	}()
	return &apiv1.DeleteProjectResponse{Completed: false},
		errors.Wrapf(err, "error deleting project (%d)", req.Id)
}

func (a *apiServer) MoveProject(
	ctx context.Context, req *apiv1.MoveProjectRequest) (*apiv1.MoveProjectResponse,
	error,
) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	p, err := a.GetProjectByID(ctx, req.ProjectId, *curUser)
	if err != nil { // Can view project?
		return nil, err
	}
	// Allow projects to be moved from immutable workspaces but not to immutable workspaces.
	from, err := a.GetWorkspaceByID(ctx, p.WorkspaceId, *curUser, false)
	if err != nil {
		return nil, err
	}
	to, err := a.GetWorkspaceByID(ctx, req.DestinationWorkspaceId, *curUser, true)
	if err != nil {
		return nil, err
	}
	if err = project.AuthZProvider.Get().CanMoveProject(ctx, *curUser, p, from, to); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	holder := &projectv1.Project{}
	err = a.m.db.QueryProto("move_project", holder, req.ProjectId, req.DestinationWorkspaceId)
	if err != nil {
		return nil, errors.Wrapf(err, "error moving project (%d)", req.ProjectId)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) does not exist or not moveable by this user",
			req.ProjectId)
	}

	return &apiv1.MoveProjectResponse{}, nil
}

func (a *apiServer) ArchiveProject(
	ctx context.Context, req *apiv1.ArchiveProjectRequest) (*apiv1.ArchiveProjectResponse,
	error,
) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanArchiveProject)
	if err != nil {
		return nil, err
	}
	if err = a.CheckParentWorkspaceUnarchived(p); err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	if err = a.m.db.QueryProto("archive_project", holder, req.Id, true); err != nil {
		return nil, errors.Wrapf(err, "error archiving project (%d)", req.Id)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) is not archive-able by this user",
			req.Id)
	}

	return &apiv1.ArchiveProjectResponse{}, nil
}

func (a *apiServer) UnarchiveProject(
	ctx context.Context, req *apiv1.UnarchiveProjectRequest) (*apiv1.UnarchiveProjectResponse,
	error,
) {
	p, _, err := a.getProjectAndCheckCanDoActions(ctx, req.Id,
		project.AuthZProvider.Get().CanUnarchiveProject)
	if err != nil {
		return nil, err
	}
	if err = a.CheckParentWorkspaceUnarchived(p); err != nil {
		return nil, err
	}

	holder := &projectv1.Project{}
	if err = a.m.db.QueryProto("archive_project", holder, req.Id, false); err != nil {
		return nil, errors.Wrapf(err, "error unarchiving project (%d)", req.Id)
	}
	if holder.Id == 0 {
		return nil, errors.Wrapf(err, "project (%d) is not unarchive-able by this user",
			req.Id)
	}
	return &apiv1.UnarchiveProjectResponse{}, nil
}

func (a *apiServer) GetProjectsByUserActivity(
	ctx context.Context, req *apiv1.GetProjectsByUserActivityRequest,
) (*apiv1.GetProjectsByUserActivityResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	p := []*model.Project{}

	limit := req.Limit

	if limit > apiutils.MaxLimit {
		return nil, apiutils.ErrInvalidLimit
	}

	err = db.Bun().NewSelect().Model(p).NewRaw(`
	SELECT
		w.name AS workspace_name,
		u.username,
		p.id,
		p.name,
		p.archived,
		p.workspace_id,
		p.description,
		p.immutable,
		p.notes,
		p.user_id,
		'WORKSPACE_STATE_' || p.state AS state,
		p.error_message,
		COUNT(*) FILTER (WHERE e.project_id = p.id) AS num_experiments,
		COUNT(*) FILTER (WHERE e.project_id = p.id AND e.state = 'ACTIVE') AS num_active_experiments,
		MAX(e.start_time) FILTER (WHERE e.project_id = p.id) AS last_experiment_started_at
	FROM
		projects AS p
		INNER JOIN activity AS a ON p.id = a.entity_id AND a.user_id = ?
		LEFT JOIN users AS u ON u.id = p.user_id
		LEFT JOIN workspaces AS w ON w.id = p.workspace_id
		LEFT JOIN experiments AS e ON e.project_id = p.id
	GROUP BY
		p.id,
		u.username,
		w.name,
		a.activity_time
	ORDER BY
		a.activity_time DESC NULLS LAST
	LIMIT ?;
	`, curUser.ID, limit).
		Scan(ctx, &p)
	if err != nil {
		return nil, err
	}

	projects := model.ProjectsToProto(p)
	viewableProjects := []*projectv1.Project{}

	for _, pr := range projects {
		canView, err := project.AuthZProvider.Get().CanGetProject(ctx, *curUser, pr)
		if err != nil {
			return nil, err
		}
		if canView {
			viewableProjects = append(viewableProjects, pr)
		}
	}

	return &apiv1.GetProjectsByUserActivityResponse{Projects: viewableProjects}, nil
}
