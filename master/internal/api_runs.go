package internal

import (
	"context"
	"fmt"
	"strings"

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

func (a *apiServer) GetRuns(
	ctx context.Context, req *apiv1.GetRunsRequest,
) (*apiv1.GetRunsResponse, error) {
	resp := &apiv1.GetRunsResponse{Runs: []*trialv1.Run{}}
	query := db.Bun().NewSelect().
		Model(&resp.Runs).
		ModelTableExpr("runs AS r").
		Apply(getRunstColumns)
		// Limit(int(req.Limit)).
		// Scan(ctx)
	// Construct the ordering expression.
	orderColMap := map[apiv1.GetRunsRequest_SortBy]string{
		apiv1.GetRunsRequest_SORT_BY_UNSPECIFIED:      "id",
		apiv1.GetRunsRequest_SORT_BY_ID:               "id",
		apiv1.GetRunsRequest_SORT_BY_DESCRIPTION:      "description",
		apiv1.GetRunsRequest_SORT_BY_NAME:             "name",
		apiv1.GetRunsRequest_SORT_BY_START_TIME:       "e.start_time",
		apiv1.GetRunsRequest_SORT_BY_END_TIME:         "e.end_time",
		apiv1.GetRunsRequest_SORT_BY_STATE:            "r.state",
		apiv1.GetRunsRequest_SORT_BY_PROGRESS:         "COALESCE(progress, 0)",
		apiv1.GetRunsRequest_SORT_BY_USER:             "display_name",
		apiv1.GetRunsRequest_SORT_BY_FORKED_FROM:      "e.parent_id",
		apiv1.GetRunsRequest_SORT_BY_RESOURCE_POOL:    "resource_pool",
		apiv1.GetRunsRequest_SORT_BY_CHECKPOINT_SIZE:  "checkpoint_size",
		apiv1.GetRunsRequest_SORT_BY_CHECKPOINT_COUNT: "checkpoint_count",
		apiv1.GetRunsRequest_SORT_BY_SEARCHER_METRIC_VAL: `(
			SELECT
				searcher_metric_value
			FROM trials t
			WHERE t.id = e.best_trial_id
		) `,
	}
	sortByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC NULLS LAST",
	}
	orderExpr := ""
	switch _, ok := orderColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case orderColMap[req.SortBy] != "id": //nolint:goconst // Not actually the same constant.
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			orderColMap[req.SortBy], sortByMap[req.OrderBy], sortByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", sortByMap[req.OrderBy])
	}
	query = query.OrderExpr(orderExpr)

	// Filtering
	if req.Description != "" {
		query = query.Where("e.config->>'description' ILIKE ('%%' || ? || '%%')", req.Description)
	}
	if req.Name != "" {
		query = query.Where("e.config->>'name' ILIKE ('%%' || ? || '%%')", req.Name)
	}
	if len(req.Labels) > 0 {
		// In the event labels were removed, if all were removed we insert null,
		// which previously broke this query.
		query = query.Where(`string_to_array(?, ',') <@ ARRAY(SELECT jsonb_array_elements_text(
				CASE WHEN e.config->'labels'::text = 'null'
				THEN NULL
				ELSE e.config->'labels' END
			))`, strings.Join(req.Labels, ",")) // Trying bun.In doesn't work.
	}
	if req.Archived != nil {
		query = query.Where("e.archived = ?", req.Archived.Value)
		if req.ProjectId == 0 && !req.Archived.Value {
			query = query.Where("w.archived= ?", req.Archived.Value)
			query = query.Where("p.archived= ?", req.Archived.Value)
		}
	}
	if len(req.States) > 0 {
		// FIXME(DET-9567): the api state parameter and the database state column do not match.
		var allStates []string
		for _, state := range req.States {
			allStates = append(allStates, strings.TrimPrefix(state.String(), "STATE_"))
		}
		query = query.Where("e.state IN (?)", bun.In(allStates))
	}
	if len(req.Users) > 0 {
		query = query.Where("u.username IN (?)", bun.In(req.Users))
	}
	if len(req.UserIds) > 0 {
		query = query.Where("e.owner_id IN (?)", bun.In(req.UserIds))
	}

	if req.RunIdFilter != nil {
		var err error
		query, err = db.ApplyInt32FieldFilter(query, bun.Ident("r.id"), req.RunIdFilter)
		if err != nil {
			return nil, err
		}
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	var proj *projectv1.Project
	if req.ProjectId != 0 {
		proj, err = a.GetProjectByID(ctx, req.ProjectId, *curUser)
		if err != nil {
			return nil, err
		}

		query = query.Where("e.project_id = ?", req.ProjectId)
	}
	if query, err = experiment.AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, proj, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA},
		); err != nil {
		return nil, err
	}

	pagination, err := runPagedBunExperimentsQuery(ctx, query, int(req.Offset), int(req.Limit))
	resp.Pagination = pagination
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func getRunstColumns(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		Column("r.id").
		ColumnExpr("proto_time(e.start_time) AS start_time").
		ColumnExpr("proto_time(e.end_time) AS end_time").
		ColumnExpr(bunutils.ProtoStateDBCaseString(trialv1.State_value, "r.state", "state",
			"STATE_")).
		Column("r.tags").
		Column("r.checkpoint_size").
		Column("r.checkpoint_count").
		Column("r.external_run_id").
		ColumnExpr("r.hparams AS hyperparameters").
		ColumnExpr("r.summary_metrics AS metrics").
		ColumnExpr("CASE WHEN e.parent_id IS NULL THEN NULL ELSE " +
			"json_build_object('value', e.parent_id) END AS forked_from").
		ColumnExpr("CASE WHEN e.progress IS NULL THEN NULL ELSE " +
			"json_build_object('value', e.progress) END AS progress").
		ColumnExpr("e.id AS experiment_id").
		Column("e.owner_id").
		ColumnExpr("e.config->>'description' AS description").
		ColumnExpr("e.config->'resources'->>'resource_pool' AS resource_pool").
		ColumnExpr("e.config->'searcher'->>'name' AS searcher_type").
		ColumnExpr("e.config->'searcher'->>'metric' AS searcher_metric").
		ColumnExpr("e.config->>'name' as experiment_name").
		Join("JOIN experiments AS e ON r.experiment_id=e.id")
}
