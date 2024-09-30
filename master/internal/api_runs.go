package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/db/bunutils"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/run"
	"github.com/determined-ai/determined/master/internal/storage"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/runv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

type runCandidateResult struct {
	Archived     bool
	ExpArchived  bool
	ID           int32
	ExpID        *int32
	IsMultitrial bool
	IsTerminal   bool
}

func (a *apiServer) RunPrepareForReporting(
	ctx context.Context, req *apiv1.RunPrepareForReportingRequest,
) (*apiv1.RunPrepareForReportingResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// TODO(runs) run specific RBAC.
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.RunId), curUser,
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
		query, err = filterRunQuery(query, req.Filter)
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
		Column("r.searcher_metric_value").
		ColumnExpr("COALESCE((r.archived OR e.archived), FALSE) AS archived").
		ColumnExpr("CONCAT(p.key, '-' , r.local_id::text) as local_id").
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
			'searcher_metric', e.config->'searcher'->>'metric',
			'resource_pool', e.config->'resources'->>'resource_pool',
			'name', e.config->>'name',
			'description', e.config->>'description',
			'unmanaged', e.unmanaged,
			'progress', e.progress,
			'forked_from', e.parent_id,
			'external_experiment_id', e.external_experiment_id,
			'is_multitrial', (e.config->'searcher'->>'name' != 'single'),
			'pachyderm_integration', NULLIF(e.config#>'{integrations,pachyderm}', 'null'),
			'id', e.id) AS experiment`).
		ColumnExpr("rm.metadata AS metadata").
		ColumnExpr("r.log_signal AS log_signal").
		Join("LEFT JOIN experiments AS e ON r.experiment_id=e.id").
		Join("LEFT JOIN runs_metadata AS rm ON r.id=rm.run_id").
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
		"experimentDescription": "e.config->>'description'",
		"experimentName":        "e.config->>'name'",
		"searcherType":          "e.config->'searcher'->>'name'",
		"searcherMetric":        "e.config->'searcher'->>'metric'",
		"startTime":             "r.start_time",
		"endTime":               "r.end_time",
		"state":                 "r.state",
		"experimentProgress":    "COALESCE(e.progress, 0)",
		"user":                  "COALESCE(u.username, u.display_name)",
		"forkedFrom":            "e.parent_id",
		"resourcePool":          "e.config->'resources'->>'resource_pool'",
		"projectId":             "r.project_id",
		"checkpointSize":        "checkpoint_size",
		"checkpointCount":       "checkpoint_count",
		"duration":              "duration",
		"searcherMetricsVal":    "r.searcher_metric_value",
		"externalExperimentId":  "e.external_experiment_id",
		"externalRunId":         "r.external_run_id",
		"experimentId":          "e.id",
		"isExpMultitrial":       "(e.config->'searcher'->>'name' != 'single')",
		"parentArchived":        "(w.archived OR p.archived)",
		"localId":               "r.local_id",
		"archived":              "COALESCE((r.archived OR e.archived), FALSE)",
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
		case strings.HasPrefix(paramDetail[0], "metadata."):
			param := strings.ReplaceAll(paramDetail[0], "'", "")
			mdt := strings.Split(strings.TrimPrefix(param, "metadata."), ".")
			var queryArgs []interface{}
			for i := 0; i < len(mdt); i++ {
				queryArgs = append(queryArgs, mdt[i])
				mdt[i] = "?"
			}
			mdtQuery := strings.Join(mdt, "->")
			queryArgs = append(queryArgs, bun.Safe(sortDirection))
			runQuery.OrderExpr(fmt.Sprintf(`rm.metadata->%s ?`, mdtQuery), queryArgs...)
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

func filterRunQuery(getQ *bun.SelectQuery, filter *string) (*bun.SelectQuery, error) {
	var efr experimentFilterRoot
	err := json.Unmarshal([]byte(*filter), &efr)
	if err != nil {
		return nil, err
	}
	getQ = getQ.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		_, err = efr.toSQL(q)
		return q
	}).WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
		if !efr.ShowArchived {
			return q.Where(`NOT (r.archived OR e.archived)`)
		}
		return q
	})
	if err != nil {
		return nil, err
	}
	return getQ, nil
}

func getSelectRunsQueryTables() *bun.SelectQuery {
	return db.Bun().NewSelect().
		ModelTableExpr("runs AS r").
		Join("LEFT JOIN experiments e ON r.experiment_id=e.id").
		Join("JOIN projects p ON r.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id")
}

func (a *apiServer) MoveRuns(
	ctx context.Context, req *apiv1.MoveRunsRequest,
) (*apiv1.MoveRunsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// check that user can view source project
	srcProject, err := a.GetProjectByID(ctx, req.SourceProjectId, *curUser)
	if err != nil {
		return nil, err
	}
	if srcProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot have runs moved from it",
			srcProject.Id)
	}

	// check suitable destination project
	destProject, err := a.GetProjectByID(ctx, req.DestinationProjectId, *curUser)
	if err != nil {
		return nil, err
	}
	if destProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot add new runs",
			req.DestinationProjectId)
	}
	if err = experiment.AuthZProvider.Get().CanCreateExperiment(ctx, *curUser, destProject); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	if req.SourceProjectId == req.DestinationProjectId {
		return &apiv1.MoveRunsResponse{Results: []*apiv1.RunActionResult{}}, nil
	}

	var runChecks []runCandidateResult
	getQ := db.Bun().NewSelect().
		ModelTableExpr("runs AS r").
		Model(&runChecks).
		Column("r.id").
		ColumnExpr("COALESCE((r.archived OR e.archived OR p.archived OR w.archived), FALSE) AS archived").
		ColumnExpr("r.experiment_id as exp_id").
		ColumnExpr("(e.config->'searcher'->>'name' != 'single') as is_multitrial").
		Join("LEFT JOIN experiments e ON r.experiment_id=e.id").
		Join("JOIN projects p ON r.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id").
		Where("r.project_id = ?", req.SourceProjectId)

	if req.Filter == nil {
		getQ = getQ.Where("r.id IN (?)", bun.In(req.RunIds))
	} else {
		getQ, err = filterRunQuery(getQ, req.Filter)
		if err != nil {
			return nil, err
		}
	}

	if getQ, err = experiment.AuthZProvider.Get().FilterExperimentsQuery(ctx, *curUser, nil, getQ,
		[]rbacv1.PermissionType{
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA,
			rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT,
		}); err != nil {
		return nil, err
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.RunActionResult
	visibleIDs := set.New[int32]()
	var validIDs []int32
	// associated experiments to move
	var expMoveIds []int32
	for _, check := range runChecks {
		visibleIDs.Insert(check.ID)
		if check.Archived {
			results = append(results, &apiv1.RunActionResult{
				Error: "Run is archived.",
				Id:    check.ID,
			})
			continue
		}
		if check.IsMultitrial && req.SkipMultitrial {
			results = append(results, &apiv1.RunActionResult{
				Error: fmt.Sprintf("Skipping run '%d' (part of multi-trial).", check.ID),
				Id:    check.ID,
			})
			continue
		}
		if check.ExpID != nil {
			expMoveIds = append(expMoveIds, *check.ExpID)
		}
		validIDs = append(validIDs, check.ID)
	}
	if req.Filter == nil {
		for _, originalID := range req.RunIds {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.RunActionResult{
					Error: fmt.Sprintf("Run with id '%d' not found in project with id '%d'", originalID, req.SourceProjectId),
					Id:    originalID,
				})
			}
		}
	}
	if len(validIDs) > 0 {
		expMoveResults, err := experiment.MoveExperiments(ctx, srcProject.Id, expMoveIds, nil, req.DestinationProjectId)
		if err != nil {
			return nil, err
		}
		tx, err := db.Bun().BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer func() {
			txErr := tx.Rollback()
			if txErr != nil && txErr != sql.ErrTxDone {
				log.WithError(txErr).Error("error rolling back transaction in MoveRuns")
			}
		}()
		failedExpMoveIds := []int32{-1}
		successExpMoveIds := []int32{-1}
		for _, res := range expMoveResults {
			if res.Error != nil {
				failedExpMoveIds = append(failedExpMoveIds, res.ID)
			} else {
				successExpMoveIds = append(successExpMoveIds, res.ID)
			}
		}
		var acceptedIDs []int32
		if _, err = tx.NewUpdate().Table("runs").
			Set("project_id = ?", req.DestinationProjectId).
			Where("runs.id IN (?)", bun.In(validIDs)).
			Where("runs.experiment_id NOT IN (?)", bun.In(failedExpMoveIds)).
			Where("runs.experiment_id NOT IN (?)", bun.In(successExpMoveIds)).
			Returning("runs.id").
			Model(&acceptedIDs).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("updating run's project IDs: %w", err)
		}

		for _, acceptID := range acceptedIDs {
			results = append(results, &apiv1.RunActionResult{
				Error: "",
				Id:    acceptID,
			})
		}

		if err = db.AddProjectHparams(ctx, tx, int(req.DestinationProjectId), acceptedIDs); err != nil {
			return nil, err
		}
		if err = db.RemoveOutdatedProjectHparams(ctx, tx, int(req.SourceProjectId)); err != nil {
			return nil, err
		}

		if _, err = tx.NewRaw(`
		UPDATE runs SET local_id=s.local_id
		FROM
			(
				SELECT
					r.id as id,
					(p.max_local_id + ROW_NUMBER() OVER(PARTITION BY p.id ORDER BY r.id)) as local_id
				FROM
					projects p
					JOIN runs r ON r.project_id=p.id
				WHERE r.id IN (?)
			) as s 
		WHERE s.id=runs.id
		`,
			bun.In(acceptedIDs)).Exec(ctx); err != nil {
			return nil, fmt.Errorf("updating run's local IDs: %w", err)
		}

		if _, err = tx.NewRaw(`
		UPDATE projects SET max_local_id=s.max_local_id
		FROM 
			(
				SELECT 
					project_id,
					COALESCE(MAX(local_id), 1) as max_local_id
				FROM 
					runs
				GROUP BY
					project_id
				HAVING project_id=?
			) as s
		WHERE projects.id=?
		`, req.DestinationProjectId, req.DestinationProjectId).Exec(ctx); err != nil {
			return nil, fmt.Errorf("updating projects max local id: %w", err)
		}

		if _, err = tx.NewRaw(`
		INSERT INTO local_id_redirect (run_id, project_id, project_key, local_id)
		SELECT 
			r.id as runs_id,
			p.id as project_id,
			p.key as project_key,
			r.local_id
		FROM 
			projects p
			JOIN runs r 
			ON r.project_id=p.id
		WHERE r.id IN (?)
		`, bun.In(acceptedIDs)).Exec(ctx); err != nil {
			return nil, fmt.Errorf("adding local id redirect: %w", err)
		}

		var failedRunIDs []int32
		if err = tx.NewSelect().Table("runs").
			Column("id").
			Where("runs.id IN (?)", bun.In(validIDs)).
			Where("runs.experiment_id IN (?)", bun.In(failedExpMoveIds)).
			Scan(ctx, &failedRunIDs); err != nil {
			return nil, fmt.Errorf("getting failed experiment move run IDs: %w", err)
		}
		for _, failedRunID := range failedRunIDs {
			results = append(results, &apiv1.RunActionResult{
				Error: "Failed to move associated experiment",
				Id:    failedRunID,
			})
		}

		var successRunIDs []int32
		if err = tx.NewSelect().Table("runs").
			Column("id").
			Where("runs.experiment_id IN (?)", bun.In(successExpMoveIds)).
			Scan(ctx, &successRunIDs); err != nil {
			return nil, fmt.Errorf("getting failed experiment move run IDs: %w", err)
		}
		for _, successRunID := range successRunIDs {
			results = append(results, &apiv1.RunActionResult{
				Error: "",
				Id:    successRunID,
			})
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
	}
	return &apiv1.MoveRunsResponse{Results: results}, nil
}

func (a *apiServer) KillRuns(ctx context.Context, req *apiv1.KillRunsRequest,
) (*apiv1.KillRunsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	type killRunOKResult struct {
		ID         int32
		RequestID  *string
		IsTerminal bool
	}

	var killCandidatees []killRunOKResult
	getQ := getSelectRunsQueryTables().
		Model(&killCandidatees).
		Join("LEFT JOIN trials_v2 t ON r.id=t.run_id").
		Column("r.id").
		ColumnExpr("t.request_id").
		ColumnExpr("r.state IN (?) AS is_terminal", bun.In(model.StatesToStrings(model.TerminalStates))).
		Where("r.project_id = ?", req.ProjectId)

	if req.Filter == nil {
		getQ = getQ.Where("r.id IN (?)", bun.In(req.RunIds))
	} else {
		getQ, err = filterRunQuery(getQ, req.Filter)
		if err != nil {
			return nil, err
		}
	}

	if getQ, err = experiment.AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, nil, getQ,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT}); err != nil {
		return nil, err
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.RunActionResult
	visibleIDs := set.New[int32]()
	var validIDs []int32
	for _, cand := range killCandidatees {
		visibleIDs.Insert(cand.ID)
		switch {
		case cand.IsTerminal:
			results = append(results, &apiv1.RunActionResult{
				Error: "",
				Id:    cand.ID,
			})
		// This should be impossible in the current system but we will leave this check here
		// to cover a possible error in integration tests
		case cand.RequestID == nil:
			results = append(results, &apiv1.RunActionResult{
				Error: "Run has no associated request id.",
				Id:    cand.ID,
			})
		default:
			validIDs = append(validIDs, cand.ID)
		}
	}
	if req.Filter == nil {
		for _, originalID := range req.RunIds {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.RunActionResult{
					Error: fmt.Sprintf("Run with id '%d' not found", originalID),
					Id:    originalID,
				})
			}
		}
	}

	for _, runID := range validIDs {
		_, err = a.KillTrial(ctx, &apiv1.KillTrialRequest{
			Id: runID,
		})
		if err != nil {
			results = append(results, &apiv1.RunActionResult{
				Error: fmt.Sprintf("Failed to kill run: %s", err),
				Id:    runID,
			})
		} else {
			results = append(results, &apiv1.RunActionResult{
				Error: "",
				Id:    runID,
			})
		}
	}
	return &apiv1.KillRunsResponse{Results: results}, nil
}

func (a *apiServer) DeleteRuns(ctx context.Context, req *apiv1.DeleteRunsRequest,
) (*apiv1.DeleteRunsResponse, error) {
	if len(req.RunIds) > 0 && req.Filter != nil {
		return nil, fmt.Errorf("if filter is provided run id list must be empty")
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// get runs to delete
	var deleteCandidates []runCandidateResult
	getQ := getSelectRunsQueryTables().
		Model(&deleteCandidates).
		Column("r.id").
		ColumnExpr("COALESCE((r.archived OR e.archived OR p.archived OR w.archived), FALSE) AS archived").
		ColumnExpr("r.experiment_id as exp_id").
		ColumnExpr("(e.config->'searcher'->>'name' != 'single') as is_multitrial").
		ColumnExpr("r.state IN (?) AS is_terminal", bun.In(model.StatesToStrings(model.TerminalStates))).
		Where("r.project_id = ?", req.ProjectId)

	if req.Filter == nil {
		getQ = getQ.Where("r.id IN (?)", bun.In(req.RunIds))
	} else {
		getQ, err = filterRunQuery(getQ, req.Filter)
		if err != nil {
			return nil, err
		}
	}

	if getQ, err = experiment.AuthZProvider.Get().FilterExperimentsQuery(ctx, *curUser, nil, getQ,
		[]rbacv1.PermissionType{
			rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT,
		}); err != nil {
		return nil, err
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.RunActionResult
	visibleIDs := set.New[int32]()
	var validIDs []int32
	for _, cand := range deleteCandidates {
		visibleIDs.Insert(cand.ID)
		switch {
		case !cand.IsTerminal:
			results = append(results, &apiv1.RunActionResult{
				Error: "Run is not in a terminal state.",
				Id:    cand.ID,
			})
		default:
			validIDs = append(validIDs, cand.ID)
		}
	}
	if req.Filter == nil {
		for _, originalID := range req.RunIds {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.RunActionResult{
					Error: fmt.Sprintf("Run with id '%d' not found in project with id '%d'", originalID, req.ProjectId),
					Id:    originalID,
				})
			}
		}
	}
	if len(validIDs) == 0 {
		return &apiv1.DeleteRunsResponse{Results: results}, nil
	}
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		txErr := tx.Rollback()
		if txErr != nil && txErr != sql.ErrTxDone {
			log.WithError(txErr).Error("error rolling back transaction in DeleteRuns")
		}
	}()

	// delete run logs
	if _, err = tx.NewDelete().Table("trial_logs").
		Where("trial_logs.trial_id IN (?)", bun.In(validIDs)).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete run logs: %w", err)
	}

	// delete task logs
	trialTaskQuery := tx.NewSelect().Table("run_id_task_id").
		ColumnExpr("task_id").
		Where("run_id IN (?)", bun.In(validIDs))
	if _, err = tx.NewDelete().Table("task_logs").
		Where("task_logs.task_id IN (?)", trialTaskQuery).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete runs: %w", err)
	}

	var acceptedIDs []int
	if _, err = tx.NewDelete().Table("runs").
		Where("runs.id IN (?)", bun.In(validIDs)).
		Returning("runs.id").
		Model(&acceptedIDs).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("delete runs: %w", err)
	}

	for _, acceptID := range acceptedIDs {
		results = append(results, &apiv1.RunActionResult{
			Error: "",
			Id:    int32(acceptID),
		})
	}

	// delete run hparams
	if _, err = tx.NewDelete().Table("run_hparams").
		Where("run_id IN (?)", bun.In(acceptedIDs)).
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("deleting run hparams: %w", err)
	}
	// remove project hparams
	if err = db.RemoveOutdatedProjectHparams(ctx, tx, int(req.ProjectId)); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &apiv1.DeleteRunsResponse{Results: results}, nil
}

func (a *apiServer) ArchiveRuns(
	ctx context.Context, req *apiv1.ArchiveRunsRequest,
) (*apiv1.ArchiveRunsResponse, error) {
	results, err := archiveUnarchiveAction(ctx, true, req.RunIds, req.ProjectId, req.Filter)
	if err != nil {
		return nil, err
	}
	return &apiv1.ArchiveRunsResponse{Results: results}, nil
}

func (a *apiServer) UnarchiveRuns(
	ctx context.Context, req *apiv1.UnarchiveRunsRequest,
) (*apiv1.UnarchiveRunsResponse, error) {
	results, err := archiveUnarchiveAction(ctx, false, req.RunIds, req.ProjectId, req.Filter)
	if err != nil {
		return nil, err
	}
	return &apiv1.UnarchiveRunsResponse{Results: results}, nil
}

func archiveUnarchiveAction(ctx context.Context, archive bool, runIDs []int32,
	projectID int32, filter *string,
) ([]*apiv1.RunActionResult, error) {
	if len(runIDs) > 0 && filter != nil {
		return nil, fmt.Errorf("if filter is provided run id list must be empty")
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	var runCandidates []runCandidateResult
	query := db.Bun().NewSelect().
		ModelTableExpr("runs AS r").
		Model(&runCandidates).
		Column("r.id").
		Column("r.archived").
		ColumnExpr("e.archived AS exp_archived").
		ColumnExpr("r.experiment_id as exp_id").
		ColumnExpr("false as is_multitrial").
		ColumnExpr("r.state IN (?) AS is_terminal", bun.In(model.StatesToStrings(model.TerminalStates))).
		Join("LEFT JOIN experiments e ON r.experiment_id=e.id").
		Join("JOIN projects p ON r.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id").
		Where("r.project_id = ?", projectID)

	if filter == nil {
		query = query.Where("r.id IN (?)", bun.In(runIDs))
	} else {
		query, err = filterRunQuery(query, filter)
		if err != nil {
			return nil, err
		}
	}

	query, err = experiment.AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, nil, query,
			[]rbacv1.PermissionType{rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT_METADATA})
	if err != nil {
		return nil, err
	}

	err = query.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.RunActionResult
	visibleIDs := set.New[int32]()
	var validIDs []int32
	for _, cand := range runCandidates {
		visibleIDs.Insert(cand.ID)
		switch {
		case cand.ExpArchived:
			results = append(results, &apiv1.RunActionResult{
				Error: fmt.Sprintf("Run is part of archived Search (id: '%d').", *cand.ExpID),
				Id:    cand.ID,
			})
		case cand.Archived && archive:
			results = append(results, &apiv1.RunActionResult{
				Error: "",
				Id:    cand.ID,
			})
		case !cand.Archived && !archive:
			results = append(results, &apiv1.RunActionResult{
				Error: "",
				Id:    cand.ID,
			})
		case !cand.IsTerminal:
			results = append(results, &apiv1.RunActionResult{
				Error: "Run is not in terminal state.",
				Id:    cand.ID,
			})
		default:
			validIDs = append(validIDs, cand.ID)
		}
	}

	if filter == nil {
		for _, originalID := range runIDs {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.RunActionResult{
					Error: fmt.Sprintf("Run with id '%d' not found in project with id '%d'", originalID, projectID),
					Id:    originalID,
				})
			}
		}
	}

	if len(validIDs) > 0 {
		var acceptedIDs []int32
		_, err = db.Bun().NewUpdate().
			ModelTableExpr("runs as r").
			Set("archived = ?", archive).
			Where("id IN (?)", bun.In(validIDs)).
			Returning("id").
			Model(&acceptedIDs).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to archive/unarchive runs: %w", err)
		}
		for _, acceptID := range acceptedIDs {
			results = append(results, &apiv1.RunActionResult{
				Error: "",
				Id:    acceptID,
			})
		}
	}

	return results, nil
}

func (a *apiServer) PauseRuns(ctx context.Context, req *apiv1.PauseRunsRequest,
) (*apiv1.PauseRunsResponse, error) {
	results, err := pauseResumeAction(ctx, true, req.ProjectId, req.RunIds, req.Filter)
	if err != nil {
		return nil, err
	}
	return &apiv1.PauseRunsResponse{Results: results}, nil
}

func (a *apiServer) ResumeRuns(ctx context.Context, req *apiv1.ResumeRunsRequest,
) (*apiv1.ResumeRunsResponse, error) {
	results, err := pauseResumeAction(ctx, false, req.ProjectId, req.RunIds, req.Filter)
	if err != nil {
		return nil, err
	}
	return &apiv1.ResumeRunsResponse{Results: results}, nil
}

func pauseResumeAction(ctx context.Context, isPause bool, projectID int32,
	runIds []int32, filter *string) (
	[]*apiv1.RunActionResult, error,
) {
	if len(runIds) > 0 && filter != nil {
		return nil, fmt.Errorf("if filter is provided run id list must be empty")
	}
	// Get experiment ids
	var err error
	var runCandidates []runCandidateResult
	isRunIDAction := (len(runIds) > 0)
	getQ := db.Bun().NewSelect().
		ModelTableExpr("runs AS r").
		Model(&runCandidates).
		Column("r.id").
		ColumnExpr("COALESCE((r.archived OR e.archived OR p.archived OR w.archived), FALSE) AS archived").
		ColumnExpr("r.experiment_id as exp_id").
		ColumnExpr("(e.config->'searcher'->>'name' != 'single') as is_multitrial").
		Join("LEFT JOIN experiments e ON r.experiment_id=e.id").
		Join("JOIN projects p ON r.project_id = p.id").
		Join("JOIN workspaces w ON p.workspace_id = w.id").
		Where("r.project_id = ?", projectID)

	if isRunIDAction {
		getQ = getQ.Where("r.id IN (?)", bun.In(runIds))
	} else {
		getQ, err = filterRunQuery(getQ, filter)
		if err != nil {
			return nil, err
		}
	}

	err = getQ.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var results []*apiv1.RunActionResult
	visibleIDs := set.New[int32]()
	expIDs := set.New[int32]()
	expToRun := make(map[int32]int32)
	for _, cand := range runCandidates {
		visibleIDs.Insert(cand.ID)
		if cand.Archived {
			results = append(results, &apiv1.RunActionResult{
				Error: "Run is archived.",
				Id:    cand.ID,
			})
			continue
		}
		if cand.IsMultitrial {
			results = append(results, &apiv1.RunActionResult{
				Error: fmt.Sprintf("Cannot pause/unpause run '%d' (part of multi-trial).", cand.ID),
				Id:    cand.ID,
			})
			continue
		}
		if cand.ExpID == nil {
			results = append(results, &apiv1.RunActionResult{
				Error: fmt.Sprintf("Cannot pause run '%d' (no associated experiment).", cand.ID),
				Id:    cand.ID,
			})
			continue
		}
		expToRun[*cand.ExpID] = cand.ID
		expIDs.Insert(*cand.ExpID)
	}
	if isRunIDAction {
		for _, originalID := range runIds {
			if !visibleIDs.Contains(originalID) {
				results = append(results, &apiv1.RunActionResult{
					Error: fmt.Sprintf("Run with id '%d' not found in project with id '%d'", originalID, projectID),
					Id:    originalID,
				})
			}
		}
	}
	// Pause/Resume experiments
	var expResults []experiment.ExperimentActionResult
	var errMsg string
	if isPause {
		expResults, err = experiment.PauseExperiments(ctx, projectID, expIDs.ToSlice(), nil)
		errMsg = "Failed to pause associated experiment: %s"
	} else {
		expResults, err = experiment.ActivateExperiments(ctx, projectID, expIDs.ToSlice(), nil)
		errMsg = "Failed to resume associated experiment: %s"
	}
	if err != nil {
		return nil, err
	}
	for _, expRes := range expResults {
		val, ok := expToRun[expRes.ID]
		if !ok {
			results = append(results, &apiv1.RunActionResult{
				Error: fmt.Sprintf("Unexpected action performed on experiment '%d'", expRes.ID),
				Id:    -1,
			})
		}

		if expRes.Error != nil {
			results = append(results, &apiv1.RunActionResult{
				Error: fmt.Sprintf(errMsg, expRes.Error),
				Id:    val,
			})
		} else {
			results = append(results, &apiv1.RunActionResult{
				Error: "",
				Id:    val,
			})
		}
	}
	return results, nil
}

// GetRunMetadata returns the metadata of a run.
func (a *apiServer) GetRunMetadata(
	ctx context.Context, req *apiv1.GetRunMetadataRequest,
) (*apiv1.GetRunMetadataResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// TODO(runs) run specific RBAC.
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.RunId), curUser,
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}
	metadata, err := db.GetRunMetadata(ctx, int(req.RunId))
	if err != nil {
		return nil, err
	}
	return &apiv1.GetRunMetadataResponse{
		Metadata: protoutils.ToStruct(metadata),
	}, nil
}

// PostRunMetadata updates the metadata of a run.
func (a *apiServer) PostRunMetadata(
	ctx context.Context, req *apiv1.PostRunMetadataRequest,
) (*apiv1.PostRunMetadataResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// TODO(runs) run specific RBAC.
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.RunId), curUser,
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	// Flatten Request Metadata.
	flatMetadata, err := run.FlattenMetadata(req.Metadata.AsMap())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Update the metadata in DB
	result, err := db.UpdateRunMetadata(ctx,
		int(req.RunId),
		req.Metadata.AsMap(),
		flatMetadata,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "error updating metadata on run(%d)", req.RunId)
	}
	return &apiv1.PostRunMetadataResponse{Metadata: protoutils.ToStruct(result)}, nil
}
