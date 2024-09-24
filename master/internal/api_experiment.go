package internal

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	structpbmap "google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/db/bunutils"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoless"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/metricv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// Catches information on active running experiments.
type experimentAllocation struct {
	Job      model.JobID
	Pulling  bool
	Running  bool
	Starting bool
}

// SummaryMetricStatistics lists values possibly queryable within summary metrics.
var SummaryMetricStatistics = []string{"last", "max", "mean", "min"}

const maxConcurrentDeletes = 10

func (a *apiServer) enrichExperimentState(experiments ...*experimentv1.Experiment) error {
	return a.enrichExperimentStateTx(context.Background(), db.Bun(), experiments...)
}

// Enrich one or more experiments by converting Active state to Queued/Pulling/Starting/Running.
func (a *apiServer) enrichExperimentStateTx(
	ctx context.Context, idb bun.IDB, experiments ...*experimentv1.Experiment,
) error {
	// filter allocations by JobIDs on this page of experiments
	jobFilter := make([]string, 0, len(experiments))
	for _, exp := range experiments {
		jobFilter = append(jobFilter, exp.JobId)
	}

	// get active experiments by JobID
	tasks := []experimentAllocation{}
	query := `
	SELECT
		j.job_id AS job,
		BOOL_OR(CASE WHEN a.state = 'PULLING' THEN true ELSE false END) AS pulling,
		BOOL_OR(CASE WHEN a.state = 'STARTING' THEN true ELSE false END) AS starting,
		BOOL_OR(CASE WHEN a.state = 'RUNNING' THEN true ELSE false END) AS running
	FROM
		jobs j
		JOIN tasks t ON t.job_id = j.job_id
		JOIN allocations a ON a.task_id = t.task_id
	WHERE j.job_id in (SELECT unnest(string_to_array(?, ',')))
	GROUP BY j.job_id
	`
	err := db.MatchSentinelError(idb.NewRaw(query, strings.Join(jobFilter, ",")).Scan(ctx, &tasks))
	if err != nil {
		return err
	}

	// Collect state information by JobID
	byJobID := make(map[model.JobID]experimentv1.State, len(tasks))
	for _, task := range tasks {
		switch {
		case task.Running:
			byJobID[task.Job] = experimentv1.State_STATE_RUNNING
		case task.Starting:
			byJobID[task.Job] = model.MostProgressedExperimentState(byJobID[task.Job],
				experimentv1.State_STATE_STARTING)
		case task.Pulling:
			byJobID[task.Job] = model.MostProgressedExperimentState(byJobID[task.Job],
				experimentv1.State_STATE_PULLING)
		default:
			byJobID[task.Job] = model.MostProgressedExperimentState(byJobID[task.Job],
				experimentv1.State_STATE_QUEUED)
		}
	}

	// Active experiments converted to Queued, Pulling, Starting, or Running
	for _, exp := range experiments {
		if exp.State == experimentv1.State_STATE_ACTIVE {
			if setState, ok := byJobID[model.JobID(exp.JobId)]; ok {
				exp.State = setState
			} else {
				exp.State = experimentv1.State_STATE_QUEUED
			}
		}
	}
	return nil
}

// Return if experiment state is Active or any of its sub-states.
func isActiveExperimentState(state experimentv1.State) bool {
	return slices.Contains([]experimentv1.State{
		experimentv1.State_STATE_ACTIVE,
		experimentv1.State_STATE_PULLING, experimentv1.State_STATE_QUEUED,
		experimentv1.State_STATE_RUNNING, experimentv1.State_STATE_STARTING,
	}, state)
}

// Return a single experiment with enriched state, if the user can access it.
func (a *apiServer) getExperiment(
	ctx context.Context, curUser model.User, experimentID int,
) (*experimentv1.Experiment, error) {
	return a.getExperimentTx(ctx, db.Bun(), curUser, experimentID)
}

// Return a single experiment with enriched state, if the user can access it.
func (a *apiServer) getExperimentTx(
	ctx context.Context, idb bun.IDB, curUser model.User, experimentID int,
) (*experimentv1.Experiment, error) {
	expNotFound := api.NotFoundErrs("experiment", strconv.Itoa(experimentID), true)
	exp := &experimentv1.Experiment{}
	expMap := map[string]interface{}{}
	query := `
	WITH trial_ids AS (
		SELECT id
		FROM trials
		WHERE experiment_id = ?
		ORDER BY id
	)
	SELECT
		e.id AS id,
		e.original_config AS original_config,
		e.config AS config,
		e.config->>'name' AS name,
		e.config->>'description' AS description,
		e.config->'labels' AS labels,
		e.config->'resources'->>'resource_pool' as resource_pool,
		e.config->'searcher'->'name' as searcher_type,
		e.notes AS notes,
		to_json(e.start_time)#>>'{}' AS start_time,
		to_json(e.end_time)#>>'{}' AS end_time,
		'STATE_' || e.state AS state,
		e.archived AS archived,
		e.progress AS progress,
		e.job_id AS job_id,
		e.parent_id AS forked_from,
		e.owner_id AS user_id,
		e.checkpoint_size AS checkpoint_size,
		e.checkpoint_count AS checkpoint_count,
		u.username AS username,
		(SELECT json_agg(id) FROM trial_ids) AS trial_ids,
		(SELECT count(id) FROM trial_ids) AS num_trials,
		p.id AS project_id,
		p.name AS project_name,
		p.user_id AS project_owner_id,
		w.id AS workspace_id,
		w.name AS workspace_name,
		(w.archived OR p.archived) AS parent_archived,
		e.unmanaged AS unmanaged,
		length(e.model_definition) AS model_definition_size,
		NULLIF(e.config#>'{integrations, pachyderm}', 'null') AS pachyderm_integration
	FROM
		experiments e
	JOIN users u ON e.owner_id = u.id
	LEFT JOIN projects p ON e.project_id = p.id
	LEFT JOIN workspaces w ON p.workspace_id = w.id
	WHERE e.id = ?
	`
	err := db.MatchSentinelError(idb.NewRaw(query, experimentID, experimentID).Scan(ctx, &expMap))
	if errors.Is(err, db.ErrNotFound) {
		return nil, expNotFound
	} else if err != nil {
		return nil, errors.Wrapf(err, "error fetching experiment from database: %d", experimentID)
	}
	// Cast string -> []byte `ParseMapToProto` magic.
	jsonFields := []string{"config", "trial_ids", "labels", "pachyderm_integration"}
	for _, field := range jsonFields {
		if sVal, ok := expMap[field].(string); ok {
			expMap[field] = []byte(sVal)
		}
	}
	if err := db.ParseMapToProto(expMap, exp); err != nil {
		return nil, fmt.Errorf("failed to parse map into proto: %w", err)
	}

	modelExp, err := model.ExperimentFromProto(exp)
	if err != nil {
		return nil, err
	}
	if authErr := experiment.AuthZProvider.Get().
		CanGetExperiment(ctx, curUser, modelExp); authErr != nil {
		return nil, authz.SubIfUnauthorized(authErr, expNotFound)
	}

	if err = a.enrichExperimentStateTx(ctx, idb, exp); err != nil {
		return nil, err
	}

	if err = authz.ObfuscateExperiments(exp); err != nil {
		return nil, err
	}

	return exp, nil
}

func (a *apiServer) getExperimentAndCheckCanDoActions(
	ctx context.Context,
	expID int,
	actions ...func(context.Context, model.User, *model.Experiment) error,
) (*model.Experiment, model.User, error) {
	return experiment.GetExperimentAndCheckCanDoActions(ctx, expID, actions...)
}

func (a *apiServer) GetSearcherEvents(
	ctx context.Context, req *apiv1.GetSearcherEventsRequest,
) (*apiv1.GetSearcherEventsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	exp, err := a.getExperiment(ctx, *curUser, int(req.ExperimentId))
	if err != nil {
		return nil, err
	}
	if !isActiveExperimentState(exp.State) {
		return &apiv1.GetSearcherEventsResponse{
			SearcherEvents: []*experimentv1.SearcherEvent{{
				Id: -1,
				Event: &experimentv1.SearcherEvent_ExperimentInactive{
					ExperimentInactive: &experimentv1.ExperimentInactive{
						ExperimentState: exp.State,
					},
				},
			}},
		}, nil
	}

	e, ok := experiment.ExperimentRegistry.Load(int(req.ExperimentId))
	if !ok {
		return nil, api.NotFoundErrs("experiment", strconv.Itoa(int(req.ExperimentId)), true)
	}
	w, err := e.GetSearcherEventsWatcher()
	if err != nil {
		return nil, status.Errorf(codes.Internal,
			"failed to get searcher events: long polling %v", err)
	}
	defer func() {
		if err := e.UnwatchEvents(w.ID); err != nil {
			log.WithError(err).Errorf("error unwatching searcher events")
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(60)*time.Second)
	defer cancel()

	select {
	case events := <-w.C:
		return &apiv1.GetSearcherEventsResponse{
			SearcherEvents: events,
		}, nil
	case <-ctx.Done():
		return &apiv1.GetSearcherEventsResponse{
			SearcherEvents: nil,
		}, nil
	}
}

func (a *apiServer) PostSearcherOperations(
	ctx context.Context,
	req *apiv1.PostSearcherOperationsRequest,
) (
	resp *apiv1.PostSearcherOperationsResponse, err error,
) {
	_, _, err = a.getExperimentAndCheckCanDoActions(
		ctx, int(req.ExperimentId), experiment.AuthZProvider.Get().CanRunCustomSearch,
	)
	if err != nil {
		return nil, errors.Wrap(err, "fetching experiment from database")
	}

	e, ok := experiment.ExperimentRegistry.Load(int(req.ExperimentId))
	if !ok {
		return nil, api.NotFoundErrs("experiment", strconv.Itoa(int(req.ExperimentId)), true)
	}
	if err := e.PerformSearcherOperations(req); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to post operations: %v", err)
	}

	log.Infof("posted operations %v", req.SearcherOperations)
	return &apiv1.PostSearcherOperationsResponse{}, nil
}

func (a *apiServer) GetExperiment(
	ctx context.Context, req *apiv1.GetExperimentRequest,
) (*apiv1.GetExperimentResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	exp, err := a.getExperiment(ctx, *user, int(req.ExperimentId))
	if err != nil {
		return nil, err
	}

	// Update this when we remove the proto type.
	resp := apiv1.GetExperimentResponse{
		Experiment: exp,
		Config:     exp.Config, //nolint:staticcheck
	}

	// Only continue to add a job summary if it's an active experiment.
	if !isActiveExperimentState(exp.State) {
		return &resp, nil
	}

	jobID := model.JobID(exp.JobId)
	jobSummary, err := jobservice.DefaultService.GetJobSummary(jobID, rm.ResourcePoolName(exp.ResourcePool))
	if err != nil {
		// An error here either is real or just that the experiment was not yet terminal in the DB
		// when we first queried it but was by the time it got around to handling out ask. We can't
		// just refresh our DB state to see which it was, since there is a time between an actor
		// closing and PostStop (where the DB state is set) being received where the actor may not
		// respond but still is not terminal -- more clearly, there is a time where the actor is
		// truly non-terminal and not reachable. We _could_ await its stop and recheck, but it's not
		// easy deducible how long that would block. So the best we can really do is return without
		// an error if we're in this case and log. This is a debug log because of how often the
		// happens when polling for an experiment to end.
		if !strings.Contains(err.Error(), sproto.ErrJobNotFound(jobID).Error()) {
			return nil, err
		}
		log.WithError(err).Debugf("asking for job summary")
	} else {
		resp.JobSummary = jobSummary
	}

	return &resp, nil
}

func (a *apiServer) DeleteExperiment(
	ctx context.Context, req *apiv1.DeleteExperimentRequest,
) (*apiv1.DeleteExperimentResponse, error) {
	e, curUser, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		experiment.AuthZProvider.Get().CanDeleteExperiment)
	if err != nil {
		return nil, err
	}

	results, _, err := experiment.DeleteExperiments(
		ctx,
		experiment.GlobalProjectID,
		[]int32{req.ExperimentId},
		nil,
	)
	// report error from the multi-experiment selection code
	if err != nil {
		return nil, err
	}

	// report any error on the individual experiment
	if len(results) == 0 {
		return nil, errors.Errorf("DeleteExperiments returned neither pass nor fail on delete query")
	}
	if results[0].Error != nil {
		return nil, results[0].Error
	}

	go func() {
		if err := a.deleteExperiments([]*model.Experiment{e}, &curUser); err != nil {
			log.WithError(err).Errorf("deleting experiment %d", e.ID)
			e.State = model.DeleteFailedState
			if err := a.m.db.SaveExperimentState(e); err != nil {
				log.WithError(err).Errorf("transitioning experiment %d to %s", e.ID, e.State)
			}
		} else {
			log.Infof("experiment %d deleted successfully", e.ID)
		}
	}()

	return &apiv1.DeleteExperimentResponse{}, nil
}

func (a *apiServer) DeleteExperiments(
	ctx context.Context, req *apiv1.DeleteExperimentsRequest,
) (*apiv1.DeleteExperimentsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	results, experiments, err := experiment.DeleteExperiments(ctx, req.ProjectId, req.ExperimentIds, req.Filters)
	if err != nil {
		return nil, err
	}

	go func() {
		err := a.deleteExperiments(experiments, curUser)
		if err != nil {
			// set experiment state to DeleteFailed
			for _, id := range req.ExperimentIds {
				log.WithError(err).Errorf("deleting experiment %d", id)
			}
			_, err = db.Bun().NewUpdate().
				ModelTableExpr("experiments as e").
				Set("state = ?", model.DeleteFailedState).
				Where("id IN (?)", bun.In(req.ExperimentIds)).
				Exec(ctx)
			if err != nil {
				for _, id := range req.ExperimentIds {
					log.WithError(err).Errorf("transitioning experiment %d to %s", id,
						model.DeleteFailedState)
				}
			}
			return
		}
		for _, id := range req.ExperimentIds {
			log.Infof("deleted experiment %d", id)
		}
	}()

	return &apiv1.DeleteExperimentsResponse{Results: experiment.ToAPIResults(results)}, err
}

// deleteExperiments synchronously tries to delete all artifacts associated with the provided experiments. An error
// indicates all the experiments were not successfully deleted. Since all artifacts cannot be delete transactionally the
// experiments may be in a partially deleted state, but the experiment at least row will still exist. This can be
// safetly retried as many times as it takes to successfully delete the experiments.
func (a *apiServer) deleteExperiments(exps []*model.Experiment, userModel *model.User) error {
	taskSpec := *a.m.taskSpec

	var expIDs []int
	for _, e := range exps {
		expIDs = append(expIDs, e.ID)
	}
	workspaceIDs, err := workspace.WorkspacesIDsByExperimentIDs(context.TODO(), expIDs)
	if err != nil {
		return err
	}

	sema := make(chan struct{}, maxConcurrentDeletes)
	g, _ := errgroup.WithContext(context.Background())
	for i, exp := range exps {
		g.Go(func() error {
			sema <- struct{}{}
			defer func() { <-sema }()

			agentUserGroup, err := user.GetAgentUserGroup(context.TODO(), *exp.OwnerID, workspaceIDs[i])
			if err != nil {
				log.WithError(err).Errorf("failed to delete experiment: %d", exp.ID)
				return err
			}

			checkpoints, err := experiment.ExperimentCheckpointsToGCRaw(context.TODO(), exp.ID, 0, 0, 0)
			if err != nil {
				log.WithError(err).Errorf("failed to delete experiment: %d", exp.ID)
				return err
			}
			if len(checkpoints) > 0 {
				if err := runCheckpointGCForCheckpoints(
					a.m.rm, a.m.db, exp.JobID, exp.StartTime,
					&taskSpec, exp.ID, exp.Config, checkpoints,
					[]string{fullDeleteGlob}, true, agentUserGroup, userModel, nil,
				); err != nil {
					log.WithError(err).Errorf("failed to gc checkpoints for experiment: %d", exp.ID)
					return err
				}
			}

			// delete jobs per experiment
			resp, err := a.m.rm.DeleteJob(sproto.DeleteJob{
				JobID: exp.JobID,
			})
			if err != nil {
				log.WithError(err).Errorf("requesting cleanup of resource manager resources")
				return err
			}
			if err = <-resp.Err; err != nil {
				log.WithError(err).Errorf("cleaning up resource manager resources")
				return err
			}
			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		return errors.Wrapf(err, "failed to checkpoint gc")
	}

	ctx := context.Background()
	trialIDs, taskIDs, err := db.ExperimentsTrialAndTaskIDs(ctx, db.Bun(), expIDs)
	if err != nil {
		return errors.Wrapf(err, "failed to gather trial IDs for experiment")
	}

	if err = a.m.trialLogBackend.DeleteTrialLogs(trialIDs); err != nil {
		return errors.Wrapf(err, "failed to delete trial logs from backend")
	}

	if err = a.m.taskLogBackend.DeleteTaskLogs(taskIDs); err != nil {
		return errors.Wrapf(err, "failed to delete trial logs from backend (task logs)")
	}

	if err = a.m.db.DeleteExperiments(ctx, expIDs); err != nil {
		return errors.Wrapf(err, "deleting experiments from database")
	}
	return nil
}

func getExperimentColumns(q *bun.SelectQuery) *bun.SelectQuery {
	return q.
		Column("e.id").
		ColumnExpr("e.config->>'description' AS description").
		ColumnExpr("e.config->>'labels' AS labels").
		ColumnExpr("proto_time(e.start_time) AS start_time").
		ColumnExpr("proto_time(e.end_time) AS end_time").
		ColumnExpr("extract(epoch FROM coalesce(e.end_time, now()) - e.start_time)::int AS duration").
		ColumnExpr(bunutils.ProtoStateDBCaseString(experimentv1.State_value, "e.state", "state",
			"STATE_")).
		Column("e.archived").
		ColumnExpr(
			"(SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) AS num_trials").
		// Intentionally not sending trial_ids due to performance.
		ColumnExpr("COALESCE(u.display_name, u.username) as display_name").
		ColumnExpr("e.owner_id as user_id").
		Column("u.username").
		ColumnExpr("e.config->'resources'->>'resource_pool' AS resource_pool").
		ColumnExpr("e.config->'searcher'->>'name' AS searcher_type").
		ColumnExpr("e.config->'searcher'->>'metric' AS searcher_metric").
		ColumnExpr("e.config->'hyperparameters' AS hyperparameters").
		ColumnExpr("e.config->>'name' as NAME").
		ColumnExpr(
			"CASE WHEN NULLIF(e.notes, '') IS NULL THEN NULL ELSE 'omitted' END AS notes").
		Column("e.job_id").
		ColumnExpr("CASE WHEN e.parent_id IS NULL THEN NULL ELSE " +
			"json_build_object('value', e.parent_id) END AS forked_from").
		ColumnExpr("CASE WHEN e.progress IS NULL THEN NULL ELSE " +
			"json_build_object('value', e.progress) END AS progress").
		ColumnExpr("p.name AS project_name").
		Column("e.project_id").
		ColumnExpr("w.id AS workspace_id").
		ColumnExpr("w.name AS workspace_name").
		ColumnExpr("(w.archived OR p.archived) AS parent_archived").
		ColumnExpr("p.user_id AS project_owner_id").
		Column("e.config").
		Column("e.checkpoint_size").
		Column("e.checkpoint_count").
		Column("e.unmanaged").
		Column("e.external_experiment_id").
		ColumnExpr(`r.external_run_id AS external_trial_id`).
		ColumnExpr("NULLIF(e.config#>'{integrations, pachyderm}', 'null') AS pachyderm_integration").
		Join("LEFT JOIN users u ON e.owner_id = u.id").
		Join("LEFT JOIN projects p ON e.project_id = p.id").
		Join("LEFT JOIN workspaces w ON p.workspace_id = w.id").
		Join("LEFT JOIN runs AS r ON r.id = e.best_trial_id")
}

func (a *apiServer) GetExperiments(
	ctx context.Context, req *apiv1.GetExperimentsRequest,
) (*apiv1.GetExperimentsResponse, error) {
	resp := &apiv1.GetExperimentsResponse{Experiments: []*experimentv1.Experiment{}}
	query := db.Bun().NewSelect().
		Model(&resp.Experiments).
		ModelTableExpr("experiments as e").
		Apply(getExperimentColumns)

	if req.ShowTrialData {
		query.ColumnExpr(`
		(
			SELECT searcher_metric_value
			FROM trials t
			WHERE t.id = e.best_trial_id
		) AS best_trial_searcher_metric`)
	}

	// Construct the ordering expression.
	orderColMap := map[apiv1.GetExperimentsRequest_SortBy]string{
		apiv1.GetExperimentsRequest_SORT_BY_UNSPECIFIED:      "id",
		apiv1.GetExperimentsRequest_SORT_BY_ID:               "id",
		apiv1.GetExperimentsRequest_SORT_BY_DESCRIPTION:      "description",
		apiv1.GetExperimentsRequest_SORT_BY_NAME:             "name",
		apiv1.GetExperimentsRequest_SORT_BY_START_TIME:       "e.start_time",
		apiv1.GetExperimentsRequest_SORT_BY_END_TIME:         "e.end_time",
		apiv1.GetExperimentsRequest_SORT_BY_STATE:            "e.state",
		apiv1.GetExperimentsRequest_SORT_BY_NUM_TRIALS:       "num_trials",
		apiv1.GetExperimentsRequest_SORT_BY_PROGRESS:         "COALESCE(progress, 0)",
		apiv1.GetExperimentsRequest_SORT_BY_USER:             "display_name",
		apiv1.GetExperimentsRequest_SORT_BY_FORKED_FROM:      "e.parent_id",
		apiv1.GetExperimentsRequest_SORT_BY_RESOURCE_POOL:    "resource_pool",
		apiv1.GetExperimentsRequest_SORT_BY_PROJECT_ID:       "e.project_id",
		apiv1.GetExperimentsRequest_SORT_BY_CHECKPOINT_SIZE:  "checkpoint_size",
		apiv1.GetExperimentsRequest_SORT_BY_CHECKPOINT_COUNT: "checkpoint_count",
		apiv1.GetExperimentsRequest_SORT_BY_SEARCHER_METRIC_VAL: `(
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

	if req.ExperimentIdFilter != nil {
		var err error
		query, err = db.ApplyInt32FieldFilter(query, bun.Ident("e.id"), req.ExperimentIdFilter)
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

	resp.Pagination, err = runPagedBunExperimentsQuery(ctx, query, int(req.Offset), int(req.Limit))
	if err != nil {
		return nil, err
	}

	if err = a.enrichExperimentState(resp.Experiments...); err != nil {
		return nil, err
	}

	return resp, nil
}

func runPagedBunExperimentsQuery(
	ctx context.Context, query *bun.SelectQuery, offset, limit int,
) (*apiv1.Pagination, error) {
	// Count number of items without any limits or offsets.
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate end and start indexes.
	startIndex := offset
	if offset > total || offset < -total {
		startIndex = total
	} else if offset < 0 {
		startIndex = total + offset
	}

	endIndex := startIndex + limit
	switch {
	case limit == -2:
		endIndex = startIndex
	case limit == -1:
		endIndex = total
	case limit == 0:
		endIndex = 100 + startIndex
		if total < endIndex {
			endIndex = total
		}
	case startIndex+limit > total:
		endIndex = total
	}

	// Add start and end index to query.
	query.Offset(startIndex)
	query.Limit(endIndex - startIndex)

	// Bun bug treating limit=0 as no limit when it
	// should be the exact opposite of no records returned.
	if endIndex-startIndex != 0 {
		if err = query.Scan(ctx); err != nil {
			return nil, err
		}
	}

	return &apiv1.Pagination{
		Offset:     int32(offset),
		Limit:      int32(limit),
		Total:      int32(total),
		StartIndex: int32(startIndex),
		EndIndex:   int32(endIndex),
	}, nil
}

func (a *apiServer) GetExperimentLabels(ctx context.Context,
	req *apiv1.GetExperimentLabelsRequest,
) (*apiv1.GetExperimentLabelsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	resp := &apiv1.GetExperimentLabelsResponse{}
	var labels [][]string
	query := db.Bun().NewSelect().
		Table("experiments").
		Model(&labels).
		ColumnExpr("config->'labels' AS labels").
		Distinct()

	var proj *projectv1.Project
	if req.ProjectId != 0 {
		proj, err = a.GetProjectByID(ctx, req.ProjectId, *curUser)
		if err != nil {
			return nil, err
		}

		query = query.Where("project_id = ?", req.ProjectId)
	}

	if query, err = experiment.AuthZProvider.Get().
		FilterExperimentLabelsQuery(ctx, *curUser, proj, query); err != nil {
		return nil, err
	}

	if err = query.Scan(ctx); err != nil {
		return nil, err
	}

	// Sort labels by usage.
	labelUsage := make(map[string]int)
	for _, labelArr := range labels {
		for _, l := range labelArr {
			labelUsage[l]++
		}
	}

	resp.Labels = make([]string, len(labelUsage))
	i := 0
	for label := range labelUsage {
		resp.Labels[i] = label
		i++
	}
	sort.Slice(resp.Labels, func(i, j int) bool {
		return labelUsage[resp.Labels[i]] > labelUsage[resp.Labels[j]]
	})
	return resp, nil
}

func (a *apiServer) GetExperimentValidationHistory(
	ctx context.Context, req *apiv1.GetExperimentValidationHistoryRequest,
) (*apiv1.GetExperimentValidationHistoryResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	var resp apiv1.GetExperimentValidationHistoryResponse
	switch err := a.m.db.QueryProto("proto_experiment_validation_history", &resp, req.ExperimentId); {
	case err == db.ErrNotFound:
		return nil, api.NotFoundErrs("experiment", strconv.Itoa(int(req.ExperimentId)), true)
	case err != nil:
		return nil, errors.Wrapf(err,
			"error fetching validation history for experiment from database: %d", req.ExperimentId)
	}
	return &resp, nil
}

func (a *apiServer) PreviewHPSearch(
	ctx context.Context, req *apiv1.PreviewHPSearchRequest,
) (*apiv1.PreviewHPSearchResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = experiment.AuthZProvider.Get().CanPreviewHPSearch(ctx, *curUser); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	bytes, err := protojson.Marshal(req.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error parsing experiment config: %s", err)
	}

	// Parse the provided experiment config.
	config, err := expconf.ParseAnyExperimentConfigYAML(bytes)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "invalid experiment configuration: %s", err,
		)
	}

	// Get the useful subconfigs for preview search.
	if config.RawSearcher == nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "invalid experiment configuration; missing searcher",
		)
	}
	sc := *config.RawSearcher
	hc := config.RawHyperparameters

	// Apply any json-schema-defined defaults.
	sc = schemas.WithDefaults(sc)
	hc = schemas.WithDefaults(hc)

	// Make sure the searcher config has all eventuallyRequired fields.
	if err = schemas.IsComplete(sc); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid searcher configuration: %s", err)
	}
	if err = schemas.IsComplete(hc); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "invalid hyperparameters configuration: %s", err,
		)
	}

	// Disallow EOL searchers.
	if err = sc.AssertCurrent(); err != nil {
		return nil, errors.Wrap(err, "invalid experiment configuration")
	}

	sm := searcher.NewSearchMethod(sc)
	s := searcher.NewSearcher(req.Seed, sm, hc)
	sim, err := searcher.Simulate(s, nil, searcher.RandomValidation, true, sc.Metric())
	if err != nil {
		return nil, err
	}
	protoSim := &experimentv1.ExperimentSimulation{Seed: req.Seed}
	indexes := make(map[string]int, len(sim.Results))
	toProto := func(op searcher.ValidateAfter) ([]*experimentv1.RunnableOperation, error) {
		return []*experimentv1.RunnableOperation{
			{
				Type:   experimentv1.RunnableType_RUNNABLE_TYPE_TRAIN,
				Length: op.Length,
			},
			{
				Type: experimentv1.RunnableType_RUNNABLE_TYPE_VALIDATE,
			},
		}, nil
	}
	for _, result := range sim.Results {
		var operations []*experimentv1.RunnableOperation
		for _, msg := range result {
			ops, err := toProto(msg)
			if err != nil {
				return nil, errors.Wrapf(err, "error converting msg in simultion result %s", msg)
			}
			operations = append(operations, ops...)
		}
		hash := fmt.Sprint(operations)
		if i, ok := indexes[hash]; ok {
			protoSim.Trials[i].Occurrences++
		} else {
			protoSim.Trials = append(protoSim.Trials,
				&experimentv1.TrialSimulation{Operations: operations, Occurrences: 1})
			indexes[hash] = len(protoSim.Trials) - 1
		}
	}
	return &apiv1.PreviewHPSearchResponse{Simulation: protoSim}, nil
}

func (a *apiServer) ActivateExperiment(
	ctx context.Context, req *apiv1.ActivateExperimentRequest,
) (resp *apiv1.ActivateExperimentResponse, err error) {
	if _, _, err = a.getExperimentAndCheckCanDoActions(ctx, int(req.Id),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	e, ok := experiment.ExperimentRegistry.Load(int(req.Id))
	if !ok {
		return nil, api.NotFoundErrs("experiment", strconv.Itoa(int(req.Id)), true)
	}
	if err := e.ActivateExperiment(); err != nil {
		return nil, err
	}
	return &apiv1.ActivateExperimentResponse{}, nil
}

func (a *apiServer) ActivateExperiments(
	ctx context.Context, req *apiv1.ActivateExperimentsRequest,
) (*apiv1.ActivateExperimentsResponse, error) {
	results, err := experiment.ActivateExperiments(ctx, req.ProjectId, req.ExperimentIds, req.Filters)
	return &apiv1.ActivateExperimentsResponse{Results: experiment.ToAPIResults(results)}, err
}

func (a *apiServer) PauseExperiment(
	ctx context.Context, req *apiv1.PauseExperimentRequest,
) (resp *apiv1.PauseExperimentResponse, err error) {
	results, err := experiment.PauseExperiments(ctx, experiment.GlobalProjectID, []int32{req.Id}, nil)

	if err == nil {
		if len(results) == 0 {
			return nil, errors.Errorf("PauseExperiments returned neither pass nor fail on query")
		} else if results[0].Error != nil {
			return nil, results[0].Error
		}
	}

	return &apiv1.PauseExperimentResponse{}, err
}

func (a *apiServer) PauseExperiments(
	ctx context.Context, req *apiv1.PauseExperimentsRequest,
) (*apiv1.PauseExperimentsResponse, error) {
	results, err := experiment.PauseExperiments(ctx, req.ProjectId, req.ExperimentIds, req.Filters)
	return &apiv1.PauseExperimentsResponse{Results: experiment.ToAPIResults(results)}, err
}

func (a *apiServer) CancelExperiment(
	ctx context.Context, req *apiv1.CancelExperimentRequest,
) (resp *apiv1.CancelExperimentResponse, err error) {
	results, err := experiment.CancelExperiments(ctx, experiment.GlobalProjectID, []int32{req.Id}, nil)

	if err == nil {
		if len(results) == 0 {
			return nil, errors.Errorf("CancelExperiments returned neither pass nor fail on query")
		} else if results[0].Error != nil {
			return nil, results[0].Error
		}
	}

	return &apiv1.CancelExperimentResponse{}, err
}

func (a *apiServer) CancelExperiments(
	ctx context.Context, req *apiv1.CancelExperimentsRequest,
) (*apiv1.CancelExperimentsResponse, error) {
	results, err := experiment.CancelExperiments(ctx, req.ProjectId, req.ExperimentIds, req.Filters)
	return &apiv1.CancelExperimentsResponse{Results: experiment.ToAPIResults(results)}, err
}

func (a *apiServer) KillExperiment(
	ctx context.Context, req *apiv1.KillExperimentRequest,
) (resp *apiv1.KillExperimentResponse, err error) {
	results, err := experiment.KillExperiments(ctx, experiment.GlobalProjectID, []int32{req.Id}, nil)

	if err == nil {
		if len(results) == 0 {
			return nil, errors.Errorf("KillExperiments returned neither pass nor fail on query")
		} else if results[0].Error != nil {
			return nil, results[0].Error
		}
	}

	return &apiv1.KillExperimentResponse{}, err
}

func (a *apiServer) KillExperiments(
	ctx context.Context, req *apiv1.KillExperimentsRequest,
) (*apiv1.KillExperimentsResponse, error) {
	results, err := experiment.KillExperiments(ctx, req.ProjectId, req.ExperimentIds, req.Filters)
	return &apiv1.KillExperimentsResponse{Results: experiment.ToAPIResults(results)}, err
}

func (a *apiServer) ArchiveExperiment(
	ctx context.Context, req *apiv1.ArchiveExperimentRequest,
) (*apiv1.ArchiveExperimentResponse, error) {
	results, err := experiment.ArchiveExperiments(ctx, experiment.GlobalProjectID, []int32{req.Id}, nil)

	if err == nil {
		if len(results) == 0 {
			return nil, errors.Errorf("ArchiveExperiments returned neither pass nor fail on query")
		} else if results[0].Error != nil {
			return nil, results[0].Error
		}
	}

	return &apiv1.ArchiveExperimentResponse{}, err
}

func (a *apiServer) ArchiveExperiments(
	ctx context.Context, req *apiv1.ArchiveExperimentsRequest,
) (*apiv1.ArchiveExperimentsResponse, error) {
	results, err := experiment.ArchiveExperiments(ctx, req.ProjectId, req.ExperimentIds, req.Filters)
	return &apiv1.ArchiveExperimentsResponse{Results: experiment.ToAPIResults(results)}, err
}

func (a *apiServer) UnarchiveExperiment(
	ctx context.Context, req *apiv1.UnarchiveExperimentRequest,
) (*apiv1.UnarchiveExperimentResponse, error) {
	results, err := experiment.UnarchiveExperiments(ctx, experiment.GlobalProjectID, []int32{req.Id}, nil)

	if err == nil {
		if len(results) == 0 {
			return nil, errors.Errorf("UnarchiveExperiments returned neither pass nor fail on query")
		} else if results[0].Error != nil {
			return nil, results[0].Error
		}
	}

	return &apiv1.UnarchiveExperimentResponse{}, err
}

func (a *apiServer) UnarchiveExperiments(
	ctx context.Context, req *apiv1.UnarchiveExperimentsRequest,
) (*apiv1.UnarchiveExperimentsResponse, error) {
	results, err := experiment.UnarchiveExperiments(ctx, req.ProjectId, req.ExperimentIds, req.Filters)
	return &apiv1.UnarchiveExperimentsResponse{Results: experiment.ToAPIResults(results)}, err
}

func (a *apiServer) PatchExperiment(
	ctx context.Context, req *apiv1.PatchExperimentRequest,
) (*apiv1.PatchExperimentResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	exp, err := a.getExperiment(ctx, *curUser, int(req.Experiment.Id))
	if err != nil {
		return nil, err
	}
	modelExp, err := model.ExperimentFromProto(exp)
	if err != nil {
		return nil, err
	}

	if err = experiment.AuthZProvider.Get().CanEditExperimentsMetadata(
		ctx, *curUser, modelExp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	madeChanges := false
	if req.Experiment.Name != nil && exp.Name != req.Experiment.Name.Value {
		madeChanges = true
		if len(strings.TrimSpace(req.Experiment.Name.Value)) == 0 {
			return nil, status.Errorf(codes.InvalidArgument,
				"`name` must not be an empty or whitespace string")
		}
		exp.Name = req.Experiment.Name.Value
	}

	if req.Experiment.Notes != nil && exp.Notes != req.Experiment.Notes.Value {
		madeChanges = true
		exp.Notes = req.Experiment.Notes.Value
	}

	if req.Experiment.Description != nil && exp.Description != req.Experiment.Description.Value {
		madeChanges = true
		exp.Description = req.Experiment.Description.Value
	}

	if req.Experiment.Labels != nil {
		// avoid duplicate keys
		reqLabelSet := make(map[string]struct{}, len(req.Experiment.Labels.Values))
		for _, el := range req.Experiment.Labels.Values {
			if _, ok := el.GetKind().(*structpb.Value_StringValue); ok {
				reqLabelSet[el.GetStringValue()] = struct{}{}
			}
		}
		reqLabelList := make([]string, len(reqLabelSet))
		i := 0
		for key := range reqLabelSet {
			reqLabelList[i] = key
			i++
		}
		reqLabels := strings.Join(reqLabelList, ",")
		if strings.Join(exp.Labels, ",") != reqLabels {
			madeChanges = true
			exp.Labels = reqLabelList
			prom.AssociateExperimentIDLabels(strconv.Itoa(int(req.Experiment.Id)),
				exp.Labels)
		}
	}

	if madeChanges {
		type experimentPatch struct {
			Labels      []string `json:"labels"`
			Description string   `json:"description"`
			Name        string   `json:"name"`
		}
		patches := experimentPatch{
			Labels:      exp.Labels,
			Description: exp.Description,
			Name:        exp.Name,
		}
		marshalledPatches, patchErr := json.Marshal(patches)
		if patchErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal experiment patch")
		}

		_, err = a.m.db.RawQuery(
			"patch_experiment", exp.Id, marshalledPatches, exp.Notes)
		if err != nil {
			return nil, errors.Wrapf(err, "error updating experiment in database: %d", req.Experiment.Id)
		}
	}

	if req.Experiment.Resources != nil || req.Experiment.CheckpointStorage != nil {
		// TODO(DET-8577): Remove unnecessary active config usage.
		activeConfig, err := a.m.db.ActiveExperimentConfig(int(exp.Id))
		if err != nil {
			return nil, errors.Wrapf(
				err, "unable to load no-longer-valid config for experiment %v", exp.Id,
			)
		}

		newResources := req.Experiment.Resources
		if newResources != nil {
			resources := activeConfig.Resources()
			if newResources.MaxSlots != nil {
				if err = experiment.AuthZProvider.Get().
					CanSetExperimentsMaxSlots(ctx, *curUser, modelExp, int(*newResources.MaxSlots)); err != nil {
					return nil, echo.NewHTTPError(http.StatusForbidden, err.Error())
				}

				resources.SetMaxSlots(ptrs.Ptr(int(*newResources.MaxSlots)))
			}
			if newResources.Weight != nil {
				if err = experiment.AuthZProvider.Get().
					CanSetExperimentsWeight(ctx, *curUser, modelExp, *newResources.Weight); err != nil {
					return nil, echo.NewHTTPError(http.StatusForbidden, err.Error())
				}

				resources.SetWeight(*newResources.Weight)
			}
			if newResources.Priority != nil {
				if err = experiment.AuthZProvider.Get().
					CanSetExperimentsPriority(ctx, *curUser, modelExp, int(*newResources.Priority)); err != nil {
					return nil, echo.NewHTTPError(http.StatusForbidden, err.Error())
				}

				resources.SetPriority(ptrs.Ptr(int(*newResources.Priority)))
			}
			activeConfig.SetResources(resources)
		}
		newCheckpointStorage := req.Experiment.CheckpointStorage

		if newCheckpointStorage != nil {
			if err = experiment.AuthZProvider.Get().
				CanSetExperimentsCheckpointGCPolicy(ctx, *curUser, modelExp); err != nil {
				return nil, echo.NewHTTPError(http.StatusForbidden, err.Error())
			}

			storage := activeConfig.CheckpointStorage()
			storage.SetSaveExperimentBest(int(newCheckpointStorage.SaveExperimentBest))
			storage.SetSaveTrialBest(int(newCheckpointStorage.SaveTrialBest))
			storage.SetSaveTrialLatest(int(newCheckpointStorage.SaveTrialLatest))
			activeConfig.SetCheckpointStorage(storage)
		}

		// `patch` represents the allowed mutations that can be performed on an experiment, in JSON
		if err := a.m.db.SaveExperimentConfig(modelExp.ID, activeConfig); err != nil {
			return nil, errors.Wrapf(err, "patching experiment %d", modelExp.ID)
		}

		if newResources != nil {
			e, ok := experiment.ExperimentRegistry.Load(int(exp.Id))
			if !ok {
				return nil, api.NotFoundErrs("experiment", strconv.Itoa(int(exp.Id)), true)
			}

			if newResources.MaxSlots != nil {
				msg := sproto.SetGroupMaxSlots{MaxSlots: ptrs.Ptr(int(*newResources.MaxSlots))}
				e.SetGroupMaxSlots(msg)
			}
			if newResources.Weight != nil {
				err := e.SetGroupWeight(*newResources.Weight)
				if err != nil {
					return nil, errors.Wrapf(err, "cannot change experiment weight to %v", *newResources.Weight)
				}
			}
			if newResources.Priority != nil {
				err := e.SetGroupPriority(int(*newResources.Priority))
				if err != nil {
					return nil, errors.Wrapf(err, "cannot change experiment priority to %v", *newResources.Priority)
				}
			}
		}

		if newCheckpointStorage != nil {
			checkpoints, err := experiment.ExperimentCheckpointsToGCRaw(
				ctx,
				modelExp.ID,
				modelExp.Config.CheckpointStorage.SaveExperimentBest(),
				modelExp.Config.CheckpointStorage.SaveTrialBest(),
				modelExp.Config.CheckpointStorage.SaveTrialLatest(),
			)
			if err != nil {
				return nil, err
			}

			workspaceID, err := workspace.WorkspacesIDsByExperimentIDs(ctx, []int{modelExp.ID})
			if err != nil {
				return nil, err
			}
			agentUserGroup, err := user.GetAgentUserGroup(context.TODO(), *modelExp.OwnerID, workspaceID[0])
			if err != nil {
				return nil, err
			}

			ownerFullUser, err := user.ByID(ctx, *modelExp.OwnerID)
			if err != nil {
				return nil, errors.Errorf("cannot find user %v who owns experiment", modelExp.OwnerID)
			}

			taskSpec := *a.m.taskSpec
			user := &model.User{
				ID:       ownerFullUser.ID,
				Username: ownerFullUser.Username,
			}

			go func() {
				if err := runCheckpointGCForCheckpoints(
					a.m.rm, a.m.db, modelExp.JobID, modelExp.StartTime,
					&taskSpec, modelExp.ID, modelExp.Config, checkpoints,
					[]string{fullDeleteGlob}, true, agentUserGroup, user, nil,
				); err != nil {
					log.WithError(err).Error("failed to GC checkpoints in patch experiment")
				}
			}()
		}
	}

	// include queued / pulling / starting / running state
	if err = a.enrichExperimentState(exp); err != nil {
		return nil, err
	}

	return &apiv1.PatchExperimentResponse{Experiment: exp}, nil
}

func (a *apiServer) GetExperimentCheckpoints(
	ctx context.Context, req *apiv1.GetExperimentCheckpointsRequest,
) (*apiv1.GetExperimentCheckpointsResponse, error) {
	experimentID := int(req.Id)
	useSearcherSortBy := req.GetSortByAttr() == checkpointv1.SortBy_SORT_BY_SEARCHER_METRIC
	exp, _, err := a.getExperimentAndCheckCanDoActions(ctx, experimentID,
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts)
	if err != nil {
		return nil, err
	}

	// If SORT_BY_SEARCHER_METRIC is specified without an OrderBy
	// default to ordering by "better" checkpoints.
	if useSearcherSortBy && req.OrderBy == apiv1.OrderBy_ORDER_BY_UNSPECIFIED {
		if exp.Config.Searcher.SmallerIsBetter {
			req.OrderBy = apiv1.OrderBy_ORDER_BY_ASC
		} else {
			req.OrderBy = apiv1.OrderBy_ORDER_BY_DESC
		}
	}

	resp := &apiv1.GetExperimentCheckpointsResponse{}
	resp.Checkpoints = []*checkpointv1.Checkpoint{}
	switch err = a.m.db.QueryProto("get_checkpoints_for_experiment", &resp.Checkpoints, req.Id); {
	case err == db.ErrNotFound:
		return nil, api.NotFoundErrs("checkpoints for experiment", strconv.Itoa(int(req.Id)), true)
	case err != nil:
		return nil,
			errors.Wrapf(err, "error fetching checkpoints for experiment %d from database", req.Id)
	}

	api.Where(&resp.Checkpoints, func(i int) bool {
		v := resp.Checkpoints[i]

		found := false
		for _, state := range req.States {
			if state == v.State {
				found = true
				break
			}
		}

		if len(req.States) != 0 && !found {
			return false
		}

		return true
	})

	sort.Slice(resp.Checkpoints, func(i, j int) bool {
		ai, aj := resp.Checkpoints[i], resp.Checkpoints[j]
		if useSearcherSortBy {
			if order, done := protoless.CheckpointSearcherMetricNullsLast(ai, aj); done {
				return order
			}
		}

		if req.OrderBy == apiv1.OrderBy_ORDER_BY_DESC {
			aj, ai = ai, aj
		}
		switch req.GetSortBy().(type) {
		case *apiv1.GetExperimentCheckpointsRequest_SortByAttr:
			switch req.GetSortByAttr() {
			case checkpointv1.SortBy_SORT_BY_BATCH_NUMBER:
				return protoless.CheckpointStepsCompletedLess(ai, aj)
			case checkpointv1.SortBy_SORT_BY_UUID:
				return ai.Uuid < aj.Uuid
			case checkpointv1.SortBy_SORT_BY_TRIAL_ID:
				return protoless.CheckpointTrialIDLess(ai, aj)
			case checkpointv1.SortBy_SORT_BY_END_TIME:
				return protoless.CheckpointReportTimeLess(ai, aj)
			case checkpointv1.SortBy_SORT_BY_STATE:
				return ai.State.Number() < aj.State.Number()
			case checkpointv1.SortBy_SORT_BY_SEARCHER_METRIC:
				return protoless.CheckpointSearcherMetricLess(ai, aj)
			case checkpointv1.SortBy_SORT_BY_UNSPECIFIED:
				fallthrough
			default:
				return protoless.CheckpointTrialIDLess(ai, aj)
			}
		case *apiv1.GetExperimentCheckpointsRequest_SortByMetric:
			return protoless.CheckpointMetricNameLess(ai, aj, req.GetSortByMetric())
		default:
			return protoless.CheckpointTrialIDLess(ai, aj)
		}
	})
	return resp, api.Paginate(&resp.Pagination, &resp.Checkpoints, req.Offset, req.Limit)
}

func (a *apiServer) createUnmanagedExperimentTx(
	ctx context.Context, idb bun.IDB, dbExp *model.Experiment, modelDef []byte,
	activeConfig expconf.ExperimentConfigV0, user *model.User,
) (*apiv1.CreateExperimentResponse, error) {
	e, _, err := newUnmanagedExperiment(ctx, idb, dbExp, modelDef, activeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to make new unmanaged experiment: %w", err)
	}

	protoExp, err := a.getExperimentTx(ctx, idb, *user, e.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}
	return &apiv1.CreateExperimentResponse{
		Experiment: protoExp,
		Config:     protoutils.ToStruct(activeConfig),
	}, nil
}

func (a *apiServer) parseAndMergeContinueConfig(expID int, overrideConfig string) ([]byte, bool, error) {
	if overrideConfig == "" {
		overrideConfig = "{}" //nolint: goconst
	}

	activeConfig, err := a.m.db.ActiveExperimentConfig(expID)
	if err != nil {
		return nil, false, fmt.Errorf("loading active config for experiment %d: %w", expID, err)
	}
	name := activeConfig.Searcher().AsLegacy().Name
	isSingle := name == "single"                           //nolint: goconst
	if !isSingle && (name != "grid" && name != "random") { //nolint: goconst
		return nil, false, status.Errorf(codes.InvalidArgument,
			fmt.Sprintf("Unsupported searcher type provided: '%s'", name))
	}
	if !isSingle && strings.TrimSpace(overrideConfig) != "{}" { //nolint: goconst
		return nil, false, status.Errorf(codes.InvalidArgument,
			fmt.Sprintf("override config is provided and experiment is not single searcher, got '%s' instead", name))
	}

	providedConfig, err := expconf.ParseAnyExperimentConfigYAML([]byte(overrideConfig))
	if err != nil {
		// Add a helpful error message if a user just submits
		// searcher.max_length.batches = 2. They would also need
		// searcher.name = "single", which all experiments will always be here.
		if strings.Contains(err.Error(), `unknown field "max_length"`) {
			return nil, false, status.Errorf(codes.InvalidArgument,
				`unknown field "max_length", you might also need to specify searcher.name=single`)
		}

		return nil, false, status.Errorf(codes.InvalidArgument,
			fmt.Errorf("parsing override config: %w", err).Error())
	}

	if providedConfig.RawProject != nil {
		return nil, false, status.Errorf(codes.InvalidArgument, "'project' in override config "+
			"cannot be specified, use `det experiment move` first if you want to change the project")
	}
	if providedConfig.RawWorkspace != nil {
		return nil, false, status.Errorf(codes.InvalidArgument, "'workspace' in override config "+
			"cannot be specified, use `det experiment move` first if you want to change the workspace")
	}
	mergedConfig := schemas.Merge(providedConfig, activeConfig)
	if overrideName := mergedConfig.Searcher().AsLegacy().Name; isSingle && overrideName != "single" {
		return nil, false, status.Errorf(codes.InvalidArgument,
			fmt.Sprintf("override config must have single searcher type got '%s' instead", overrideName))
	}

	bytes, err := mergedConfig.Value()
	if err != nil {
		return nil, false, fmt.Errorf("getting value of merged config: %w", err)
	}

	return bytes.([]byte), isSingle, nil
}

var errContinueHPSearchCompleted = status.Error(codes.FailedPrecondition,
	"experiment has been completed, cannot continue this experiment")

func (a *apiServer) ContinueExperiment(
	ctx context.Context, req *apiv1.ContinueExperimentRequest,
) (*apiv1.ContinueExperimentResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	origExperiment, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.Id),
		experiment.AuthZProvider.Get().CanEditExperiment)
	if err != nil {
		return nil, err
	}

	trialsResp, err := a.GetExperimentTrials(ctx, &apiv1.GetExperimentTrialsRequest{
		ExperimentId: req.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("getting experiment trials: %w", err)
	}
	configBytes, isSingle, err := a.parseAndMergeContinueConfig(int(req.Id), req.OverrideConfig)
	if err != nil {
		return nil, err
	}

	dbExp, modelDef, activeConfig, _, taskSpec, err := a.m.parseCreateExperiment(ctx,
		&apiv1.CreateExperimentRequest{
			Config: string(configBytes),
		}, user,
	)
	if err != nil {
		return nil, fmt.Errorf("parsing continue experiment request: %w", err)
	}
	dbExp.ID = int(req.Id)
	dbExp.JobID = origExperiment.JobID // Revive job.

	e, launchWarnings, err := newExperiment(a.m, dbExp, modelDef, activeConfig, taskSpec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create experiment: %s", err)
	}

	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Lock experiment state.
		var expState model.State
		if err = tx.
			NewRaw(`SELECT state FROM experiments WHERE id = ? FOR UPDATE`, req.Id).
			Scan(ctx, &expState); err != nil {
			return fmt.Errorf("getting / locking experiment state: %w", err)
		}
		if !model.TerminalStates[expState] {
			return status.Error(codes.FailedPrecondition, fmt.Sprintf(
				"experiment in non terminal state '%s', try again later", expState))
		}

		if expState == model.CompletedState && !isSingle {
			hasIncompleteTrials := false
			for _, trial := range trialsResp.Trials {
				if trial.State != trialv1.State_STATE_COMPLETED {
					hasIncompleteTrials = true
					break
				}
			}
			if !hasIncompleteTrials {
				return errContinueHPSearchCompleted
			}
		} else if isSingle && len(trialsResp.Trials) > 0 {
			if _, err := tx.NewUpdate().Table("runs"). // TODO(nick-runs) call runs package.
									Set("state = ?", model.PausedState).
									Where("id = ?", trialsResp.Trials[0].Id).
									Exec(ctx); err != nil {
				return fmt.Errorf("changing trial state to PAUSED: %w", err)
			}
		}

		if _, err := tx.NewUpdate().Model(&model.Experiment{}).
			Set("state = ?", model.PausedState). // Throw it in paused.
			Set("progress = ?", 0.0).            // Reset progress.
			Set("end_time = null").
			Where("id = ?", req.Id).
			Exec(ctx); err != nil {
			return fmt.Errorf("updating experiments config: %w", err)
		}

		if _, err := db.Bun().NewUpdate().Model(&model.Job{}).
			Set("q_position = DEFAULT").
			Where("job_id = ?", dbExp.JobID).
			Exec(ctx); err != nil {
			return fmt.Errorf("updating experiment's job: %w", err)
		}

		// Update active config but not original config.
		// We actually do this in experiment's PreStart in setWeight but relying on that
		// is a fun regression waiting to happen.
		activeConfigStr, err := json.Marshal(activeConfig)
		if err != nil {
			return fmt.Errorf("unmarshaling exp config %v: %w", activeConfig, err)
		}
		if _, err := tx.NewUpdate().Model(&model.Experiment{}).
			Set("config = ?", string(activeConfigStr)).
			Where("id = ?", req.Id).
			Exec(ctx); err != nil {
			return fmt.Errorf("updating experiments config: %w", err)
		}

		// Zero out trial restarts. We do somewhat lose information about how many times
		// the previous failed but likely people care only about current run.
		var trialIDs []int32
		for _, t := range trialsResp.Trials {
			trialIDs = append(trialIDs, t.Id)
		}
		if len(trialIDs) > 0 {
			if _, err := tx.NewUpdate().Table("runs"). // TODO(nick-runs) call runs package.
									Set("restarts = 0").
									Set("end_time = null").
									Where("id IN (?)", bun.In(trialIDs)).
									Exec(ctx); err != nil {
				return fmt.Errorf("zeroing out trial restarts: %w", err)
			}
		}

		e.continueTrials = true

		// Check at the end to minimize chance of experiment already being created somehow.
		if _, ok := experiment.ExperimentRegistry.Load(int(req.Id)); ok {
			return fmt.Errorf("experiment already exists")
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("experiment continue database updates: %w", err)
	}

	if err = e.Start(); err != nil {
		return nil, errors.Wrapf(err, "failed to start experiment %d", e.ID)
	}

	_, err = a.ActivateExperiment(ctx, &apiv1.ActivateExperimentRequest{Id: int32(e.ID)})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to activate experiment: %s", err)
	}

	protoExp, err := a.getExperiment(ctx, *user, int(req.Id))
	if err != nil {
		return nil, err
	}
	return &apiv1.ContinueExperimentResponse{
		Experiment: protoExp,
		Warnings:   command.LaunchWarningToProto(launchWarnings),
	}, nil
}

func (a *apiServer) CreateExperiment(
	ctx context.Context, req *apiv1.CreateExperimentRequest,
) (*apiv1.CreateExperimentResponse, error) {
	user, session, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	if req.ParentId != 0 {
		// Can't use getExperimentAndCheckDoActions since model.Experiment doesn't have ParentArchived.
		var parentExp *experimentv1.Experiment
		parentExp, err = a.getExperiment(ctx, *user, int(req.ParentId))
		if err != nil {
			return nil, err
		}
		var modelExp *model.Experiment
		modelExp, err = model.ExperimentFromProto(parentExp)
		if err != nil {
			return nil, err
		}

		if err = experiment.AuthZProvider.Get().
			CanForkFromExperiment(ctx, *user, modelExp); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
		if parentExp.ParentArchived {
			return nil, status.Errorf(codes.Internal,
				"forking an experiment in an archived workspace/project")
		}
	}

	dbExp, modelDef, activeConfig, p, taskSpec, err := a.m.parseCreateExperiment(ctx,
		req, user,
	)
	if err != nil {
		return nil, err
	}

	pachyEnvVars, err := a.getOIDCPachydermEnvVars(session)
	if err != nil {
		return nil, err
	}

	if taskSpec.ExtraEnvVars == nil {
		taskSpec.ExtraEnvVars = map[string]string{}
	}
	maps.Copy(taskSpec.ExtraEnvVars, pachyEnvVars)

	if err = experiment.AuthZProvider.Get().CanCreateExperiment(ctx, *user, p); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	if req.ValidateOnly {
		return &apiv1.CreateExperimentResponse{
			Experiment: &experimentv1.Experiment{},
		}, nil
	}

	if req.Unmanaged != nil && *req.Unmanaged {
		return a.createUnmanagedExperimentTx(ctx, db.Bun(), dbExp, modelDef, activeConfig, user)
	}
	// Check user has permission for what they are trying to do
	// before actually saving the experiment.
	if req.Activate {
		if err = experiment.AuthZProvider.Get().CanEditExperiment(ctx, *user, dbExp); err != nil {
			return nil, status.Errorf(codes.PermissionDenied, err.Error())
		}
	}

	e, launchWarnings, err := newExperiment(a.m, dbExp, modelDef, activeConfig, taskSpec)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create experiment: %s", err)
	}
	modelDef = nil //nolint:ineffassign

	if err = e.Start(); err != nil {
		return nil, errors.Wrapf(err, "failed to start experiment %d", e.ID)
	}

	if req.Activate {
		_, err = a.ActivateExperiment(ctx, &apiv1.ActivateExperimentRequest{Id: int32(e.ID)})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to activate experiment: %s", err)
		}
	}

	protoExp, err := a.getExperiment(ctx, *user, e.ID)
	if err != nil {
		return nil, err
	}
	return &apiv1.CreateExperimentResponse{
		Experiment: protoExp,
		Config:     protoutils.ToStruct(activeConfig),
		Warnings:   command.LaunchWarningToProto(launchWarnings),
	}, nil
}

func (a *apiServer) PutExperimentRetainLogs(
	ctx context.Context, req *apiv1.PutExperimentRetainLogsRequest,
) (*apiv1.PutExperimentRetainLogsResponse, error) {
	results, err := experiment.BulkUpdateLogRetention(
		ctx,
		a.m.db,
		experiment.GlobalProjectID,
		[]int32{req.ExperimentId},
		nil,
		int16(req.NumDays),
	)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.Errorf("PutExperimentRetainLogs returned neither pass nor fail on query")
	} else if results[0].Error != nil {
		return nil, results[0].Error
	}
	return &apiv1.PutExperimentRetainLogsResponse{}, nil
}

func (a *apiServer) PutExperimentsRetainLogs(
	ctx context.Context, req *apiv1.PutExperimentsRetainLogsRequest,
) (*apiv1.PutExperimentsRetainLogsResponse, error) {
	results, err := experiment.BulkUpdateLogRetention(
		ctx,
		a.m.db,
		req.ProjectId,
		req.ExperimentIds,
		req.Filters,
		int16(req.NumDays),
	)
	if err != nil {
		return nil, err
	}
	return &apiv1.PutExperimentsRetainLogsResponse{Results: experiment.ToAPIResults(results)}, nil
}

func (a *apiServer) PutExperiment(
	ctx context.Context, req *apiv1.PutExperimentRequest,
) (*apiv1.PutExperimentResponse, error) {
	if req.CreateExperimentRequest.Unmanaged == nil || !*req.CreateExperimentRequest.Unmanaged {
		return nil, fmt.Errorf("only unmanaged experiments are supported")
	}

	if req.CreateExperimentRequest.ParentId != 0 {
		return nil, fmt.Errorf("can't fork into an unmanaged experiment")
	}

	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	dbExp, modelDef, activeConfig, p, _, err := a.m.parseCreateExperiment(ctx,
		req.CreateExperimentRequest, user,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse exp config: %w", err)
	}
	if err = experiment.AuthZProvider.Get().CanCreateExperiment(ctx, *user, p); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	var innerResp *apiv1.CreateExperimentResponse

	dbExp.ExternalExperimentID = &req.ExternalExperimentId

	innerResp, err = a.createUnmanagedExperimentTx(
		ctx, db.Bun(), dbExp, modelDef, activeConfig, user,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create unmanaged experiment: %w", err)
	}

	resp := apiv1.PutExperimentResponse{
		Experiment: innerResp.Experiment,
		Config:     innerResp.Config,
	}

	return &resp, nil
}

var (
	defaultMetricsStreamPeriod = 30 * time.Second
	recheckAuthPeriod          = 5 * time.Minute
)

func (a *apiServer) ExpMetricNames(req *apiv1.ExpMetricNamesRequest,
	resp apiv1.Determined_ExpMetricNamesServer,
) error {
	// remove duplicate ids to improve performance
	slices.Sort(req.Ids)
	req.Ids = slices.Compact(req.Ids)

	if len(req.Ids) == 0 {
		return status.Error(codes.InvalidArgument, "must specify at least one experiment id")
	}

	period := time.Duration(req.PeriodSeconds) * time.Second

	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	seenSearcher := make(map[string]bool)
	seenTrain := make(map[string]bool)
	seenValid := make(map[string]bool)
	seenMetrics := make(map[model.MetricGroup]map[string]bool)

	var timeSinceLastAuth time.Time
	for {
		var response apiv1.ExpMetricNamesResponse
		if time.Since(timeSinceLastAuth) >= recheckAuthPeriod {
			for _, expID := range req.Ids {
				exp, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), int(expID),
					experiment.AuthZProvider.Get().CanGetExperimentArtifacts)
				if err != nil {
					return err
				}

				if timeSinceLastAuth.Equal((time.Time{})) { // Initialization.
					searcherMetric := exp.Config.Searcher.Metric

					if seen := seenSearcher[searcherMetric]; !seen {
						response.SearcherMetrics = append(response.SearcherMetrics, searcherMetric)
						seenSearcher[searcherMetric] = true
					}
				}
			}
			timeSinceLastAuth = time.Now()
		}
		expIDs := make([]int, len(req.Ids))
		for i, ID := range req.Ids {
			expIDs[i] = int(ID)
		}

		metricNames, err := a.m.db.MetricNames(resp.Context(), expIDs)
		if err != nil {
			return errors.Wrapf(err,
				"error fetching metric names for experiment: %d", req.Ids)
		}

		for _, name := range metricNames[model.TrainingMetricGroup] {
			if seen := seenTrain[name]; !seen {
				//nolint:staticcheck // SA1019: backward compatibility
				response.TrainingMetrics = append(response.TrainingMetrics, name)
				seenTrain[name] = true
			}
		}
		for _, name := range metricNames[model.ValidationMetricGroup] {
			if seen := seenValid[name]; !seen {
				//nolint:staticcheck // SA1019: backward compatibility
				response.ValidationMetrics = append(response.ValidationMetrics, name)
				seenValid[name] = true
			}
		}
		for metricGroup, names := range metricNames {
			for _, name := range names {
				if seen := seenMetrics[metricGroup][name]; !seen {
					typedMetric := metricv1.MetricIdentifier{
						Group: metricGroup.ToString(),
						Name:  name,
					}
					response.MetricNames = append(response.MetricNames, &typedMetric)
					if seenMetrics[metricGroup] == nil {
						seenMetrics[metricGroup] = make(map[string]bool)
					}
					seenMetrics[metricGroup][name] = true
				}
			}
		}

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return err
		}

		numNonTerminalExperiments, err := db.GetNonTerminalExperimentCount(resp.Context(), req.Ids)
		if err != nil {
			return errors.Wrap(err, "error looking up state of experiments")
		}

		if numNonTerminalExperiments == 0 {
			return nil
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) MetricBatches(req *apiv1.MetricBatchesRequest,
	resp apiv1.Determined_MetricBatchesServer,
) error {
	experimentID := int(req.ExperimentId)
	metricName := req.MetricName
	if metricName == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric name")
	}
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	var timeSinceLastAuth time.Time
	seenBatches := make(map[int32]bool)
	var startTime time.Time
	for {
		if time.Since(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), experimentID,
				experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return err
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.MetricBatchesResponse

		var newBatches []int32
		var endTime time.Time
		var err error
		//nolint:staticcheck // SA1019: backward compatibility
		metricGroup, err := a.parseMetricGroupArgs(req.MetricType, model.MetricGroup(req.Group))
		if err != nil {
			return err
		}
		if metricGroup == "" {
			return status.Error(codes.InvalidArgument, "must specify a metric group")
		}
		newBatches, endTime, err = db.MetricBatches(experimentID, metricName, startTime, metricGroup)
		if err != nil {
			return errors.Wrapf(err, "error fetching batches recorded for metric")
		}
		startTime = endTime

		for _, batch := range newBatches {
			if seen := seenBatches[batch]; !seen {
				response.Batches = append(response.Batches, batch)
				seenBatches[batch] = true
			}
		}

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return errors.Wrapf(err, "error sending batches recorded for metric")
		}

		state, _, err := a.m.db.GetExperimentStatus(experimentID)
		if err != nil {
			return errors.Wrap(err, "error looking up experiment state")
		}
		if model.TerminalStates[state] {
			return nil
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) TrialsSnapshot(req *apiv1.TrialsSnapshotRequest,
	resp apiv1.Determined_TrialsSnapshotServer,
) error {
	experimentID := int(req.ExperimentId)
	metricName := req.MetricName
	if metricName == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric name")
	}

	var metricGroup model.MetricGroup
	//nolint:staticcheck // SA1019: backward compatibility
	switch req.MetricType {
	case apiv1.MetricType_METRIC_TYPE_TRAINING:
		metricGroup = model.TrainingMetricGroup //nolint:goconst
	case apiv1.MetricType_METRIC_TYPE_VALIDATION:
		metricGroup = model.ValidationMetricGroup //nolint:goconst
	default:
		metricGroup = model.MetricGroup(req.Group)
	}
	if metricGroup == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric group")
	}

	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	batchesProcessed := int(req.BatchesProcessed)
	batchesMargin := int(req.BatchesMargin)
	if batchesMargin > 100 {
		return status.Error(codes.InvalidArgument, "margin must be <= 100")
	}
	minBatches := batchesProcessed - batchesMargin
	if minBatches < 0 {
		minBatches = 0
	}
	maxBatches := batchesProcessed + batchesMargin
	if maxBatches < 0 {
		maxBatches = math.MaxInt32
	}

	var timeSinceLastAuth time.Time
	var startTime time.Time
	for {
		if time.Since(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), experimentID,
				experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return err
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.TrialsSnapshotResponse
		newTrials, endTime, err := a.m.db.TrialsSnapshot(experimentID,
			minBatches, maxBatches, metricName, startTime, metricGroup)
		if err != nil {
			return errors.Wrapf(err,
				"error fetching snapshots of metrics for %s metric %s in experiment %d at %d batches",
				metricGroup, metricName, experimentID, batchesProcessed)
		}
		startTime = endTime

		response.Trials = newTrials

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return errors.Wrapf(err, "error sending batches recorded for metrics")
		}

		state, _, err := a.m.db.GetExperimentStatus(experimentID)
		if err != nil {
			return errors.Wrap(err, "error looking up experiment state")
		}
		if model.TerminalStates[state] {
			return nil
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) topTrials(
	ctx context.Context, experimentID int, maxTrials int, s expconf.LegacySearcher,
) (trials []int32, err error) {
	type Ranking int
	const (
		byMetricOfInterest Ranking = 1
		byTrainingLength   Ranking = 2
	)
	var ranking Ranking

	switch s.Name {
	case "random":
		ranking = byMetricOfInterest
	case "grid":
		ranking = byMetricOfInterest
	case "custom":
		ranking = byMetricOfInterest
	case "async_halving":
		ranking = byTrainingLength
	case "adaptive_asha":
		ranking = byTrainingLength
	case "single":
		return nil, fmt.Errorf("single-trial experiments are not supported for trial sampling")
	// EOL searcher configs:
	case "adaptive":
		ranking = byTrainingLength
	case "adaptive_simple":
		ranking = byTrainingLength
	case "sync_halving":
		ranking = byTrainingLength
	default:
		return nil, errors.Errorf("unable to detect a searcher algorithm for trial sampling")
	}
	switch ranking {
	case byMetricOfInterest:
		return db.TopTrialsByMetric(ctx, experimentID, maxTrials, s.Metric, s.SmallerIsBetter)
	case byTrainingLength:
		return a.m.db.TopTrialsByTrainingLength(experimentID, maxTrials, s.Metric, s.SmallerIsBetter)
	default:
		panic("Invalid state in trial sampling")
	}
}

func (a *apiServer) fetchTrialSample(trialID int32, metricName string, metricGroup model.MetricGroup,
	maxDatapoints int, startBatches int, endBatches int, currentTrials map[int32]bool,
	trialCursors map[int32]time.Time,
) (*apiv1.TrialsSampleResponse_Trial, error) {
	var endTime time.Time
	var zeroTime time.Time
	var err error
	var trial apiv1.TrialsSampleResponse_Trial
	var metricMeasurements []db.MetricMeasurements
	xAxisLabelMetrics := []string{"epoch"}

	trial.TrialId = trialID

	if _, current := currentTrials[trialID]; !current {
		var trialConfig *model.Trial
		trialConfig, err = db.TrialByID(context.TODO(), int(trialID))
		if err != nil {
			return nil, errors.Wrapf(err, "error fetching trial metadata")
		}
		trial.Hparams = protoutils.ToStruct(trialConfig.HParams)
	}

	startTime, seenBefore := trialCursors[trialID]
	if !seenBefore {
		startTime = zeroTime
	}
	metricMeasurements, err = trials.MetricsTimeSeries(trialID, startTime,
		[]string{metricName},
		startBatches, endBatches, xAxisLabelMetrics, maxDatapoints,
		"batches", nil, metricGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "error fetching time series of metrics")
	}
	if len(metricMeasurements) > 0 {
		// if we get empty results, the endTime is incorrectly zero
		trialCursors[trialID] = endTime
	}

	if !seenBefore {
		for _, in := range metricMeasurements {
			valueMap, err := structpbmap.NewStruct(in.Values)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse metric values")
			}
			out := apiv1.DataPoint{
				Batches: int32(in.Batches),
				Values:  valueMap,
				Time:    timestamppb.New(in.Time),
				Epoch:   in.Epoch,
			}
			trial.Data = append(trial.Data, &out)
		}
	}

	return &trial, nil
}

func (a *apiServer) TrialsSample(req *apiv1.TrialsSampleRequest,
	resp apiv1.Determined_TrialsSampleServer,
) error {
	experimentID := int(req.ExperimentId)
	maxTrials := int(req.MaxTrials)
	if maxTrials == 0 {
		maxTrials = 25
	}
	maxDatapoints := int(req.MaxDatapoints)
	if maxDatapoints == 0 {
		maxDatapoints = 1000
	}
	startBatches := int(req.StartBatches)
	endBatches := int(req.EndBatches)
	if endBatches <= 0 {
		endBatches = math.MaxInt32
	}
	period := time.Duration(req.PeriodSeconds) * time.Second
	if period == 0 {
		period = defaultMetricsStreamPeriod
	}

	metricName := req.MetricName
	if metricName == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric name")
	}

	var metricGroup model.MetricGroup
	//nolint:staticcheck // SA1019: backward compatibility
	switch req.MetricType {
	case apiv1.MetricType_METRIC_TYPE_TRAINING:
		metricGroup = model.TrainingMetricGroup
	case apiv1.MetricType_METRIC_TYPE_VALIDATION:
		metricGroup = model.ValidationMetricGroup
	default:
		metricGroup = model.MetricGroup(req.Group)
	}
	if metricGroup == "" {
		return status.Error(codes.InvalidArgument, "must specify a metric group")
	}

	var timeSinceLastAuth time.Time
	var searcherConfig expconf.LegacySearcher
	trialCursors := make(map[int32]time.Time)
	currentTrials := make(map[int32]bool)
	for {
		if time.Since(timeSinceLastAuth) >= recheckAuthPeriod {
			exp, _, err := a.getExperimentAndCheckCanDoActions(resp.Context(), experimentID,
				experiment.AuthZProvider.Get().CanGetExperimentArtifacts)
			if err != nil {
				return err
			}

			if timeSinceLastAuth.IsZero() { // Initialzation.
				searcherConfig = exp.Config.Searcher
			}
			timeSinceLastAuth = time.Now()
		}

		var response apiv1.TrialsSampleResponse
		var promotedTrials []int32
		var demotedTrials []int32
		var trials []*apiv1.TrialsSampleResponse_Trial

		seenThisRound := make(map[int32]bool)

		trialIDs, err := a.topTrials(resp.Context(), experimentID, maxTrials, searcherConfig)
		if err != nil {
			return errors.Wrapf(err, "error determining top trials")
		}
		for _, trialID := range trialIDs {
			var trial *apiv1.TrialsSampleResponse_Trial
			trial, err = a.fetchTrialSample(trialID, metricName, metricGroup, maxDatapoints,
				startBatches, endBatches, currentTrials, trialCursors)
			if err != nil {
				return err
			}

			if _, current := currentTrials[trialID]; !current {
				promotedTrials = append(promotedTrials, trialID)
				currentTrials[trialID] = true
			}
			seenThisRound[trialID] = true

			trials = append(trials, trial)
		}
		for oldTrial := range currentTrials {
			if !seenThisRound[oldTrial] {
				demotedTrials = append(demotedTrials, oldTrial)
				delete(trialCursors, oldTrial)
			}
		}
		// Deletes from currentTrials have to happen when not looping over currentTrials
		for _, oldTrial := range demotedTrials {
			delete(currentTrials, oldTrial)
		}

		response.Trials = trials
		response.PromotedTrials = promotedTrials
		response.DemotedTrials = demotedTrials

		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
		if err = resp.Send(&response); err != nil {
			return errors.Wrap(err, "error sending sample of trial metric streams")
		}

		state, _, err := a.m.db.GetExperimentStatus(experimentID)
		if err != nil {
			return errors.Wrap(err, "error looking up experiment state")
		}
		if model.TerminalStates[state] {
			return nil
		}

		time.Sleep(period)
		if grpcutil.ConnectionIsClosed(resp) {
			return nil
		}
	}
}

func (a *apiServer) GetBestSearcherValidationMetric(
	ctx context.Context, req *apiv1.GetBestSearcherValidationMetricRequest,
) (*apiv1.GetBestSearcherValidationMetricResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	metric, err := db.ExperimentBestSearcherValidation(ctx, int(req.ExperimentId))
	switch {
	case errors.Is(err, db.ErrNotFound):
		return nil, status.Errorf(codes.NotFound, "no validations for experiment")
	case err != nil:
		return nil, err
	}

	return &apiv1.GetBestSearcherValidationMetricResponse{
		Metric: metric,
	}, nil
}

func (a *apiServer) GetModelDef(
	ctx context.Context, req *apiv1.GetModelDefRequest,
) (*apiv1.GetModelDefResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	tgz, err := a.m.db.ExperimentModelDefinitionRaw(int(req.ExperimentId))
	if err != nil {
		return nil, errors.Wrapf(err,
			"error fetching model definition from database: %d", req.ExperimentId)
	}

	b64Tgz := base64.StdEncoding.EncodeToString(tgz)

	return &apiv1.GetModelDefResponse{B64Tgz: b64Tgz}, nil
}

func (a *apiServer) GetTaskContextDirectory(
	ctx context.Context, req *apiv1.GetTaskContextDirectoryRequest,
) (*apiv1.GetTaskContextDirectoryResponse, error) {
	if _, _, err := a.canDoActionsOnTask(ctx, model.TaskID(req.TaskId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	isExp, exp, err := expFromTaskID(ctx, model.TaskID(req.TaskId))
	if err != nil {
		return nil, err
	}

	var tgz []byte
	if isExp {
		tgz, err = a.m.db.ExperimentModelDefinitionRaw(exp.ID)
		if err != nil {
			return nil, fmt.Errorf(
				"fetching experiment's taskID %s model definition from database: %w",
				req.TaskId, err)
		}
	} else {
		tgz, err = db.NonExperimentTasksContextDirectory(ctx, model.TaskID(req.TaskId))
		if err != nil {
			return nil, fmt.Errorf(
				"fetching taskID %s context directory from database: %s", req.TaskId, err)
		}
	}

	return &apiv1.GetTaskContextDirectoryResponse{
		B64Tgz: base64.StdEncoding.EncodeToString(tgz),
	}, nil
}

func (a *apiServer) MoveExperiment(
	ctx context.Context, req *apiv1.MoveExperimentRequest,
) (*apiv1.MoveExperimentResponse, error) {
	// get experiment info
	exp, curUser, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId))
	if err != nil {
		return nil, err
	}
	if exp.Archived {
		return nil, errors.Errorf("experiment (%v) is archived and cannot be moved", exp.ID)
	}

	// check that user can view source project
	srcProject, err := a.GetProjectByID(ctx, int32(exp.ProjectID), curUser)
	if err != nil {
		return nil, err
	}
	if srcProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot have experiments moved from it",
			srcProject.Id)
	}

	// check suitable destination project
	destProject, err := a.GetProjectByID(ctx, req.DestinationProjectId, curUser)
	if err != nil {
		return nil, err
	}
	if destProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot add new experiments",
			req.DestinationProjectId)
	}
	if err = experiment.AuthZProvider.Get().CanCreateExperiment(ctx, curUser, destProject); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	results, err := experiment.MoveExperiments(
		ctx,
		experiment.GlobalProjectID,
		[]int32{req.ExperimentId},
		nil,
		req.DestinationProjectId,
	)
	if err == nil {
		if len(results) == 0 {
			return nil, errors.Errorf("MoveExperiments returned neither pass nor fail on query")
		} else if results[0].Error != nil {
			return nil, results[0].Error
		}
	}

	return &apiv1.MoveExperimentResponse{}, err
}

func (a *apiServer) MoveExperiments(
	ctx context.Context, req *apiv1.MoveExperimentsRequest,
) (*apiv1.MoveExperimentsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	// check suitable destination project
	destProject, err := a.GetProjectByID(ctx, req.DestinationProjectId, *curUser)
	if err != nil {
		return nil, err
	}
	if destProject.Archived {
		return nil, errors.Errorf("project (%v) is archived and cannot add new experiments",
			req.DestinationProjectId)
	}
	if err = experiment.AuthZProvider.Get().CanCreateExperiment(ctx, *curUser, destProject); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	results, err := experiment.MoveExperiments(ctx, req.ProjectId, req.ExperimentIds,
		req.Filters, req.DestinationProjectId)
	return &apiv1.MoveExperimentsResponse{Results: experiment.ToAPIResults(results)}, err
}

func (a *apiServer) GetModelDefTree(
	ctx context.Context, req *apiv1.GetModelDefTreeRequest,
) (*apiv1.GetModelDefTreeResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	modelDefCache := GetModelDefCache()
	fileTree, err := modelDefCache.FileTreeNested(int(req.ExperimentId))
	if err != nil {
		return nil, err
	}
	return &apiv1.GetModelDefTreeResponse{Files: fileTree}, nil
}

func (a *apiServer) GetModelDefFile(
	ctx context.Context, req *apiv1.GetModelDefFileRequest,
) (*apiv1.GetModelDefFileResponse, error) {
	if _, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	modelDefCache := GetModelDefCache()
	file, err := modelDefCache.FileContent(int(req.ExperimentId), req.Path)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetModelDefFileResponse{File: file}, nil
}

func sortExperiments(sortString *string, experimentQuery *bun.SelectQuery) error {
	if sortString == nil {
		return nil
	}
	orderColMap := map[string]string{
		"id":              "id",
		"description":     "description",
		"name":            "name",
		"searcherType":    "searcher_type",
		"searcherMetric":  "searcher_metric",
		"startTime":       "e.start_time",
		"endTime":         "e.end_time",
		"state":           "e.state",
		"numTrials":       "num_trials",
		"progress":        "COALESCE(progress, 0)",
		"user":            "display_name",
		"forkedFrom":      "e.parent_id",
		"resourcePool":    "resource_pool",
		"projectId":       "e.project_id",
		"checkpointSize":  "checkpoint_size",
		"checkpointCount": "checkpoint_count",
		"duration":        "duration",
		"searcherMetricsVal": `(
			SELECT
				searcher_metric_value
			FROM trials t
			WHERE t.experiment_id = e.id
			ORDER BY searcher_metric_value_signed ASC
			LIMIT 1
		) `,
		"externalExperimentId": "e.external_experiment_id",
		"externalTrialId":      "r.external_run_id",
	}
	sortByMap := map[string]string{
		"asc":  "ASC",
		"desc": "DESC NULLS LAST",
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
			hps := strings.ReplaceAll(strings.TrimPrefix(param, "hp."), ".", "'->'")
			experimentQuery.OrderExpr(
				fmt.Sprintf("e.config->'hyperparameters'->'%s' %s", hps, sortDirection))
		case strings.Contains(paramDetail[0], "."):
			metricGroup, metricName, metricQualifier, err := parseMetricsName(paramDetail[0])
			if err != nil {
				return err
			}
			experimentQuery.OrderExpr("r.summary_metrics->?->?->>? ?",
				metricGroup, metricName, metricQualifier, bun.Safe(sortDirection))
		default:
			if _, ok := orderColMap[paramDetail[0]]; !ok {
				return status.Errorf(codes.InvalidArgument, "invalid sort col: %s", paramDetail[0])
			}
			hasIDSort = hasIDSort || paramDetail[0] == "id"
			experimentQuery.OrderExpr(
				fmt.Sprintf("%s %s", orderColMap[paramDetail[0]], sortDirection))
		}
	}
	if !hasIDSort {
		experimentQuery.OrderExpr("id ASC")
	}
	return nil
}

func (a *apiServer) SearchExperiments(
	ctx context.Context,
	req *apiv1.SearchExperimentsRequest,
) (*apiv1.SearchExperimentsResponse, error) {
	resp := &apiv1.SearchExperimentsResponse{}
	var experiments []*experimentv1.Experiment
	var trials []*trialv1.Trial
	experimentQuery := db.Bun().NewSelect().
		Model(&experiments).
		ModelTableExpr("experiments as e").
		Column("e.best_trial_id").
		Where("e.state != ?", model.DeletingState).
		Apply(getExperimentColumns)

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	var proj *projectv1.Project
	if req.ProjectId != nil {
		proj, err = a.GetProjectByID(ctx, *req.ProjectId, *curUser)
		if err != nil {
			return nil, err
		}

		experimentQuery = experimentQuery.Where("e.project_id = ?", req.ProjectId)
	}
	if experimentQuery, err = experiment.AuthZProvider.Get().
		FilterExperimentsQuery(ctx, *curUser, proj, experimentQuery,
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
		experimentQuery = experimentQuery.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
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
		err = sortExperiments(req.Sort, experimentQuery)
		if err != nil {
			return nil, err
		}
	} else {
		experimentQuery.OrderExpr("id ASC")
	}

	resp.Pagination, err = runPagedBunExperimentsQuery(
		ctx,
		experimentQuery,
		int(req.Offset),
		int(req.Limit),
	)
	if err != nil {
		return nil, err
	}

	if len(experiments) == 0 {
		return resp, nil
	}

	if err = a.enrichExperimentState(experiments...); err != nil {
		return nil, err
	}

	// get the best trial associated with the experiment.
	trialIDs := make([]int32, 0, len(experiments)+1)
	trialIDs = append(trialIDs, -1) // hack to avoid bun error for 0 length array.
	for _, e := range experiments {
		if e.BestTrialId != nil {
			trialIDs = append(trialIDs, *e.BestTrialId)
		}
	}

	trialsInnerQuery := db.Bun().NewSelect().
		Table("trials").
		Column("trials.id").
		Column("trials.experiment_id").
		Column("trials.runner_state").
		Column("trials.checkpoint_count").
		Column("trials.summary_metrics").
		Column("trials.log_retention_days").
		ColumnExpr(`(
				SELECT tt.task_id FROM run_id_task_id tt
				JOIN tasks ta ON tt.task_id = ta.task_id
				WHERE tt.run_id = trials.id
				ORDER BY ta.start_time
				LIMIT 1
			) AS task_id`).
		ColumnExpr(`(
				(SELECT json_agg(task_id) FROM (
					SELECT tt.task_id FROM run_id_task_id tt
					JOIN tasks ta ON tt.task_id = ta.task_id
					WHERE tt.run_id = trials.id
					ORDER BY ta.start_time
				) sub_tasks)) AS task_ids`).
		ColumnExpr("proto_time(trials.start_time) AS start_time").
		ColumnExpr("proto_time(trials.end_time) AS end_time").
		Column("trials.restarts").
		ColumnExpr("new_ckpt.uuid AS warm_start_checkpoint_uuid").
		ColumnExpr("trials.checkpoint_size AS total_checkpoint_size").
		ColumnExpr(bunutils.ProtoStateDBCaseString(trialv1.State_value, "trials.state", "state",
			"STATE_")).
		ColumnExpr(`(CASE WHEN trials.hparams = 'null'::jsonb
				THEN null ELSE trials.hparams END) AS hparams`).
		ColumnExpr("trials.total_batches AS total_batches_processed").
		ColumnExpr(`CASE WHEN trials.latest_validation_id IS NULL THEN NULL ELSE jsonb_build_object(
				'trial_id', trials.id,
				'total_batches', lv.total_batches,
				'end_time', proto_time(lv.end_time),
				'metrics', json_build_object('avg_metrics', lv.metrics->'validation_metrics'),
				'num_inputs', lv.metrics->'num_inputs') END AS latest_validation`).
		ColumnExpr(`jsonb_build_object(
				'trial_id', trials.id,
				'total_batches', bv.total_batches,
				'end_time', proto_time(bv.end_time),
				'metrics', json_build_object('avg_metrics', bv.metrics->'validation_metrics'),
				'num_inputs', bv.metrics->'num_inputs') AS best_validation`).
		ColumnExpr("null::jsonb AS best_checkpoint").
		ColumnExpr("null::jsonb AS wall_clock_time").
		Column("searcher_metric_value").
		Column("trials.external_trial_id").
		ColumnExpr("nullif(trials.metadata, 'null') as metadata").
		ColumnExpr("NULL as log_signal").
		Join("LEFT JOIN validations bv ON trials.best_validation_id = bv.id").
		Join("LEFT JOIN validations lv ON trials.latest_validation_id = lv.id").
		Join("LEFT JOIN checkpoints_v2 new_ckpt ON new_ckpt.id = trials.warm_start_checkpoint_id").
		Where("trials.id IN (?)", bun.In(trialIDs))

	err = db.Bun().NewSelect().
		Model(&trials).
		ModelTableExpr("(?) AS trial", trialsInnerQuery).Scan(ctx)
	if err != nil {
		return nil, err
	}

	trialsByExperimentID := make(map[int32]*trialv1.Trial, len(trials))
	for _, trial := range trials {
		trialsByExperimentID[trial.ExperimentId] = trial
	}
	for _, experiment := range experiments {
		trial := trialsByExperimentID[experiment.Id]
		if trial != nil {
			// Correct trial restarts because
			// `restart` count is incremented before `restart <= max_restarts` stop restart check,
			// so trials in terminal state have restarts = max + 1.
			// Update this correction to happen in the database when we do the remove.
			//nolint:staticcheck
			configRestarts, ok := experiment.Config.Fields["max_restarts"].AsInterface().(float64)
			if ok && trial.Restarts > int32(configRestarts) {
				trial.Restarts = int32(configRestarts)
			}
		}

		resp.Experiments = append(
			resp.Experiments,
			&apiv1.SearchExperimentExperiment{Experiment: experiment, BestTrial: trial},
		)
	}

	return resp, nil
}

func (a *apiServer) createTrialTx(
	ctx context.Context, tx bun.Tx, req *apiv1.CreateTrialRequest, externalTrialID *string,
) (*apiv1.CreateTrialResponse, error) {
	if !req.Unmanaged {
		return nil, fmt.Errorf("only unmanaged trials are supported")
	}

	exp, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		experiment.AuthZProvider.Get().CanEditExperiment)
	if err != nil {
		return nil, err
	}

	if !exp.Unmanaged {
		return nil, fmt.Errorf("trials can only be created on unmanaged experiments")
	}

	var taskIDSuffix string
	if externalTrialID == nil {
		// Persistent taskIDSuffix enables UPSERT in `AddTaskTx`.
		taskIDSuffix = model.NewTaskID().String()
	} else {
		taskIDSuffix = *externalTrialID
	}

	// HACK: needed for ``experimentIDFromTrialTaskID``.
	taskID := model.TaskID(fmt.Sprintf("%d.%s", exp.ID, taskIDSuffix))

	trialModel := model.NewTrial(
		model.PausedState,
		model.RequestID{},
		exp.ID,
		req.Hparams.AsMap(),
		nil,
		0,
		a.m.taskSpec.LogRetentionDays)

	if err := db.AddTask(ctx, &model.Task{
		TaskID:     taskID,
		TaskType:   model.TaskTypeTrial,
		StartTime:  time.Now(),
		JobID:      nil,
		LogVersion: model.CurrentTaskLogVersion,
	}); err != nil {
		return nil, err
	}

	if externalTrialID != nil {
		trialModel.ExternalTrialID = externalTrialID
	}

	if err := db.UpsertTrialByExternalIDTx(ctx, tx, trialModel, taskID); err != nil {
		return nil, err
	}

	trialRes, err := trials.ProtoGetTrialsPlusTx(ctx, tx, []int{trialModel.ID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get trial %d", trialModel.ID)
	}
	resp := &apiv1.CreateTrialResponse{Trial: trialRes[0]}

	return resp, nil
}

// CreateTrial creates a trial.
func (a *apiServer) CreateTrial(
	ctx context.Context, req *apiv1.CreateTrialRequest,
) (*apiv1.CreateTrialResponse, error) {
	var resp *apiv1.CreateTrialResponse
	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		res, err := a.createTrialTx(ctx, tx, req, nil)
		if err != nil {
			return err
		}
		resp = res

		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// PutTrial puts a trial.
func (a *apiServer) PutTrial(ctx context.Context, req *apiv1.PutTrialRequest) (
	*apiv1.PutTrialResponse, error,
) {
	if !req.CreateTrialRequest.Unmanaged {
		return nil, fmt.Errorf("only unmanaged trials are supported")
	}

	_, _, err := a.getExperimentAndCheckCanDoActions(ctx, int(req.CreateTrialRequest.ExperimentId),
		experiment.AuthZProvider.Get().CanEditExperiment)
	if err != nil {
		return nil, err
	}

	var trial *trialv1.Trial

	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		innerResp, err := a.createTrialTx(ctx, tx, req.CreateTrialRequest, &req.ExternalTrialId)
		if err != nil {
			return fmt.Errorf("failed to create trial: %w", err)
		}
		trial = innerResp.Trial
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run create trial tx: %w", err)
	}

	resp := &apiv1.PutTrialResponse{Trial: trial}

	return resp, nil
}

// PatchTrial patches a trial.
func (a *apiServer) PatchTrial(ctx context.Context, req *apiv1.PatchTrialRequest) (
	*apiv1.PatchTrialResponse, error,
) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	trialID := int(req.TrialId)
	// TODO(ilia): Since this experiment can be updated later by
	// `UpdateUnmanagedExperimentStatesTx`, ideally we'll lock it with `FOR UPDATE`.
	// But the metrics code currently locks trials first, then inside `setTrialBestValidation`
	// it'll lock the experiments, causing a deadlock.
	// Instead, we are planning to denormalize, store the searcher metric on the trials,
	// so `trial.best_validation` can be computed by itself.
	exp, err := db.ExperimentByTrialID(ctx, trialID)
	if err != nil {
		return nil, err
	}

	if err = experiment.AuthZProvider.Get().CanEditExperimentsMetadata(
		ctx, *curUser, exp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	if !exp.Unmanaged {
		return nil, fmt.Errorf("only unmanaged trials are supported")
	}

	obj := model.Run{
		ID:           trialID,
		LastActivity: ptrs.Ptr(time.Now()),
	}

	columns := []string{"last_activity"}

	if req.State != nil {
		obj.State = model.State(strings.TrimPrefix(req.State.String(), "STATE_"))
		columns = append(columns, "state")
		if model.TerminalStates[obj.State] {
			obj.EndTime = ptrs.Ptr(time.Now())
			columns = append(columns, "end_time")
		}
	}

	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(&obj).
			Column(columns...).
			WherePK().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update trial state: %w", err)
		}

		err = trials.UpdateUnmanagedExperimentStatesTx(ctx, tx, []*model.Experiment{exp})
		if err != nil {
			return fmt.Errorf("failed to update experiment state: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update state: %w", err)
	}

	resp := &apiv1.PatchTrialResponse{Trial: &trialv1.Trial{}}

	if err := a.m.db.QueryProtof(
		"proto_get_trials_plus",
		[]any{"($1::int, $2::int)"},
		resp.Trial,
		trialID,
		1,
	); err != nil {
		return nil, fmt.Errorf("failed to get trial %d: %w", trialID, err)
	}

	return resp, nil
}

func (a *apiServer) PutExperimentLabel(ctx context.Context,
	req *apiv1.PutExperimentLabelRequest,
) (*apiv1.PutExperimentLabelResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Error("error rolling back transaction in create workspace")
		}
	}()

	exp := &experimentv1.Experiment{}
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(exp).
		Apply(getExperimentColumns).
		Where("e.id = ?", req.ExperimentId)
	if err = query.Scan(ctx); err != nil {
		return nil, err
	}
	modelExp, err := model.ExperimentFromProto(exp)
	if err != nil {
		return nil, err
	}

	if err = experiment.AuthZProvider.Get().CanEditExperimentsMetadata(
		ctx, *curUser, modelExp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	if slices.Contains(exp.Labels, req.Label) {
		return &apiv1.PutExperimentLabelResponse{Labels: exp.Labels}, nil
	}
	exp.Labels = append(exp.Labels, req.Label)

	_, err = tx.NewUpdate().Model(modelExp).
		Set("config = jsonb_set(config, '{labels}', ?, true)", exp.Labels).
		Where("id = ?", exp.Id).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating experiment %v in database %w", exp.Id, err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit patch experiment labels transaction %w", err)
	}

	return &apiv1.PutExperimentLabelResponse{Labels: exp.Labels}, nil
}

func (a *apiServer) DeleteExperimentLabel(ctx context.Context,
	req *apiv1.DeleteExperimentLabelRequest,
) (*apiv1.DeleteExperimentLabelResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.WithError(err).Error("error rolling back transaction in create workspace")
		}
	}()

	exp := &experimentv1.Experiment{}
	query := db.Bun().NewSelect().
		ModelTableExpr("experiments as e").
		Model(exp).
		Apply(getExperimentColumns).
		Where("e.id = ?", req.ExperimentId).
		For("UPDATE")
	if err = query.Scan(ctx); err != nil {
		return nil, err
	}

	modelExp, err := model.ExperimentFromProto(exp)
	if err != nil {
		return nil, err
	}

	if err = experiment.AuthZProvider.Get().CanEditExperimentsMetadata(
		ctx, *curUser, modelExp); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}

	i := slices.Index(exp.Labels, req.Label)
	if i == -1 {
		return &apiv1.DeleteExperimentLabelResponse{Labels: exp.Labels}, nil
	}
	exp.Labels = slices.Delete(exp.Labels, i, i+1)

	_, err = tx.NewUpdate().Model(modelExp).
		Set("config = jsonb_set(config, '{labels}', ?, true)", exp.Labels).
		Where("id = ?", exp.Id).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating experiment %v in database: %w", exp.Id, err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit delete experiment labels transaction: %w", err)
	}

	return &apiv1.DeleteExperimentLabelResponse{Labels: exp.Labels}, nil
}

func (a *apiServer) DeleteTensorboardFiles(
	ctx context.Context, req *apiv1.DeleteTensorboardFilesRequest,
) (resp *apiv1.DeleteTensorboardFilesResponse, err error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	exp, err := db.ExperimentByID(ctx, int(req.ExperimentId))
	if err != nil {
		return nil, err
	}

	workspaceID, err := workspace.WorkspacesIDsByExperimentIDs(ctx, []int{exp.ID})
	if err != nil {
		return nil, err
	}
	agentUserGroup, err := user.GetAgentUserGroup(ctx, *exp.OwnerID, workspaceID[0])
	if err != nil {
		return nil, err
	}

	var uuidList []uuid.UUID
	err = runCheckpointGCTask(
		a.m.rm, a.m.db, model.NewTaskID(), exp.JobID, exp.StartTime, *a.m.taskSpec, exp.ID,
		exp.Config, nil, uuidList, nil, true, agentUserGroup, curUser,
		nil,
	)
	if err != nil {
		log.WithError(err).Errorf("failed to gc tensorboard for experiment: %d", exp.ID)
		return nil, err
	}

	return &apiv1.DeleteTensorboardFilesResponse{}, nil
}
