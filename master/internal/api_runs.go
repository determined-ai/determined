package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"strings"

	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/db/bunutils"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/storage"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/runv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

func (a *apiServer) RunPrepareForReporting(
	ctx context.Context, req *apiv1.RunPrepareForReportingRequest,
) (*apiv1.RunPrepareForReportingResponse, error) {
	// TODO(runs) run specific RBAC.
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.RunId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	var storageID *int32
	if req.CheckpointStorage != nil {
		bytes, err := req.CheckpointStorage.MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshaling checkpoint storage %+v: %w", req.CheckpointStorage, err)
		}
		cs := &expconf.CheckpointStorageConfig{} //nolint:exhaustruct
		if err := cs.UnmarshalJSON(bytes); err != nil {
			return nil, fmt.Errorf("unmarshaling json bytes %s: %w", string(bytes), err)
		}

		id, err := storage.AddBackend(ctx, cs)
		if err != nil {
			return nil, fmt.Errorf("adding storage ID for runID %d: %w", req.RunId, err)
		}
		storageID = ptrs.Ptr(int32(id))
	}

	return &apiv1.RunPrepareForReportingResponse{
		StorageId: storageID,
	}, nil
}

func (a *apiServer) SearchRuns(
	ctx context.Context, req *apiv1.SearchRunsRequest,
) (*apiv1.SearchRunsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	resp := &apiv1.SearchRunsResponse{}
	var runs []*runv1.FlatRun
	query := db.Bun().NewSelect().
		Model(&runs).
		ModelTableExpr("runs AS r").
		Apply(getRunsColumns)

	var proj *projectv1.Project
	if req.ProjectId != nil {
		proj, err = a.GetProjectByID(ctx, *req.ProjectId, *curUser)
		if err != nil {
			return nil, err
		}

		query = query.Where("r.project_id = ?", req.ProjectId)
	}
	if query, err = experiment.AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, proj, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA},
		); err != nil {
		return nil, err
	}

	if req.Filter != nil {
		var efr experimentFilterRoot
		err := json.Unmarshal([]byte(*req.Filter), &efr)
		if err != nil {
			return nil, err
		}
		query = query.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			_, err = efr.toSQL(q)
			return q
		}).WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			if !efr.ShowArchived {
				return q.Where(`e.archived = false`)
			}
			return q
		})
		if err != nil {
			return nil, err
		}
	}

	if req.Sort != nil {
		err = sortRuns(req.Sort, query)
		if err != nil {
			return nil, err
		}
	} else {
		query.OrderExpr("id ASC")
	}

	pagination, err := runPagedBunExperimentsQuery(ctx, query, int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, err
	}
	resp.Pagination = pagination
	resp.Runs = runs
	return resp, nil
}

func getRunsColumns(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		Column("r.id").
		ColumnExpr("proto_time(r.start_time) AS start_time").
		ColumnExpr("proto_time(r.end_time) AS end_time").
		ColumnExpr(bunutils.ProtoStateDBCaseString(trialv1.State_value, "r.state", "state",
			"STATE_")).
		Column("r.checkpoint_size").
		Column("r.checkpoint_count").
		Column("r.external_run_id").
		Column("r.project_id").
		ColumnExpr("extract(epoch FROM coalesce(r.end_time, now()) - r.start_time)::int AS duration").
		ColumnExpr("CASE WHEN r.hparams='null' THEN NULL ELSE r.hparams END AS hyperparameters").
		ColumnExpr("r.summary_metrics AS summary_metrics").
		ColumnExpr("e.owner_id AS user_id").
		ColumnExpr("e.config->>'labels' AS labels").
		ColumnExpr("w.id AS workspace_id").
		ColumnExpr("w.name AS workspace_name").
		ColumnExpr("(w.archived OR p.archived) AS parent_archived").
		ColumnExpr("p.name AS project_name").
		ColumnExpr(`jsonb_build_object(
			'searcher_type', e.config->'searcher'->>'name',
			'searcher_metric', e.config->'metric'->>'name',
			'resource_pool', e.config->'resources'->>'resource_pool',
			'name', e.config->>'name',
			'description', e.config->>'description',
			'unmanaged', e.unmanaged,
			'progress', e.progress,
			'forked_from', e.parent_id,
			'external_experiment_id', e.external_experiment_id,
			'is_multitrial', ((SELECT COUNT(*) FROM runs r WHERE e.id = r.experiment_id) > 1),
			'id', e.id) AS experiment`).
		Join("LEFT JOIN experiments AS e ON r.experiment_id=e.id").
		Join("LEFT JOIN users u ON e.owner_id = u.id").
		Join("LEFT JOIN projects p ON r.project_id = p.id").
		Join("LEFT JOIN workspaces w ON p.workspace_id = w.id")
}

func sortRuns(sortString *string, runQuery *bun.SelectQuery) error {
	if sortString == nil {
		return nil
	}
	sortByMap := map[string]string{
		"asc":  "ASC",
		"desc": "DESC NULLS LAST",
	}
	orderColMap := map[string]string{
		"id":                    "id",
		"experimentDescription": "experiment_description",
		"experimentName":        "experiment_name",
		"searcherType":          "searcher_type",
		"searcherMetric":        "searcher_metric",
		"startTime":             "r.start_time",
		"endTime":               "r.end_time",
		"state":                 "r.state",
		"experimentProgress":    "COALESCE(progress, 0)",
		"user":                  "display_name",
		"forkedFrom":            "e.parent_id",
		"resourcePool":          "resource_pool",
		"projectId":             "r.project_id",
		"checkpointSize":        "checkpoint_size",
		"checkpointCount":       "checkpoint_count",
		"duration":              "duration",
		"searcherMetricsVal":    "r.searcher_metric_val",
		"externalExperimentId":  "e.external_experiment_id",
		"externalRunId":         "r.external_run_id",
		"experimentId":          "e.id",
		"isExpMultitrial":       "is_exp_multitrial",
	}
	sortParams := strings.Split(*sortString, ",")
	hasIDSort := false
	for _, sortParam := range sortParams {
		paramDetail := strings.Split(sortParam, "=")
		if len(paramDetail) != 2 {
			return status.Errorf(codes.InvalidArgument, "invalid sort parameter: %s", sortParam)
		}
		if _, ok := sortByMap[paramDetail[1]]; !ok {
			return status.Errorf(codes.InvalidArgument, "invalid sort direction: %s", paramDetail[1])
		}
		sortDirection := sortByMap[paramDetail[1]]
		switch {
		case strings.HasPrefix(paramDetail[0], "hp."):
			param := strings.ReplaceAll(paramDetail[0], "'", "")
			hp := strings.Split(strings.TrimPrefix(param, "hp."), ".")
			var queryArgs []interface{}
			for i := 0; i < len(hp); i++ {
				queryArgs = append(queryArgs, hp[i])
				hp[i] = "?"
			}
			hpQuery := strings.Join(hp, "->")
			queryArgs = append(queryArgs, bun.Safe(sortDirection))
			runQuery.OrderExpr(fmt.Sprintf(`r.hparams->%s ?`, hpQuery), queryArgs...)
		case strings.Contains(paramDetail[0], "."):
			metricGroup, metricName, metricQualifier, err := parseMetricsName(paramDetail[0])
			if err != nil {
				return err
			}
			runQuery.OrderExpr("r.summary_metrics->?->?->>? ?",
				metricGroup, metricName, metricQualifier, bun.Safe(sortDirection))
		default:
			if _, ok := orderColMap[paramDetail[0]]; !ok {
				return status.Errorf(codes.InvalidArgument, "invalid sort col: %s", paramDetail[0])
			}
			hasIDSort = hasIDSort || paramDetail[0] == "id"
			runQuery.OrderExpr(
				fmt.Sprintf("%s %s", orderColMap[paramDetail[0]], sortDirection))
		}
	}
	if !hasIDSort {
		runQuery.OrderExpr("id ASC")
	}
	return nil
}
