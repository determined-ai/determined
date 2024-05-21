package internal

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoless"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
	batches                       = "batches"
	trialLogsBatchSize            = 1000
	trialProfilerMetricsBatchSize = 100
)

var (
	trialLogsBatchMissWaitTime = time.Second

	distinctFieldBatchWaitTime = 5 * time.Second

	trialProfilerMetricsBatchMissWaitTime = 5 * time.Second

	// TrialAvailableSeriesBatchWaitTime is exported to be changed by tests.
	TrialAvailableSeriesBatchWaitTime = 15 * time.Second
)

// TrialLogBackend is an interface trial log backends, such as elastic or postgres,
// must support to provide the features surfaced in API. This is deprecated, note it
// no longer supports adding logs in favor of unified logs.
type TrialLogBackend interface {
	TrialLogs(
		trialID, limit int, filters []api.Filter, order apiv1.OrderBy, state interface{},
	) ([]*model.TrialLog, interface{}, error)
	TrialLogsCount(trialID int, filters []api.Filter) (int, error)
	TrialLogsFields(trialID int) (*apiv1.TrialLogsFieldsResponse, error)
	DeleteTrialLogs(trialIDs []int) error
}

// Catches information on active running trials.
type trialAllocation struct {
	Pulling  bool
	Running  bool
	Starting bool
	Task     model.TaskID
}

func getLatestTaskIDFromTrialProto(t *trialv1.Trial) model.TaskID {
	if len(t.TaskIds) == 0 {
		panic(fmt.Sprintf("trial proto object %+v was without associated task", t))
	}

	return model.TaskID(t.TaskIds[len(t.TaskIds)-1])
}

func (a *apiServer) enrichTrialState(trials ...*trialv1.Trial) error {
	// filter allocations by TaskIDs on this page of trials
	taskFilter := make([]string, 0, len(trials))
	for _, trial := range trials {
		taskFilter = append(taskFilter, string(getLatestTaskIDFromTrialProto(trial)))
	}

	// get active trials by TaskId
	tasks := []trialAllocation{}
	err := a.m.db.Query(
		"aggregate_allocation_state_by_task",
		&tasks,
		strings.Join(taskFilter, ","),
	)
	if err != nil {
		return err
	}

	// Collect state information by TaskID
	byTaskID := make(map[model.TaskID]trialv1.State, len(tasks))
	for _, task := range tasks {
		switch {
		case task.Running:
			byTaskID[task.Task] = trialv1.State_STATE_RUNNING
		case task.Starting:
			byTaskID[task.Task] = trialv1.State_STATE_STARTING
		case task.Pulling:
			byTaskID[task.Task] = trialv1.State_STATE_PULLING
		default:
			byTaskID[task.Task] = trialv1.State_STATE_QUEUED
		}
	}

	// Active trials converted to Queued, Pulling, Starting, or Running
	for _, trial := range trials {
		if trial.State == trialv1.State_STATE_ACTIVE {
			if setState, ok := byTaskID[getLatestTaskIDFromTrialProto(trial)]; ok {
				trial.State = setState
			} else {
				trial.State = trialv1.State_STATE_QUEUED
			}
		}
	}
	return nil
}

func (a *apiServer) TrialLogs(
	req *apiv1.TrialLogsRequest, resp apiv1.Determined_TrialLogsServer,
) error {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(resp.Context(), int(req.TrialId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return err
	}

	trialTaskIDs, err := db.TrialTaskIDsByTrialID(resp.Context(), int(req.TrialId))
	if err != nil {
		return fmt.Errorf("retreiving task IDs for trial log: %w", err)
	}
	if len(trialTaskIDs) == 0 {
		return fmt.Errorf("no task IDs associated with trial ID %d", req.TrialId)
	}
	var tasks []*model.Task
	for _, t := range trialTaskIDs {
		task, err := db.TaskByID(resp.Context(), t.TaskID)
		if err != nil {
			return fmt.Errorf("getting task version for trial logs for task ID %s: %w", t.TaskID, err)
		}
		tasks = append(tasks, task)
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()
	// First stream legacy logs.
	if tasks[0].LogVersion == model.TaskLogVersion0 {
		res := make(chan api.BatchResult, taskLogsChanBuffer)
		go a.legacyTrialLogs(ctx, req, res)
		if err := processBatches(res, func(b api.Batch) error {
			return b.ForEach(func(i interface{}) error {
				l, err := i.(*model.TrialLog).Proto()
				if err != nil {
					return err
				}
				return resp.Send(l)
			})
		}); err != nil {
			return err
		}
		// Don't return here. Continue in case this task spanned the upgrade. If it did not,
		// it will return quickly anyway.
	}

	for i, task := range tasks {
		switch task.LogVersion {
		case model.TaskLogVersion0, model.TaskLogVersion1:
			// Translate the request.
			res := make(chan api.BatchResult, taskLogsChanBuffer)
			go a.taskLogs(ctx, &apiv1.TaskLogsRequest{
				TaskId: string(task.TaskID),
				Limit:  req.Limit,
				// Only follow on the final task log. We should at most only have
				// one non terminal task per trial. But if this assumption is violated
				// users will be prevented from seeing future logs on running tasks without this
				// hack here.
				Follow:          req.Follow && i == len(trialTaskIDs)-1,
				AllocationIds:   nil,
				AgentIds:        req.AgentIds,
				ContainerIds:    req.ContainerIds,
				RankIds:         req.RankIds,
				Levels:          req.Levels,
				Stdtypes:        req.Stdtypes,
				Sources:         req.Sources,
				TimestampBefore: req.TimestampBefore,
				TimestampAfter:  req.TimestampAfter,
				OrderBy:         req.OrderBy,
				SearchText:      req.SearchText,
			}, res)
			err := processBatches(res, func(b api.Batch) error {
				return b.ForEach(func(i interface{}) error {
					l, err := i.(*model.TaskLog).Proto()
					if err != nil {
						return err
					}
					return resp.Send(&apiv1.TrialLogsResponse{
						Id:          l.Id,
						TrialId:     req.TrialId,
						Timestamp:   l.Timestamp,
						Message:     l.Message, //nolint: staticcheck // l.Message is deprecated.
						Level:       l.Level,
						AgentId:     l.AgentId,
						ContainerId: l.ContainerId,
						RankId:      l.RankId,
						Log:         &l.Log,
						Source:      l.Source,
						Stdtype:     l.Stdtype,
					})
				})
			})
			if err != nil {
				return fmt.Errorf("getting trial logs for task ID %s: %w", task.TaskID, err)
			}
		default:
			panic(fmt.Errorf("unknown task log version: %d, please report this bug", task.LogVersion))
		}
	}

	return nil
}

func (a *apiServer) legacyTrialLogs(
	ctx context.Context, req *apiv1.TrialLogsRequest, res chan api.BatchResult,
) {
	if err := grpcutil.ValidateRequest(
		grpcutil.ValidateLimit(req.Limit),
		grpcutil.ValidateFollow(req.Limit, req.Follow),
	); err != nil {
		res <- api.ErrBatchResult(err)
		return
	}

	filters, err := constructTrialLogsFilters(req)
	if err != nil {
		res <- api.ErrBatchResult(errors.Wrap(err, "unsupported filter"))
		return
	}

	var followState interface{}
	trialLogsTimeSinceLastAuth := time.Now() // time.Now() to avoid recheck from a.TrialLogs.
	fetch := func(r api.BatchRequest) (api.Batch, error) {
		if time.Since(trialLogsTimeSinceLastAuth) >= recheckAuthPeriod {
			if err = trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
				experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return nil, err
			}
			trialLogsTimeSinceLastAuth = time.Now()
		}

		switch {
		case r.Follow, r.Limit > trialLogsBatchSize:
			r.Limit = trialLogsBatchSize
		case r.Limit <= 0:
			return nil, nil
		}

		b, state, fErr := a.m.trialLogBackend.TrialLogs(
			int(req.TrialId), r.Limit, filters, req.OrderBy, followState)
		if fErr != nil {
			return nil, fErr
		}
		followState = state

		return model.TrialLogBatch(b), nil
	}

	total, err := a.m.trialLogBackend.TrialLogsCount(int(req.TrialId), filters)
	if err != nil {
		res <- api.ErrBatchResult(fmt.Errorf("failed to get trial count from backend: %w", err))
		return
	}
	effectiveLimit := api.EffectiveLimit(int(req.Limit), 0, total)

	api.NewBatchStreamProcessor(
		api.BatchRequest{Limit: effectiveLimit, Follow: req.Follow},
		fetch,
		a.isTrialTerminalFunc(int(req.TrialId), a.m.taskLogBackend.MaxTerminationDelay()),
		false,
		nil,
		&trialLogsBatchMissWaitTime,
	).Run(ctx, res)
}

func constructTrialLogsFilters(req *apiv1.TrialLogsRequest) ([]api.Filter, error) {
	var filters []api.Filter

	addInFilter := func(field string, values interface{}, count int) {
		if values != nil && count > 0 {
			filters = append(filters, api.Filter{
				Field:     field,
				Operation: api.FilterOperationIn,
				Values:    values,
			})
		}
	}

	addInFilter("agent_id", req.AgentIds, len(req.AgentIds))
	addInFilter("container_id", req.ContainerIds, len(req.ContainerIds))
	addInFilter("rank_id", req.RankIds, len(req.RankIds))
	addInFilter("stdtype", req.Stdtypes, len(req.Stdtypes))
	addInFilter("source", req.Sources, len(req.Sources))
	addInFilter("level", func() interface{} {
		var levels []string
		for _, l := range req.Levels {
			levels = append(levels, model.TaskLogLevelFromProto(l))
		}
		return levels
	}(), len(req.Levels))

	if req.TimestampBefore != nil {
		if err := req.TimestampBefore.CheckValid(); err != nil {
			return nil, err
		}
		filters = append(filters, api.Filter{
			Field:     "timestamp",
			Operation: api.FilterOperationLessThanEqual,
			Values:    req.TimestampBefore.AsTime(),
		})
	}

	if req.TimestampAfter != nil {
		if err := req.TimestampAfter.CheckValid(); err != nil {
			return nil, err
		}
		filters = append(filters, api.Filter{
			Field:     "timestamp",
			Operation: api.FilterOperationGreaterThan,
			Values:    req.TimestampAfter.AsTime(),
		})
	}
	return filters, nil
}

func (a *apiServer) TrialLogsFields(
	req *apiv1.TrialLogsFieldsRequest, resp apiv1.Determined_TrialLogsFieldsServer,
) error {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(resp.Context(), int(req.TrialId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return err
	}

	trialTaskIDs, err := db.TrialTaskIDsByTrialID(resp.Context(), int(req.TrialId))
	if err != nil {
		return fmt.Errorf("retreiving task IDs for trial log fields: %w", err)
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	// Stream fields from trial logs table, just to support pre-task-logs trials with old logs.
	trialLogsTimeSinceLastAuth := time.Now() // time.Now() to avoid recheck from above.
	resOld := make(chan api.BatchResult)
	go api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: req.Follow},
		func(lr api.BatchRequest) (api.Batch, error) {
			if time.Since(trialLogsTimeSinceLastAuth) >= recheckAuthPeriod {
				if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(resp.Context(),
					int(req.TrialId),
					experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
					return nil, err
				}
				trialLogsTimeSinceLastAuth = time.Now()
			}

			fields, err := a.m.trialLogBackend.TrialLogsFields(int(req.TrialId))
			return api.ToBatchOfOne(fields), err
		},
		a.isTrialTerminalFunc(int(req.TrialId), a.m.taskLogBackend.MaxTerminationDelay()),
		true,
		&distinctFieldBatchWaitTime,
		&distinctFieldBatchWaitTime,
	).Run(ctx, resOld)

	for i, trialTaskID := range trialTaskIDs {
		// Also stream fields from task logs table, for ordinary logs (as they are written now).
		taskLogsTimeSinceLastAuth := time.Now() // time.Now() to avoid recheck from above.
		resNew := make(chan api.BatchResult)
		go api.NewBatchStreamProcessor(
			api.BatchRequest{Follow: req.Follow && i == len(trialTaskIDs)-1},
			func(lr api.BatchRequest) (api.Batch, error) {
				if time.Since(taskLogsTimeSinceLastAuth) >= recheckAuthPeriod {
					if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(resp.Context(),
						int(req.TrialId),
						experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
						return nil, err
					}
					taskLogsTimeSinceLastAuth = time.Now()
				}

				fields, err := a.m.taskLogBackend.TaskLogsFields(trialTaskID.TaskID)
				return api.ToBatchOfOne(&apiv1.TrialLogsFieldsResponse{
					AgentIds:     fields.AgentIds,
					ContainerIds: fields.ContainerIds,
					RankIds:      fields.RankIds,
					Stdtypes:     fields.Stdtypes,
					Sources:      fields.Sources,
				}), err
			},
			a.isTaskTerminalFunc(trialTaskID.TaskID, a.m.taskLogBackend.MaxTerminationDelay()),
			true,
			&distinctFieldBatchWaitTime,
			&distinctFieldBatchWaitTime,
		).Run(ctx, resNew)

		// First iterate merge with legacy trial logs fields.
		if i == 0 {
			if err := zipBatches(resOld, resNew, func(b1, b2 api.Batch) error {
				r1 := b1.(api.BatchOfOne).Inner.(*apiv1.TrialLogsFieldsResponse)
				r2 := b2.(api.BatchOfOne).Inner.(*apiv1.TrialLogsFieldsResponse)
				return resp.Send(&apiv1.TrialLogsFieldsResponse{
					AgentIds:     setString(append(r1.AgentIds, r2.AgentIds...)...),
					ContainerIds: setString(append(r1.ContainerIds, r2.ContainerIds...)...),
					RankIds:      setInt32(append(r1.RankIds, r2.RankIds...)...),
					Stdtypes:     setString(append(r1.Stdtypes, r2.Stdtypes...)...),
					Sources:      setString(append(r1.Sources, r2.Sources...)...),
				})
			}); err != nil {
				return fmt.Errorf("merging and sending trial log field batches for task id %s: %w",
					trialTaskID.TaskID, err)
			}
		} else {
			// Any past the first is always version 2 and above.
			err := processBatches(resNew, func(b api.Batch) error {
				return b.ForEach(func(i interface{}) error {
					return resp.Send(i.(*apiv1.TrialLogsFieldsResponse))
				})
			})
			if err != nil {
				return fmt.Errorf("sending trial log field batches for task id %s: %w",
					trialTaskID.TaskID, err)
			}
		}
	}

	return nil
}

func (a *apiServer) GetTrialCheckpoints(
	ctx context.Context, req *apiv1.GetTrialCheckpointsRequest,
) (*apiv1.GetTrialCheckpointsResponse, error) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.Id),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	resp := &apiv1.GetTrialCheckpointsResponse{}
	resp.Checkpoints = []*checkpointv1.Checkpoint{}

	switch err := a.m.db.QueryProto("get_checkpoints_for_trial", &resp.Checkpoints, req.Id); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(
			codes.NotFound, "no checkpoints found for trial %d", req.Id)
	case err != nil:
		return nil,
			errors.Wrapf(err, "error fetching checkpoints for trial %d from database", req.Id)
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
		if req.OrderBy == apiv1.OrderBy_ORDER_BY_DESC {
			aj, ai = ai, aj
		}

		switch req.GetSortBy().(type) {
		case *apiv1.GetTrialCheckpointsRequest_SortByAttr:
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
		case *apiv1.GetTrialCheckpointsRequest_SortByMetric:
			return protoless.CheckpointMetricNameLess(ai, aj, req.GetSortByMetric())
		default:
			return protoless.CheckpointTrialIDLess(ai, aj)
		}
	})

	return resp, api.Paginate(&resp.Pagination, &resp.Checkpoints, req.Offset, req.Limit)
}

func (a *apiServer) KillTrial(
	ctx context.Context, req *apiv1.KillTrialRequest,
) (*apiv1.KillTrialResponse, error) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.Id),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.Id))
	if err != nil {
		return nil, err
	}

	s := experiment.PatchTrialState{
		RequestID: rID,
		State: model.StateWithReason{
			State:               model.StoppingKilledState,
			InformationalReason: "user requested kill",
		},
	}

	e, ok := experiment.ExperimentRegistry.Load(eID)
	if !ok {
		return nil, api.NotFoundErrs("experiment", fmt.Sprint(eID), true)
	}
	if err = e.PatchTrialState(s); err != nil {
		log.WithError(err).Error("error killing trial")
		return nil, err
	}
	return &apiv1.KillTrialResponse{}, nil
}

func (a *apiServer) GetExperimentTrials(
	ctx context.Context, req *apiv1.GetExperimentTrialsRequest,
) (resp *apiv1.GetExperimentTrialsResponse, err error) {
	if _, _, err = a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	// Construct the trial filtering expression.
	var allStates []string
	for _, state := range req.States {
		allStates = append(allStates, strings.TrimPrefix(state.String(), "STATE_"))
	}
	stateFilterExpr := strings.Join(allStates, ",")

	// Construct the ordering expression.
	orderColMap := map[apiv1.GetExperimentTrialsRequest_SortBy]string{
		apiv1.GetExperimentTrialsRequest_SORT_BY_UNSPECIFIED:              "id",
		apiv1.GetExperimentTrialsRequest_SORT_BY_ID:                       "id",
		apiv1.GetExperimentTrialsRequest_SORT_BY_START_TIME:               "start_time",
		apiv1.GetExperimentTrialsRequest_SORT_BY_END_TIME:                 "end_time",
		apiv1.GetExperimentTrialsRequest_SORT_BY_STATE:                    "state",
		apiv1.GetExperimentTrialsRequest_SORT_BY_BEST_VALIDATION_METRIC:   "best_signed_search_metric",
		apiv1.GetExperimentTrialsRequest_SORT_BY_LATEST_VALIDATION_METRIC: "latest_signed_search_metric",
		apiv1.GetExperimentTrialsRequest_SORT_BY_BATCHES_PROCESSED:        "total_batches_processed",
		apiv1.GetExperimentTrialsRequest_SORT_BY_DURATION:                 "duration",
		apiv1.GetExperimentTrialsRequest_SORT_BY_RESTARTS:                 "restarts",
		apiv1.GetExperimentTrialsRequest_SORT_BY_CHECKPOINT_SIZE:          "checkpoint_size",
		apiv1.GetExperimentTrialsRequest_SORT_BY_LOG_RETENTION_DAYS:       "log_retention_days",
	}
	sortByMap := map[apiv1.OrderBy]string{
		apiv1.OrderBy_ORDER_BY_UNSPECIFIED: "ASC",
		apiv1.OrderBy_ORDER_BY_ASC:         "ASC",
		apiv1.OrderBy_ORDER_BY_DESC:        "DESC",
	}
	orderExpr := ""
	switch _, ok := orderColMap[req.SortBy]; {
	case !ok:
		return nil, fmt.Errorf("unsupported sort by %s", req.SortBy)
	case orderColMap[req.SortBy] != "id":
		orderExpr = fmt.Sprintf(
			"%s %s, id %s",
			orderColMap[req.SortBy], sortByMap[req.OrderBy], sortByMap[req.OrderBy],
		)
	default:
		orderExpr = fmt.Sprintf("id %s", sortByMap[req.OrderBy])
	}

	resp = &apiv1.GetExperimentTrialsResponse{}
	if err = a.m.db.QueryProtof(
		"proto_get_trial_ids_for_experiment",
		[]interface{}{orderExpr},
		resp,
		req.ExperimentId,
		stateFilterExpr,
		req.Offset,
		req.Limit,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to get trial ids for experiment %d", req.ExperimentId)
	} else if len(resp.Trials) == 0 {
		return resp, nil
	}

	// Of the form "($1, $2), ($3, $4), ... ($N, $N+1)". Used in a VALUES expression in Postgres.
	valuesExpr := make([]string, 0, len(resp.Trials))
	trialIDsWithOrdering := make([]any, 0, len(resp.Trials))
	trialIDs := make([]int32, 0, len(resp.Trials))
	for i, trial := range resp.Trials {
		valuesExpr = append(valuesExpr, fmt.Sprintf("($%d::int, $%d::int)", i*2+1, i*2+2))
		trialIDsWithOrdering = append(trialIDsWithOrdering, trial.Id, i)
		trialIDs = append(trialIDs, trial.Id)
	}

	switch err = a.m.db.QueryProtof(
		"proto_get_trials_plus",
		[]any{strings.Join(valuesExpr, ", ")},
		&resp.Trials,
		trialIDsWithOrdering...,
	); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "trials %v not found:", trialIDs)
	case err != nil:
		return nil, errors.Wrapf(err, "failed to get trials detail for experiment %d", req.ExperimentId)
	}

	if err = a.enrichTrialState(resp.Trials...); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *apiServer) GetTrialRemainingLogRetentionDays(
	ctx context.Context, req *apiv1.GetTrialRemainingLogRetentionDaysRequest,
) (resp *apiv1.GetTrialRemainingLogRetentionDaysResponse, err error) {
	t, err := db.TrialByID(ctx, int(req.Id))
	if err != nil {
		return nil, fmt.Errorf("getting trial %v: %w", req.Id, err)
	}

	q := `
SELECT   
CASE
	WHEN MIN(t.end_time) <= ( retention_timestamp() - make_interval(days => ?) ) THEN 0
	ELSE extract(day from MIN(end_time) + make_interval(days => ?) - NOW())::int
END remaining_log_retention_days
FROM tasks t
JOIN run_id_task_id as r ON t.task_id = r.task_id
WHERE r.run_id = ?
`
	resp = &apiv1.GetTrialRemainingLogRetentionDaysResponse{}
	if t.LogRetentionDays == nil || *t.LogRetentionDays == -1 {
		days := int32(-1)
		resp.RemainingDays = &days
	} else {
		var days *int32
		err = db.Bun().NewRaw(q, t.LogRetentionDays, t.LogRetentionDays, t.ID).Scan(ctx, &days)
		if err != nil {
			return nil, fmt.Errorf("getting remaining log days for trial %v: %w", t.ID, err)
		}
		resp.RemainingDays = days
	}

	return resp, nil
}

func (a *apiServer) GetTrial(ctx context.Context, req *apiv1.GetTrialRequest) (
	*apiv1.GetTrialResponse, error,
) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	resp := &apiv1.GetTrialResponse{Trial: &trialv1.Trial{}}
	if err := a.m.db.QueryProtof(
		"proto_get_trials_plus",
		[]any{"($1::int, $2::int)"},
		resp.Trial,
		req.TrialId,
		1,
	); err != nil {
		return nil, errors.Wrapf(err, "failed to get trial %d", req.TrialId)
	}

	if resp.Trial.State == trialv1.State_STATE_ACTIVE {
		if err := a.enrichTrialState(resp.Trial); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (a *apiServer) GetTrialByExternalID(ctx context.Context, req *apiv1.GetTrialByExternalIDRequest) (
	*apiv1.GetTrialByExternalIDResponse, error,
) {
	var trialID int
	err := db.Bun().NewRaw(`
SELECT t.id
FROM trials t JOIN experiments e
ON t.experiment_id = e.id
WHERE t.external_trial_id = ? AND e.external_experiment_id = ?`,
		req.ExternalTrialId, req.ExternalExperimentId).Scan(ctx, &trialID)
	if err != nil {
		return nil, db.MatchSentinelError(err)
	}

	proxyReq := apiv1.GetTrialRequest{TrialId: int32(trialID)}
	proxyResp, err := a.GetTrial(ctx, &proxyReq)
	if err != nil {
		return nil, err
	}

	resp := apiv1.GetTrialByExternalIDResponse{
		Trial: proxyResp.Trial,
	}

	return &resp, nil
}

func (a *apiServer) PutTrialRetainLogs(
	ctx context.Context, req *apiv1.PutTrialRetainLogsRequest,
) (*apiv1.PutTrialRetainLogsResponse, error) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(
		ctx, int(req.TrialId), experiment.AuthZProvider.Get().CanEditExperiment,
	); err != nil {
		return nil, err
	}

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Table("runs").
			Set("log_retention_days = ?", req.NumDays).
			Where("id = ?", req.TrialId).
			Exec(ctx); err != nil {
			return fmt.Errorf("updating log retention days for trial: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &apiv1.PutTrialRetainLogsResponse{}, nil
}

func (a *apiServer) formatMetrics(
	m *apiv1.DownsampledMetrics, metricMeasurements []db.MetricMeasurements,
) error {
	for _, in := range metricMeasurements {
		valueMap, err := structpb.NewStruct(in.Values) // in.Value is a map.
		if err != nil {
			return errors.Wrapf(err, "error formatting metrics")
		}
		out := apiv1.DataPoint{
			Time:    timestamppb.New(in.Time),
			Batches: int32(in.Batches),
			Values:  valueMap,
			Epoch:   in.Epoch,
		}
		m.Data = append(m.Data, &out)
	}
	return nil
}

func (a *apiServer) parseMetricGroupArgs(
	legacyType apiv1.MetricType, newType model.MetricGroup,
) (model.MetricGroup, error) {
	if legacyType != apiv1.MetricType_METRIC_TYPE_UNSPECIFIED && newType != "" {
		return "", status.Errorf(codes.InvalidArgument, "cannot specify both legacy and new metric group")
	}
	if newType != "" {
		return newType, nil
	}
	conv := &protoconverter.ProtoConverter{}
	convertedLegacyType := conv.ToMetricGroup(legacyType)
	if cErr := conv.Error(); cErr != nil {
		return "", status.Errorf(codes.InvalidArgument, "converting metric group: %s", cErr)
	}
	return convertedLegacyType, nil
}

func (a *apiServer) multiTrialSample(trialID int32, metricNames []string,
	metricGroup model.MetricGroup, maxDatapoints int, startBatches int,
	endBatches int, timeSeriesFilter *commonv1.PolymorphicFilter,
	metricIds []string,
) ([]*apiv1.DownsampledMetrics, error) {
	var startTime time.Time
	var metrics []*apiv1.DownsampledMetrics
	// For now "epoch" is the only custom xAxis metric label supported so we
	// build the `MetricSeriesEpoch` array. In the future this logic should
	// be updated to support any number of xAxis metric options
	xAxisLabelMetrics := []string{"epoch"}

	if err := db.ValidatePolymorphicFilter(timeSeriesFilter); err != nil {
		return nil, err
	}

	if len(metricNames) > 0 && len(metricIds) > 0 {
		return nil, fmt.Errorf(`error fetching time series of metrics cannot specify
		both metric ids and metric names`)
	}

	if endBatches == 0 {
		endBatches = math.MaxInt32
	}
	if maxDatapoints == 0 {
		maxDatapoints = 200
	}

	var timeSeriesColumn *string

	// If no time series filter column name is supplied then default to batches.
	defaultTimeSeriesColumn := batches
	if timeSeriesFilter == nil || timeSeriesFilter.Name == nil {
		timeSeriesColumn = &defaultTimeSeriesColumn
	} else {
		timeSeriesColumn = timeSeriesFilter.Name
	}

	metricGroupToNames := make(map[model.MetricGroup][]string)
	if len(metricNames) > 0 {
		if metricGroup == "" {
			// to keep backwards compatibility.
			metricGroupToNames[model.TrainingMetricGroup] = metricNames
			metricGroupToNames[model.ValidationMetricGroup] = metricNames
		} else {
			metricGroupToNames[metricGroup] = metricNames
		}
	}
	for _, metricIDStr := range metricIds {
		metricID, err := model.DeserializeMetricIdentifier(metricIDStr)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing metric id %s", metricIDStr)
		}
		metricGroupToNames[metricID.Group] = append(metricGroupToNames[metricID.Group],
			string(metricID.Name))
	}

	getDownSampledMetric := func(aMetricNames []string, aMetricGroup model.MetricGroup,
	) (*apiv1.DownsampledMetrics, error) {
		var metric apiv1.DownsampledMetrics
		metricMeasurements, err := trials.MetricsTimeSeries(
			trialID, startTime, aMetricNames, startBatches, endBatches,
			xAxisLabelMetrics,
			maxDatapoints, *timeSeriesColumn, timeSeriesFilter, aMetricGroup)
		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("error fetching time series of %s metrics",
				aMetricGroup))
		}
		//nolint:staticcheck // SA1019: backward compatibility
		metric.Type = aMetricGroup.ToProto()
		metric.Group = aMetricGroup.ToString()
		if len(metricMeasurements) > 0 {
			if err = a.formatMetrics(&metric, metricMeasurements); err != nil {
				return nil, err
			}
			return &metric, nil
		}
		return nil, nil
	}

	metricGroups := make([]model.MetricGroup, 0, len(metricGroupToNames))
	for metricGroup := range metricGroupToNames {
		metricGroups = append(metricGroups, metricGroup)
	}
	sort.Slice(metricGroups, func(i, j int) bool {
		return metricGroups[i] < metricGroups[j]
	})

	for _, mGroup := range metricGroups {
		metricNames := metricGroupToNames[mGroup]
		metric, err := getDownSampledMetric(metricNames, mGroup)
		if err != nil {
			return nil, err
		}
		if metric != nil {
			metrics = append(metrics, metric)
		}
	}

	return metrics, nil
}

func (a *apiServer) CompareTrials(ctx context.Context,
	req *apiv1.CompareTrialsRequest,
) (*apiv1.CompareTrialsResponse, error) {
	trialsList := make([]*apiv1.ComparableTrial, 0, len(req.TrialIds))
	for _, trialID := range req.TrialIds {
		if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(trialID),
			experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
			return nil, err
		}

		container := &apiv1.ComparableTrial{Trial: &trialv1.Trial{}}
		switch err := a.m.db.QueryProto("get_trial_basic", container.Trial, trialID); {
		case err == db.ErrNotFound:
			return nil, status.Errorf(codes.NotFound, "trial %d not found:", trialID)
		case err != nil:
			return nil, errors.Wrapf(err, "failed to get trial %d", trialID)
		}

		//nolint:staticcheck // SA1019: backward compatibility
		metricGroup, err := a.parseMetricGroupArgs(req.MetricType, model.MetricGroup(req.Group))
		if err != nil {
			return nil, err
		}

		tsample, err := a.multiTrialSample(trialID, req.MetricNames, metricGroup,
			int(req.MaxDatapoints), int(req.StartBatches), int(req.EndBatches),
			req.TimeSeriesFilter, req.MetricIds)
		if err != nil {
			return nil, errors.Wrapf(err, "failed sampling")
		}
		container.Metrics = tsample
		trialsList = append(trialsList, container)
	}
	return &apiv1.CompareTrialsResponse{Trials: trialsList}, nil
}

func (a *apiServer) GetMetrics(
	req *apiv1.GetMetricsRequest, resp apiv1.Determined_GetMetricsServer,
) error {
	sendFunc := func(m []*trialv1.MetricsReport) error {
		return resp.Send(&apiv1.GetMetricsResponse{Metrics: m})
	}
	if err := a.streamMetrics(resp.Context(), req.TrialIds, sendFunc,
		model.MetricGroup(req.Group)); err != nil {
		return err
	}

	return nil
}

func (a *apiServer) GetTrainingMetrics(
	req *apiv1.GetTrainingMetricsRequest, resp apiv1.Determined_GetTrainingMetricsServer,
) error {
	sendFunc := func(m []*trialv1.MetricsReport) error {
		return resp.Send(&apiv1.GetTrainingMetricsResponse{Metrics: m})
	}
	if err := a.streamMetrics(resp.Context(), req.TrialIds, sendFunc,
		model.TrainingMetricGroup); err != nil {
		return err
	}

	return nil
}

func (a *apiServer) GetValidationMetrics(
	req *apiv1.GetValidationMetricsRequest, resp apiv1.Determined_GetValidationMetricsServer,
) error {
	sendFunc := func(m []*trialv1.MetricsReport) error {
		return resp.Send(&apiv1.GetValidationMetricsResponse{Metrics: m})
	}
	if err := a.streamMetrics(resp.Context(), req.TrialIds, sendFunc,
		model.ValidationMetricGroup); err != nil {
		return err
	}

	return nil
}

func (a *apiServer) streamMetrics(ctx context.Context,
	trialIDs []int32, sendFunc func(m []*trialv1.MetricsReport) error, metricGroup model.MetricGroup,
) error {
	if len(trialIDs) == 0 {
		return status.Error(codes.InvalidArgument, "must specify at least one trialId")
	}
	ids := make(map[int32]bool)
	for _, id := range trialIDs {
		if ids[id] {
			return status.Errorf(codes.InvalidArgument, "duplicate id=%d specified", id)
		}
	}
	slices.Sort(trialIDs)

	for _, trialID := range trialIDs {
		if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(trialID),
			experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
			return err
		}
	}

	const size = 1000

	trialIDIndex := 0
	key := -1
	mGroup := metricGroup.ToString()
	for {
		res, err := db.GetMetrics(ctx, int(trialIDs[trialIDIndex]), key, size, &mGroup)
		if err != nil {
			return err
		}
		if len(res) > 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := sendFunc(res); err != nil {
				return err
			}
			key = int(res[len(res)-1].TotalBatches)
		}

		if len(res) != size {
			trialIDIndex++
			if trialIDIndex >= len(trialIDs) {
				break
			}

			key = -1
		}
	}

	return nil
}

func (a *apiServer) GetTrialWorkloads(ctx context.Context, req *apiv1.GetTrialWorkloadsRequest) (
	*apiv1.GetTrialWorkloadsResponse, error,
) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	resp := &apiv1.GetTrialWorkloadsResponse{}
	limit := &req.Limit
	if *limit == 0 {
		limit = ptrs.Ptr[int32](-1)
	}

	sortCode := "total_batches"
	if req.SortKey != "" && req.SortKey != batches {
		sortCode = fmt.Sprintf("sort_metrics->'avg_metrics'->>'%s'",
			strings.ReplaceAll(req.SortKey, "'", ""))
	}

	switch err := a.m.db.QueryProtof(
		"proto_get_trial_workloads",
		[]interface{}{
			sortCode,
			db.OrderByToSQL(req.OrderBy),
			db.OrderByToSQL(req.OrderBy),
			db.OrderByToSQL(req.OrderBy),
		},
		resp,
		req.TrialId,
		req.Offset,
		limit,
		req.Filter.String(),
		req.IncludeBatchMetrics,
		//nolint:staticcheck // SA1019: backward compatibility
		req.MetricType.String(),
		req.RemoveDeletedCheckpoints,
	); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "trial %d workloads not found:", req.TrialId)
	case err != nil:
		return nil, errors.Wrapf(err, "failed to get trial %d workloads", req.TrialId)
	}

	return resp, nil
}

func (a *apiServer) GetTrialProfilerMetrics(
	req *apiv1.GetTrialProfilerMetricsRequest,
	resp apiv1.Determined_GetTrialProfilerMetricsServer,
) error {
	var timeSinceLastAuth time.Time
	fetch := func(lr api.BatchRequest) (api.Batch, error) {
		if time.Since(timeSinceLastAuth) >= recheckAuthPeriod {
			if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(resp.Context(),
				int(req.Labels.TrialId),
				experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return nil, err
			}
			timeSinceLastAuth = time.Now()
		}

		switch {
		case lr.Follow, lr.Limit > trialProfilerMetricsBatchSize:
			lr.Limit = trialProfilerMetricsBatchSize
		case lr.Limit <= 0:
			return nil, nil
		}
		return a.m.db.GetTrialProfilerMetricsBatches(req.Labels, lr.Offset, lr.Limit)
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult)
	go api.NewBatchStreamProcessor(
		api.BatchRequest{Limit: math.MaxInt32, Follow: req.Follow},
		fetch,
		a.isTrialTerminalFunc(int(req.Labels.TrialId), -1),
		false,
		nil,
		&trialProfilerMetricsBatchMissWaitTime,
	).Run(ctx, res)

	return processBatches(res, func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(&apiv1.GetTrialProfilerMetricsResponse{
				Batch: r.(*trialv1.TrialProfilerMetricsBatch),
			})
		})
	})
}

func (a *apiServer) GetTrialProfilerAvailableSeries(
	req *apiv1.GetTrialProfilerAvailableSeriesRequest,
	resp apiv1.Determined_GetTrialProfilerAvailableSeriesServer,
) error {
	var timeSinceLastAuth time.Time
	fetch := func(_ api.BatchRequest) (api.Batch, error) {
		if time.Since(timeSinceLastAuth) >= recheckAuthPeriod {
			if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(resp.Context(),
				int(req.TrialId),
				experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return nil, err
			}
			timeSinceLastAuth = time.Now()
		}

		var labels apiv1.GetTrialProfilerAvailableSeriesResponse
		res, err := db.GetTrialProfilerAvailableSeries(resp.Context(), req.TrialId)
		if err != nil {
			return nil, err
		}
		labels.Labels = res
		return api.ToBatchOfOne(&labels), nil
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult)
	go api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: req.Follow},
		fetch,
		a.isTrialTerminalFunc(int(req.TrialId), -1),
		true,
		&TrialAvailableSeriesBatchWaitTime,
		&TrialAvailableSeriesBatchWaitTime,
	).Run(ctx, res)

	return processBatches(res, func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(r.(*apiv1.GetTrialProfilerAvailableSeriesResponse))
		})
	})
}

func (a *apiServer) PostTrialProfilerMetricsBatch(
	ctx context.Context,
	req *apiv1.PostTrialProfilerMetricsBatchRequest,
) (*apiv1.PostTrialProfilerMetricsBatchResponse, error) {
	var errs *multierror.Error
	existingTrials := map[int]bool{}
	for _, batch := range req.Batches {
		trialID := int(batch.Labels.TrialId)
		if !existingTrials[trialID] {
			if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, trialID,
				experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
				return nil, err
			}
		}

		if len(batch.Values) != len(batch.Batches) ||
			len(batch.Batches) != len(batch.Timestamps) {
			errs = multierror.Append(errs, status.Errorf(codes.InvalidArgument,
				"values, batches and timestamps should be equal sized arrays"))
			continue
		}

		labels, err := protojson.Marshal(batch.Labels)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("failed to marshal labels: %w", err))
			continue
		}

		timestamps, err := protoutils.TimeSliceFromProto(batch.Timestamps)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("failed to convert proto timestamps: %w", err))
			continue
		}

		if err := a.m.db.InsertTrialProfilerMetricsBatch(
			batch.Values, batch.Batches, timestamps, labels,
		); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("failed to insert batch: %w", err))
			continue
		}
	}
	return &apiv1.PostTrialProfilerMetricsBatchResponse{}, errs.ErrorOrNil()
}

func (a *apiServer) AllocationPreemptionSignal(
	ctx context.Context,
	req *apiv1.AllocationPreemptionSignalRequest,
) (*apiv1.AllocationPreemptionSignalResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, fmt.Errorf("checking if allocation is editable: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	preempt, err := task.DefaultService.WatchPreemption(ctx, model.AllocationID(req.AllocationId))
	if err != nil {
		return nil, fmt.Errorf("watching preemption status: %w", err)
	}
	return &apiv1.AllocationPreemptionSignalResponse{Preempt: preempt}, nil
}

func (a *apiServer) AckAllocationPreemptionSignal(
	ctx context.Context, req *apiv1.AckAllocationPreemptionSignalRequest,
) (*apiv1.AckAllocationPreemptionSignalResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	err := task.DefaultService.AckPreemption(ctx, model.AllocationID(req.AllocationId))
	if err != nil {
		return nil, err
	}
	return &apiv1.AckAllocationPreemptionSignalResponse{}, nil
}

func (a *apiServer) AllocationPendingPreemptionSignal(
	ctx context.Context,
	req *apiv1.AllocationPendingPreemptionSignalRequest,
) (*apiv1.AllocationPendingPreemptionSignalResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	if err := a.m.rm.ExternalPreemptionPending(
		sproto.PendingPreemption{AllocationID: model.AllocationID(req.AllocationId)},
	); err != nil {
		return nil, err
	}

	return &apiv1.AllocationPendingPreemptionSignalResponse{}, nil
}

func (a *apiServer) NotifyContainerRunning(
	ctx context.Context,
	req *apiv1.NotifyContainerRunningRequest,
) (*apiv1.NotifyContainerRunningResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	if err := a.m.rm.NotifyContainerRunning(
		sproto.NotifyContainerRunning{
			AllocationID: model.AllocationID(req.AllocationId),
			NumPeers:     req.NumPeers,
			Rank:         req.Rank,
			NodeName:     req.NodeName,
		},
	); err != nil {
		return nil, err
	}

	return &apiv1.NotifyContainerRunningResponse{}, nil
}

func (a *apiServer) MarkAllocationResourcesDaemon(
	ctx context.Context, req *apiv1.MarkAllocationResourcesDaemonRequest,
) (*apiv1.MarkAllocationResourcesDaemonResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	err := task.DefaultService.SetResourcesAsDaemon(
		ctx,
		model.AllocationID(req.AllocationId),
		sproto.ResourcesID(req.ResourcesId),
	)
	if err != nil {
		return nil, err
	}
	return &apiv1.MarkAllocationResourcesDaemonResponse{}, nil
}

func (a *apiServer) GetCurrentTrialSearcherOperation(
	ctx context.Context, req *apiv1.GetCurrentTrialSearcherOperationRequest,
) (*apiv1.GetCurrentTrialSearcherOperationResponse, error) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		experiment.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	if err != nil {
		return nil, err
	}

	e, ok := experiment.ExperimentRegistry.Load(eID)
	if !ok {
		return nil, api.NotFoundErrs("experiment", fmt.Sprint(eID), true)
	}
	resp, err := e.TrialGetSearcherState(rID)
	if err != nil {
		return nil, err
	}

	return &apiv1.GetCurrentTrialSearcherOperationResponse{
		Op: &experimentv1.TrialOperation{
			Union: &experimentv1.TrialOperation_ValidateAfter{
				ValidateAfter: resp.Op.ToProto(),
			},
		},
		Completed: resp.Complete,
	}, nil
}

func (a *apiServer) CompleteTrialSearcherValidation(
	ctx context.Context, req *apiv1.CompleteTrialSearcherValidationRequest,
) (*apiv1.CompleteTrialSearcherValidationResponse, error) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	if err != nil {
		return nil, err
	}

	e, ok := experiment.ExperimentRegistry.Load(eID)
	if !ok {
		return nil, api.NotFoundErrs("experiment", fmt.Sprint(eID), true)
	}

	msg := experiment.TrialCompleteOperation{
		RequestID: rID,
		Metric:    req.CompletedOperation.SearcherMetric.AsInterface(),
		Op:        searcher.NewValidateAfter(rID, req.CompletedOperation.Op.Length),
	}
	if err := e.TrialCompleteOperation(msg); err != nil {
		return nil, err
	}
	return &apiv1.CompleteTrialSearcherValidationResponse{}, nil
}

func (a *apiServer) ReportTrialSearcherEarlyExit(
	ctx context.Context, req *apiv1.ReportTrialSearcherEarlyExitRequest,
) (*apiv1.ReportTrialSearcherEarlyExitResponse, error) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	if err != nil {
		return nil, err
	}

	e, ok := experiment.ExperimentRegistry.Load(eID)
	if !ok {
		return nil, api.NotFoundErrs("experiment", fmt.Sprint(eID), true)
	}

	msg := experiment.UserInitiatedEarlyTrialExit{
		RequestID: rID,
		Reason:    model.ExitedReasonFromProto(req.EarlyExit.Reason),
	}
	if err := e.UserInitiatedEarlyTrialExit(msg); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialSearcherEarlyExitResponse{}, nil
}

func (a *apiServer) ReportTrialProgress(
	ctx context.Context, req *apiv1.ReportTrialProgressRequest,
) (*apiv1.ReportTrialProgressResponse, error) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	if err != nil {
		return nil, err
	}

	e, ok := experiment.ExperimentRegistry.Load(eID)
	if !ok {
		return nil, api.NotFoundErrs("experiment", fmt.Sprint(eID), true)
	}

	msg := experiment.TrialReportProgress{
		RequestID: rID,
		Progress:  searcher.PartialUnits(req.Progress),
	}
	if err := e.TrialReportProgress(msg); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialProgressResponse{}, nil
}

func (a *apiServer) ReportTrialMetrics(
	ctx context.Context, req *apiv1.ReportTrialMetricsRequest,
) (*apiv1.ReportTrialMetricsResponse, error) {
	metricGroup := model.MetricGroup(req.Group)
	if err := metricGroup.Validate(); err != nil {
		return nil, err
	}
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.Metrics.TrialId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	if err := a.m.db.AddTrialMetrics(ctx, req.Metrics, metricGroup); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialMetricsResponse{}, nil
}

func (a *apiServer) ReportTrialTrainingMetrics(
	ctx context.Context, req *apiv1.ReportTrialTrainingMetricsRequest,
) (*apiv1.ReportTrialTrainingMetricsResponse, error) {
	_, err := a.ReportTrialMetrics(ctx, &apiv1.ReportTrialMetricsRequest{
		Metrics: req.TrainingMetrics,
		Group:   model.TrainingMetricGroup.ToString(),
	})
	return &apiv1.ReportTrialTrainingMetricsResponse{}, err
}

func (a *apiServer) ReportTrialValidationMetrics(
	ctx context.Context, req *apiv1.ReportTrialValidationMetricsRequest,
) (*apiv1.ReportTrialValidationMetricsResponse, error) {
	_, err := a.ReportTrialMetrics(ctx, &apiv1.ReportTrialMetricsRequest{
		Metrics: req.ValidationMetrics,
		Group:   model.ValidationMetricGroup.ToString(),
	})
	return &apiv1.ReportTrialValidationMetricsResponse{}, err
}

func (a *apiServer) ReportCheckpoint(
	ctx context.Context, req *apiv1.ReportCheckpointRequest,
) (*apiv1.ReportCheckpointResponse, error) {
	if err := a.canDoActionsOnTask(ctx, model.TaskID(req.Checkpoint.TaskId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	c, err := checkpointV2FromProtoWithDefaults(req.Checkpoint)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument, "unconvertable checkpoint: %s", err.Error())
	}

	switch c.State {
	case model.CompletedState:
	case "":
		c.State = model.CompletedState
	default:
	}

	task, err := db.TaskByID(ctx, model.TaskID(req.Checkpoint.TaskId))
	if err != nil {
		return nil, fmt.Errorf("looking up task to decide if trial: %w", err)
	}
	if task.TaskType != model.TaskTypeTrial {
		return nil, fmt.Errorf("can only report checkpoints on trial's tasks")
	}
	trial, err := db.TrialByTaskID(ctx, task.TaskID)
	if err != nil {
		return nil, fmt.Errorf("getting trial by task ID: %w", err)
	}

	if err := db.AddCheckpointMetadata(ctx, c, trial.ID); err != nil {
		return nil, err
	}

	return &apiv1.ReportCheckpointResponse{}, nil
}

func checkpointV2FromProtoWithDefaults(p *checkpointv1.Checkpoint) (*model.CheckpointV2, error) {
	conv := &protoconverter.ProtoConverter{}

	switch p.State {
	case checkpointv1.State_STATE_COMPLETED:
	case checkpointv1.State_STATE_UNSPECIFIED:
		p.State = checkpointv1.State_STATE_COMPLETED
	default:
		return nil, status.Errorf(codes.InvalidArgument,
			"invalid state for reported checkpoint: %s", p.State)
	}

	if p.ReportTime == nil || p.ReportTime.AsTime().IsZero() {
		p.ReportTime = timestamppb.New(time.Now().UTC())
	}

	var storageID *model.StorageBackendID
	if p.StorageId != nil {
		storageID = ptrs.Ptr(model.StorageBackendID(*p.StorageId))
	}

	c := &model.CheckpointV2{
		UUID:         conv.ToUUID(p.Uuid),
		TaskID:       model.TaskID(p.TaskId),
		AllocationID: model.NewAllocationID(p.AllocationId),
		ReportTime:   p.ReportTime.AsTime(),
		State:        conv.ToCheckpointState(p.State),
		Resources:    p.Resources,
		Metadata:     p.Metadata.AsMap(),
		StorageID:    storageID,
	}
	if err := conv.Error(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "converting checkpoint: %s", err)
	}
	return c, nil
}

func (a *apiServer) AllocationRendezvousInfo(
	ctx context.Context, req *apiv1.AllocationRendezvousInfoRequest,
) (*apiv1.AllocationRendezvousInfoResponse, error) {
	if req.AllocationId == "" {
		return nil, status.Error(codes.InvalidArgument, "allocation ID missing")
	}
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	info, err := task.DefaultService.WatchRendezvous(
		ctx,
		model.AllocationID(req.AllocationId),
		sproto.ResourcesID(req.ResourcesId),
	)
	if err != nil {
		return nil, err
	}
	return &apiv1.AllocationRendezvousInfoResponse{RendezvousInfo: info}, nil
}

func (a *apiServer) PostTrialRunnerMetadata(
	ctx context.Context, req *apiv1.PostTrialRunnerMetadataRequest,
) (*apiv1.PostTrialRunnerMetadataResponse, error) {
	if err := trials.CanGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		experiment.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	if err := a.m.db.UpdateTrialFields(int(req.TrialId), req.Metadata, 0, 0); err != nil {
		return nil, err
	}

	return &apiv1.PostTrialRunnerMetadataResponse{}, nil
}

// isTrialTerminalFunc returns an api.TerminationCheckFn that waits for a trial to finish and
// optionally, additionally, waits some buffer duration to give trials a bit to finish sending
// stuff after termination.
func (a *apiServer) isTrialTerminalFunc(trialID int, buffer time.Duration) api.TerminationCheckFn {
	return func() (bool, error) {
		state, endTime, err := a.m.db.TrialStatus(trialID)
		if err != nil ||
			(model.TerminalStates[state] && endTime.Add(buffer).Before(time.Now().UTC())) {
			return true, err
		}
		return false, nil
	}
}

func setInt32(xs ...int32) []int32 {
	s := map[int32]bool{}
	for _, x := range xs {
		s[x] = true
	}

	var nxs []int32
	for x := range s {
		nxs = append(nxs, x)
	}
	return nxs
}

func setString(xs ...string) []string {
	s := map[string]bool{}
	for _, x := range xs {
		s[x] = true
	}

	var nxs []string
	for x := range s {
		nxs = append(nxs, x)
	}
	return nxs
}
