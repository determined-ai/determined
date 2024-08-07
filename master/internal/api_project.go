package internal

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
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

var defaultRunsTableColumns = []*projectv1.ProjectColumn{
	{
		Column:      "id",
		DisplayName: "Global Run ID",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "experimentDescription",
		DisplayName: "Search Description",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "tags",
		DisplayName: "Tags",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "forkedFrom",
		DisplayName: "Forked",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "startTime",
		DisplayName: "Start Time",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_DATE,
	},
	{
		Column:      "endTime",
		DisplayName: "End Time",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_DATE,
	},
	{
		Column:      "duration",
		DisplayName: "Duration",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "state",
		DisplayName: "State",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "searcherType",
		DisplayName: "Searcher",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "resourcePool",
		DisplayName: "Resource Pool",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "checkpointSize",
		DisplayName: "Checkpoint Size",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "checkpointCount",
		DisplayName: "Checkpoints",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "user",
		DisplayName: "User",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "searcherMetric",
		DisplayName: "Searcher Metric",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "searcherMetricsVal",
		DisplayName: "Searcher Metric Value",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "externalExperimentId",
		DisplayName: "External Experiment ID",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "projectId",
		DisplayName: "Project ID",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "externalRunId",
		DisplayName: "External Run ID",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "experimentProgress",
		DisplayName: "Search Progress",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "experimentId",
		DisplayName: "Search ID",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	},
	{
		Column:      "experimentName",
		DisplayName: "Search Name",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
	{
		Column:      "isExpMultitrial",
		DisplayName: "Part of Search",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED,
	},
	{
		Column:      "parentArchived",
		DisplayName: "Parent Archived",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_UNSPECIFIED,
	},
	{
		Column:      "localId",
		DisplayName: "Run ID",
		Location:    projectv1.LocationType_LOCATION_TYPE_RUN,
		Type:        projectv1.ColumnType_COLUMN_TYPE_TEXT,
	},
}

func getRunSummaryMetrics(ctx context.Context, whereClause string, group []int) ([]*projectv1.ProjectColumn, error) {
	var columns []*projectv1.ProjectColumn
	summaryMetrics := []struct {
		MetricName string
		JSONPath   string
		MetricType string
		Count      int32
	}{}

	subQuery := db.BunSelectMetricGroupNames().ColumnExpr(
		`summary_metrics->jsonb_object_keys(summary_metrics)->
	jsonb_object_keys(summary_metrics->jsonb_object_keys(summary_metrics))->>'type'
	AS metric_type`).
		Where(whereClause, bun.In(group)).Distinct()
	trialsQuery := db.Bun().NewSelect().TableExpr("(?) AS stats", subQuery).
		ColumnExpr("*").ColumnExpr(
		"ROW_NUMBER() OVER(PARTITION BY json_path, metric_name order by metric_type) AS count").
		Order("json_path").Order("metric_name")
	err := trialsQuery.Scan(ctx, &summaryMetrics)
	if err != nil {
		return nil, fmt.Errorf("retreiving project summary metrics: %w", err)
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
	return columns, nil
}

func (a *apiServer) GetProjectByKey(
	ctx context.Context,
	req *apiv1.GetProjectByKeyRequest,
) (*apiv1.GetProjectByKeyResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	p, err := project.GetProjectByKey(ctx, req.Key)
	if errors.Is(err, db.ErrNotFound) {
		return nil, api.NotFoundErrs("project", req.Key, true)
	} else if err != nil {
		return nil, err
	}

	protoProject := p.Proto()
	if err := project.AuthZProvider.Get().CanGetProject(ctx, *curUser, protoProject); err != nil {
		return nil, err
	}
	return &apiv1.GetProjectByKeyResponse{Project: protoProject}, nil
}

func (a *apiServer) GetProjectByID(
	ctx context.Context, id int32, curUser model.User,
) (*projectv1.Project, error) {
	notFoundErr := api.NotFoundErrs("project", strconv.Itoa(int(id)), true)
	p, err := project.GetProjectByID(ctx, int(id))
	if errors.Is(err, db.ErrNotFound) {
		return nil, notFoundErr
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching project (%d) from database", id)
	}
	protoProject := p.Proto()
	if err := project.AuthZProvider.Get().CanGetProject(ctx, curUser, protoProject); err != nil {
		return nil, authz.SubIfUnauthorized(err, notFoundErr)
	}
	return protoProject, nil
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
	if len(trialIDs) > 0 {
		summaryMetricColumns, err := getRunSummaryMetrics(ctx, "id IN (?)", trialIDs)
		if err != nil {
			return nil, err
		}
		columns = append(columns, summaryMetricColumns...)
	}
	hparamSet := make(map[string]struct{})
	for _, hparam := range hyperparameters {
		flatHparam := expconf.FlattenHPs(hparam.Hyperparameters)

		// ensure we're iterating in order
		paramKeys := make([]string, 0, len(flatHparam))
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

func (a *apiServer) getProjectRunColumnsByID(
	ctx context.Context, id int32, curUser model.User,
) (*apiv1.GetProjectColumnsResponse, error) {
	p, err := a.GetProjectByID(ctx, id, curUser)
	if err != nil {
		return nil, err
	}

	columns := defaultRunsTableColumns

	hyperparameters := []struct {
		WorkspaceID int
		Hparam      string
		Type        string
	}{}

	// Get all runs in project.
	runsQuery := db.Bun().NewSelect().
		ColumnExpr("?::int as workspace_id", p.WorkspaceId).
		Column("hparam").
		Column("type").
		TableExpr("project_hparams").
		Where("project_id = ?", id)

	runsQuery, err = exputil.AuthZProvider.Get().FilterExperimentsQuery(
		ctx,
		curUser,
		p,
		runsQuery,
		[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA},
	)
	if err != nil {
		return nil, err
	}

	err = runsQuery.Scan(ctx, &hyperparameters)
	if err != nil {
		return nil, err
	}

	// Get summary metrics only if project is not empty.
	summaryMetricColumns, err := getRunSummaryMetrics(ctx, "project_id IN (?)", []int{int(id)})
	if err != nil {
		return nil, err
	}
	columns = append(columns, summaryMetricColumns...)
	for _, hparam := range hyperparameters {
		columns = append(columns, &projectv1.ProjectColumn{
			Column:   fmt.Sprintf("hp.%s", hparam.Hparam),
			Location: projectv1.LocationType_LOCATION_TYPE_RUN_HYPERPARAMETERS,
			Type:     parseMetricsType(hparam.Type),
		})
	}

	metadata := []struct {
		Metadata string
		Type     string
	}{}
	// Get run metadata colums
	err = db.Bun().NewSelect().Distinct().
		ColumnExpr("CONCAT('metadata.', flat_key) as metadata").
		ColumnExpr(`
		CASE
			WHEN is_array_element=true THEN 'array'
			WHEN string_value IS NOT NULL THEN 'string'
			WHEN integer_value IS NOT NULL THEN 'number'
			WHEN float_value IS NOT NULL THEN 'number'
			WHEN timestamp_value IS NOT NULL THEN 'date'
			WHEN boolean_value IS NOT NULL THEN 'boolean'
			ELSE 'string'
		END as type`).
		TableExpr("runs_metadata_index").
		Where("project_id = ?", id).
		Scan(ctx, &metadata)
	if err != nil {
		return nil, err
	}

	for _, data := range metadata {
		columns = append(columns, &projectv1.ProjectColumn{
			Column:   data.Metadata,
			Location: projectv1.LocationType_LOCATION_TYPE_RUN_METADATA,
			Type:     parseMetricsType(data.Type),
		})
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
	case db.MetricTypeArray:
		return projectv1.ColumnType_COLUMN_TYPE_ARRAY
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

	if req.TableType != nil && *req.TableType == apiv1.TableType_TABLE_TYPE_RUN {
		return a.getProjectRunColumnsByID(ctx, req.Id, *curUser)
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
		ctx, p)
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
	ctx context.Context, project *projectv1.Project,
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

	if req.Key != nil {
		// allow user to provide a key, but ensure it is uppercase.
		*req.Key = strings.ToUpper(*req.Key)
		if err = project.ValidateProjectKey(*req.Key); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	p := &model.Project{
		Name:        req.Name,
		Description: req.Description,
		WorkspaceID: int(req.WorkspaceId),
		UserID:      int(curUser.ID),
		Username:    curUser.Username,
	}

	err = project.InsertProject(ctx, p, req.Key)
	err = apiutils.MapAndFilterErrors(err, nil, nil)
	return &apiv1.PostProjectResponse{Project: p.Proto()},
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
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, errors.New("failed to get user while updating project")
	}

	if req.Project == nil {
		return nil, errors.New("project in request is nil while updating project")
	}
	if req.Project.Key != nil {
		err = project.ValidateProjectKey(req.Project.Key.Value)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	updatedProject, err := project.UpdateProject(
		ctx,
		req.Id,
		*curUser,
		req.Project,
	)
	if err != nil && errors.Is(err, db.ErrNotFound) {
		return nil, api.NotFoundErrs("project", strconv.Itoa(int(req.Id)), true)
	} else if err != nil {
		log.WithError(err).Errorf("failed to update project %d", req.Id)
		return nil, err
	}
	return &apiv1.PatchProjectResponse{Project: updatedProject.Proto()}, nil
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
	if err = a.deleteExperiments(expList, user); err != nil {
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

	expList, err := db.ProjectExperiments(context.TODO(), int(req.Id))
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
		(SELECT COUNT(*) FROM runs as r WHERE r.project_id=p.id) as num_runs,
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

func (a *apiServer) GetMetadataValues(
	ctx context.Context, req *apiv1.GetMetadataValuesRequest,
) (*apiv1.GetMetadataValuesResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	p, err := a.GetProjectByID(ctx, req.ProjectId, *curUser)
	if err != nil { // Can view project?
		return nil, err
	}

	// We only want string-type metadata.
	values := []string{}
	err = db.Bun().NewSelect().Distinct().
		Table("runs_metadata_index").
		Column("string_value").
		Where("string_value IS NOT NULL").
		Where("project_id=?", p.Id).
		Where("flat_key=?", req.Key).Scan(ctx, &values)
	if err != nil {
		return nil, err
	}
	resp := apiv1.GetMetadataValuesResponse{
		Values: values,
	}
	return &resp, nil
}
