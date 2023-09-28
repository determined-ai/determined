package internal

import (
	"context"
	"fmt"
	"sort"

	"github.com/uptrace/bun"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/api/apiutils"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	exputil "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/pkg/mathx"
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
	notFoundErr := api.NotFoundErrs("project", fmt.Sprint(id), true)
	p := &projectv1.Project{}
	if err := a.m.db.QueryProto("get_project", p, id); errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching project (%d) from database", id)
	}

	if err := project.AuthZProvider.Get().CanGetProject(ctx, curUser, p); err != nil {
		return nil, authz.SubIfUnauthorized(err, notFoundErr)
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
			DisplayName: "Searcher",
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
			DisplayName: "Checkpoints",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "user",
			DisplayName: "User",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "searcherMetric",
			DisplayName: "Searcher Metric",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "searcherMetricsVal",
			DisplayName: "Searcher Metric Value",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
		},
		{
			Column:      "externalExperimentId",
			DisplayName: "External Experiment ID",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
		{
			Column:      "externalTrialId",
			DisplayName: "External Trial ID",
			Location:    projectv1.LocationType_LOCATION_TYPE_EXPERIMENT,
			Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
		},
	}

	hyperparameters := []struct {
		WorkspaceID     int
		Hyperparameters expconf.HyperparametersV0
		BestTrialID     *int
	}{}

	// get all experiments in project
	experimentQuery := db.Bun().NewSelect().
		ColumnExpr("?::int as workspace_id", p.WorkspaceId).
		ColumnExpr("config->'hyperparameters' as hyperparameters").
		Column("best_trial_id").
		Table("experiments").
		Where("config->>'hyperparameters' IS NOT NULL").
		Where("project_id = ?", id).
		Order("id")

	experimentQuery, err = exputil.AuthZProvider.Get().FilterExperimentsQuery(
		ctx,
		curUser,
		p,
		db.Bun().NewSelect().TableExpr("(?) AS subq", experimentQuery),
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA},
	)
	if err != nil {
		return nil, err
	}

	err = experimentQuery.Scan(ctx, &hyperparameters)
	if err != nil {
		return nil, err
	}

	trialIDs := make([]int, 0, len(hyperparameters))
	for _, hparam := range hyperparameters {
		if hparam.BestTrialID != nil {
			trialIDs = append(trialIDs, *hparam.BestTrialID)
		}
	}
	summaryMetrics := []struct {
		MetricName string
		JSONPath   string
		MetricType string
		Count      int32
	}{}

	if len(trialIDs) > 0 {
		subQuery := db.BunSelectMetricGroupNames().ColumnExpr(
			`summary_metrics->jsonb_object_keys(summary_metrics)->
		jsonb_object_keys(summary_metrics->jsonb_object_keys(summary_metrics))->>'type'
		AS metric_type`).
			Where("id IN (?)", bun.In(trialIDs)).Distinct()
		trialsQuery := db.Bun().NewSelect().TableExpr("(?) AS stats", subQuery).
			ColumnExpr("*").ColumnExpr(
			"ROW_NUMBER() OVER(PARTITION BY json_path, metric_name order by metric_type) AS count").
			Order("json_path").Order("metric_name")
		err = trialsQuery.Scan(ctx, &summaryMetrics)
		if err != nil {
			return nil, err
		}
	}

	for idx, stats := range summaryMetrics {
		// If there are multiple metrics with the same group and name, report one unspecified column.
		if stats.Count > 1 {
			continue
		}

		columnType := parseMetricsType(stats.MetricType)
		if len(summaryMetrics) > idx+1 && summaryMetrics[idx+1].Count > 1 {
			columnType = projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED
		}

		columnPrefix := stats.JSONPath
		columnLocation := projectv1.LocationType_LOCATION_TYPE_CUSTOM_METRIC
		if stats.JSONPath == metricGroupTraining {
			columnPrefix = metricIDTraining
			columnLocation = projectv1.LocationType_LOCATION_TYPE_TRAINING
		}
		if stats.JSONPath == metricGroupValidation {
			columnPrefix = metricIDValidation
			columnLocation = projectv1.LocationType_LOCATION_TYPE_VALIDATIONS
		}
		// don't surface aggregates that don't make sense for non-numbers
		if columnType == projectv1.ColumnType_COLUMN_TYPE_NUMBER {
			aggregates := []string{"last", "max", "mean", "min"}
			for _, aggregate := range aggregates {
				columns = append(columns, &projectv1.ProjectColumn{
					Column:   fmt.Sprintf("%s.%s.%s", columnPrefix, stats.MetricName, aggregate),
					Location: columnLocation,
					Type:     columnType,
				})
			}
		} else {
			columns = append(columns, &projectv1.ProjectColumn{
				Column:   fmt.Sprintf("%s.%s.last", columnPrefix, stats.MetricName),
				Location: columnLocation,
				Type:     columnType,
			})
		}
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

	return &apiv1.GetProjectColumnsResponse{
		Columns: columns,
	}, nil
}

func parseMetricsType(metricType string) projectv1.ColumnType {
	switch metricType {
	case db.MetricTypeString:
		return projectv1.ColumnType_COLUMN_TYPE_TEXT
	case db.MetricTypeNumber:
		return projectv1.ColumnType_COLUMN_TYPE_NUMBER
	case db.MetricTypeDate:
		return projectv1.ColumnType_COLUMN_TYPE_DATE
	case db.MetricTypeBool:
		return projectv1.ColumnType_COLUMN_TYPE_TEXT
	default:
		// unsure of how to treat arrays/objects/nulls
		return projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED
	}
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

func (a *apiServer) GetProjectNumericMetricsRange(
	ctx context.Context, req *apiv1.GetProjectNumericMetricsRangeRequest,
) (*apiv1.GetProjectNumericMetricsRangeResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	p, err := a.GetProjectByID(ctx, req.Id, *curUser)
	if err != nil {
		return nil, err
	}
	metricsValues, searcherMetricsValue, err := a.getProjectNumericMetricsRange(
		ctx, *curUser, p)
	if err != nil {
		return nil, err
	}

	var ranges []*projectv1.MetricsRange
	for mn, mr := range metricsValues {
		ranges = append(ranges, &projectv1.MetricsRange{
			MetricsName: mn,
			Min:         mathx.Min(mr...),
			Max:         mathx.Max(mr...),
		})
	}

	if len(searcherMetricsValue) > 0 {
		ranges = append(ranges, &projectv1.MetricsRange{
			MetricsName: "searcherMetricsVal",
			Min:         mathx.Min(searcherMetricsValue...),
			Max:         mathx.Max(searcherMetricsValue...),
		})
	}

	return &apiv1.GetProjectNumericMetricsRangeResponse{Ranges: ranges}, nil
}

func (a *apiServer) getProjectNumericMetricsRange(
	ctx context.Context, curUser model.User, project *projectv1.Project,
) (map[string]([]float64), []float64, error) {
	query := db.Bun().NewSelect().Table("trials").Table("experiments").
		ColumnExpr(`searcher_metric_value_signed = searcher_metric_value AS smaller_is_better`).
		Column("searcher_metric_value").
		Column("summary_metrics").
		Where("summary_metrics IS NOT NULL").
		Where("project_id = ?", project.Id).
		Where("experiments.best_trial_id = trials.id")

	type metrics struct {
		Min  interface{}
		Max  interface{}
		Last interface{}
		Mean interface{}
		Type interface{}
	}

	var res []struct {
		ID                  int32
		SmallerIsBetter     bool
		SearcherMetricValue *float64
		SummaryMetrics      map[string](map[string]metrics)
	}

	if err := query.Scan(ctx, &res); err != nil {
		return nil, nil, errors.Wrapf(
			err, "error fetching metrics range for project (%d) from database", project.Id)
	}
	metricsValues := make(map[string]([]float64))
	searcherMetricsValue := []float64{}
	for _, r := range res {
		if r.SearcherMetricValue != nil {
			searcherMetricsValue = append(searcherMetricsValue, *r.SearcherMetricValue)
		}
		if r.SummaryMetrics != nil {
			for metricsGroup, metrics := range r.SummaryMetrics {
				for name, value := range metrics {
					if value.Type != "number" {
						continue
					}

					for _, aggregate := range SummaryMetricStatistics {
						group := metricsGroup
						if metricsGroup == metricGroupTraining {
							group = metricIDTraining
						}
						if metricsGroup == metricGroupValidation {
							group = metricIDValidation
						}
						tMetricsName := fmt.Sprintf("%s.%s.%s", group, name, aggregate)
						var tMetricsValue interface{}
						switch aggregate {
						case "last":
							tMetricsValue = value.Last
						case "max":
							tMetricsValue = value.Max
						case "min":
							tMetricsValue = value.Min
						case "mean":
							tMetricsValue = value.Mean
						}
						switch v := tMetricsValue.(type) {
						case float64:
							metricsValues[tMetricsName] = append(metricsValues[tMetricsName], v)
						case *int32:
							metricsValues[tMetricsName] = append(metricsValues[tMetricsName], float64(*v))
						}
					}
				}
			}
		}
	}
	return metricsValues, searcherMetricsValue, nil
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
		return nil, errors.Errorf("project (%d) is archived and cannot have attributes updated",
			currProject.Id)
	}
	if currProject.Immutable {
		return nil, errors.Errorf("project (%v) is immutable and cannot have attributes updated",
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
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		log.WithError(err).Errorf("failed to access user and delete project %d", projectID)
		_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
		return err
	}

	log.Debugf("deleting project %d experiments", projectID)
	if _, err = a.deleteExperiments(expList, user); err != nil {
		log.WithError(err).Errorf("failed to delete experiments")
		_ = a.m.db.QueryProto("delete_fail_project", holder, projectID, err.Error())
		return err
	}
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
		err := project.AuthZProvider.Get().CanGetProject(ctx, *curUser, pr)
		if err == nil {
			viewableProjects = append(viewableProjects, pr)
			// omit projects user doesn't have access to
		} else if !authz.IsPermissionDenied(err) {
			return nil, err
		}
	}

	return &apiv1.GetProjectsByUserActivityResponse{Projects: viewableProjects}, nil
}
