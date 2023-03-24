package internal

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/lttb"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoless"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
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

func (a *apiServer) canGetTrialsExperimentAndCheckCanDoAction(ctx context.Context,
	trialID int, actionFunc func(context.Context, model.User, *model.Experiment) error,
) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	trialNotFound := status.Errorf(codes.NotFound, "trial %d not found", trialID)
	exp, err := a.m.db.ExperimentByTrialID(trialID)
	if errors.Is(err, db.ErrNotFound) {
		return trialNotFound
	} else if err != nil {
		return err
	}
	var ok bool
	if ok, err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp); err != nil {
		return err
	} else if !ok {
		return trialNotFound
	}

	if err = actionFunc(ctx, *curUser, exp); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}
	return nil
}

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

func (a *apiServer) enrichTrialState(trials ...*trialv1.Trial) error {
	// filter allocations by TaskIDs on this page of trials
	taskFilter := make([]string, 0, len(trials))
	for _, trial := range trials {
		taskFilter = append(taskFilter, trial.TaskId)
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
	byTaskID := make(map[model.TaskID]experimentv1.State, len(tasks))
	for _, task := range tasks {
		switch {
		case task.Running:
			byTaskID[task.Task] = experimentv1.State_STATE_RUNNING
		case task.Starting:
			byTaskID[task.Task] = experimentv1.State_STATE_STARTING
		case task.Pulling:
			byTaskID[task.Task] = experimentv1.State_STATE_PULLING
		default:
			byTaskID[task.Task] = experimentv1.State_STATE_QUEUED
		}
	}

	// Active trials converted to Queued, Pulling, Starting, or Running
	for _, trial := range trials {
		if trial.State == experimentv1.State_STATE_ACTIVE {
			if setState, ok := byTaskID[model.TaskID(trial.TaskId)]; ok {
				trial.State = setState
			} else {
				trial.State = experimentv1.State_STATE_QUEUED
			}
		}
	}
	return nil
}

func (a *apiServer) TrialLogs(
	req *apiv1.TrialLogsRequest, resp apiv1.Determined_TrialLogsServer,
) error {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(resp.Context(), int(req.TrialId),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return err
	}

	var taskID model.TaskID
	switch t, err := a.m.db.TrialByID(int(req.TrialId)); {
	case err != nil:
		return err
	default:
		taskID = t.TaskID
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	switch t, err := a.m.db.TaskByID(taskID); {
	case errors.Is(err, sql.ErrNoRows):
		// This indicates the trial is existed before the task logs table, and has version 0.
		fallthrough
	case t.LogVersion == model.TaskLogVersion0:
		// First stream the legacy logs.
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
		// Then fallthrough and stream the remaining logs, in the event the trial spanned an
		// upgrade. In the event it did not, this should return quickly anyway.
		fallthrough
	case t.LogVersion == model.TaskLogVersion1:
		// Translate the request.
		res := make(chan api.BatchResult, taskLogsChanBuffer)
		go a.taskLogs(ctx, &apiv1.TaskLogsRequest{
			TaskId:          string(taskID),
			Limit:           req.Limit,
			Follow:          req.Follow,
			AllocationIds:   nil,
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
		return processBatches(res, func(b api.Batch) error {
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
	default:
		panic(fmt.Errorf("unknown task log version: %d, please report this bug", t.LogVersion))
	}
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
		if time.Now().Sub(trialLogsTimeSinceLastAuth) >= recheckAuthPeriod {
			if err = a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
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
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(resp.Context(), int(req.TrialId),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return err
	}

	trial, err := a.m.db.TrialByID(int(req.TrialId))
	if err != nil {
		return errors.Wrap(err, "retreiving trial")
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	// Stream fields from trial logs table, just to support pre-task-logs trials with old logs.
	trialLogsTimeSinceLastAuth := time.Now() // time.Now() to avoid recheck from above.
	resOld := make(chan api.BatchResult)
	go api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: req.Follow},
		func(lr api.BatchRequest) (api.Batch, error) {
			if time.Now().Sub(trialLogsTimeSinceLastAuth) >= recheckAuthPeriod {
				if err := a.canGetTrialsExperimentAndCheckCanDoAction(resp.Context(),
					int(req.TrialId),
					expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
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

	// Also stream fields from task logs table, for ordinary logs (as they are written now).
	taskLogsTimeSinceLastAuth := time.Now() // time.Now() to avoid recheck from above.
	resNew := make(chan api.BatchResult)
	go api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: req.Follow},
		func(lr api.BatchRequest) (api.Batch, error) {
			if time.Now().Sub(taskLogsTimeSinceLastAuth) >= recheckAuthPeriod {
				if err := a.canGetTrialsExperimentAndCheckCanDoAction(resp.Context(),
					int(req.TrialId),
					expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
					return nil, err
				}
				taskLogsTimeSinceLastAuth = time.Now()
			}

			fields, err := a.m.taskLogBackend.TaskLogsFields(trial.TaskID)
			return api.ToBatchOfOne(&apiv1.TrialLogsFieldsResponse{
				AgentIds:     fields.AgentIds,
				ContainerIds: fields.ContainerIds,
				RankIds:      fields.RankIds,
				Stdtypes:     fields.Stdtypes,
				Sources:      fields.Sources,
			}), err
		},
		a.isTaskTerminalFunc(trial.TaskID, a.m.taskLogBackend.MaxTerminationDelay()),
		true,
		&distinctFieldBatchWaitTime,
		&distinctFieldBatchWaitTime,
	).Run(ctx, resNew)

	// And merge the available filters.
	return zipBatches(resOld, resNew, func(b1, b2 api.Batch) error {
		r1 := b1.(api.BatchOfOne).Inner.(*apiv1.TrialLogsFieldsResponse)
		r2 := b2.(api.BatchOfOne).Inner.(*apiv1.TrialLogsFieldsResponse)
		return resp.Send(&apiv1.TrialLogsFieldsResponse{
			AgentIds:     setString(append(r1.AgentIds, r2.AgentIds...)...),
			ContainerIds: setString(append(r1.ContainerIds, r2.ContainerIds...)...),
			RankIds:      setInt32(append(r1.RankIds, r2.RankIds...)...),
			Stdtypes:     setString(append(r1.Stdtypes, r2.Stdtypes...)...),
			Sources:      setString(append(r1.Sources, r2.Sources...)...),
		})
	})
}

func (a *apiServer) GetTrialCheckpoints(
	ctx context.Context, req *apiv1.GetTrialCheckpointsRequest,
) (*apiv1.GetTrialCheckpointsResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.Id),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
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

	a.filter(&resp.Checkpoints, func(i int) bool {
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

		switch req.SortBy {
		case apiv1.GetTrialCheckpointsRequest_SORT_BY_BATCH_NUMBER:
			return protoless.CheckpointStepsCompletedLess(ai, aj)
		case apiv1.GetTrialCheckpointsRequest_SORT_BY_UUID:
			return ai.Uuid < aj.Uuid
		case apiv1.GetTrialCheckpointsRequest_SORT_BY_END_TIME:
			return protoless.CheckpointReportTimeLess(ai, aj)
		case apiv1.GetTrialCheckpointsRequest_SORT_BY_STATE:
			return ai.State.Number() < aj.State.Number()
		case apiv1.GetTrialCheckpointsRequest_SORT_BY_UNSPECIFIED:
			fallthrough
		default:
			return protoless.CheckpointStepsCompletedLess(ai, aj)
		}
	})

	return resp, a.paginate(&resp.Pagination, &resp.Checkpoints, req.Offset, req.Limit)
}

func (a *apiServer) KillTrial(
	ctx context.Context, req *apiv1.KillTrialRequest,
) (*apiv1.KillTrialResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.Id),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	t, err := a.m.db.TrialByID(int(req.Id))
	if err != nil {
		return nil, err
	}

	tr := actor.Addr("experiments", t.ExperimentID, t.RequestID)
	s := model.StateWithReason{
		State:               model.StoppingKilledState,
		InformationalReason: "user requested kill",
	}
	if err = a.ask(tr, s, nil); err != nil {
		return nil, err
	}
	return &apiv1.KillTrialResponse{}, nil
}

func (a *apiServer) GetExperimentTrials(
	ctx context.Context, req *apiv1.GetExperimentTrialsRequest,
) (resp *apiv1.GetExperimentTrialsResponse, err error) {
	if _, _, err = a.getExperimentAndCheckCanDoActions(ctx, int(req.ExperimentId),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
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

func (a *apiServer) GetTrial(ctx context.Context, req *apiv1.GetTrialRequest) (
	*apiv1.GetTrialResponse, error,
) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
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

	if resp.Trial.State == experimentv1.State_STATE_ACTIVE {
		if err := a.enrichTrialState(resp.Trial); err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (a *apiServer) formatMetricsBatch(
	m *apiv1.SummarizedMetric, metricSeries []lttb.Point,
) {
	for _, in := range metricSeries {
		out := apiv1.DataPoint{
			Batches: int32(in.X),
			Value:   in.Y,
		}
		m.Data = append(m.Data, &out)
	}
}

func timeFromFloat64(ts float64) time.Time {
	secs := int64(ts)
	nsecs := int64((ts - float64(secs)) * 1e9)
	return time.Unix(secs, nsecs)
}

func (a *apiServer) formatMetricsTime(
	m *apiv1.SummarizedMetric, metricSeries []lttb.Point,
) {
	for _, in := range metricSeries {
		out := apiv1.DataPointTime{
			Time:  timestamppb.New(timeFromFloat64(in.X)),
			Value: in.Y,
		}
		m.Time = append(m.Time, &out)
	}
}

func (a *apiServer) formatMetricsEpoch(
	m *apiv1.SummarizedMetric, metricSeries []lttb.Point,
) {
	for _, in := range metricSeries {
		out := apiv1.DataPointEpoch{
			Epoch: int32(in.X),
			Value: in.Y,
		}
		m.Epochs = append(m.Epochs, &out)
	}
}

func (a *apiServer) MultiTrialSample(trialID int32, metricNames []string,
	metricType apiv1.MetricType, maxDatapoints int, startBatches int,
	endBatches int, logScale bool, xAxis apiv1.XAxis,
) ([]*apiv1.SummarizedMetric, error) {
	var metricSeriesBatch, metricSeriesTime, metricSeriesEpoch []lttb.Point
	var startTime time.Time
	var err error
	var metrics []*apiv1.SummarizedMetric
	var metricMeasurements db.MetricMeasurements
	xAxisLabelMetrics := []string{"epoch"}
	if endBatches == 0 {
		endBatches = math.MaxInt32
	}

	for _, name := range metricNames {
		if (metricType == apiv1.MetricType_METRIC_TYPE_TRAINING) ||
			(metricType == apiv1.MetricType_METRIC_TYPE_UNSPECIFIED) {
			var metric apiv1.SummarizedMetric
			metric.Name = name
			metricMeasurements, err = a.m.db.TrainingMetricsSeries(
				trialID, startTime, name, startBatches, endBatches, xAxisLabelMetrics)
			if err != nil {
				return nil, errors.Wrapf(err, "error fetching time series of training metrics")
			}
			metric.Type = apiv1.MetricType_METRIC_TYPE_TRAINING

			metricSeriesTime = lttb.Downsample(metricMeasurements.Time, maxDatapoints, logScale)
			a.formatMetricsTime(&metric, metricSeriesTime)
			metricSeriesBatch = lttb.Downsample(metricMeasurements.Batches, maxDatapoints, logScale)
			a.formatMetricsBatch(&metric, metricSeriesBatch)

			// For now "epoch" is the only custom xAxis metric label supported so we
			// build the `MetricSeriesEpoch` array. In the future this logic should
			// be updated to support any number of xAxis metric options
			metricSeriesEpoch = lttb.Downsample(metricMeasurements.AverageMetrics["epoch"],
				maxDatapoints, logScale)
			a.formatMetricsEpoch(&metric, metricSeriesEpoch)

			if len(metricSeriesBatch) > 0 || len(metricSeriesTime) > 0 {
				metrics = append(metrics, &metric)
			}
		}
		if (metricType == apiv1.MetricType_METRIC_TYPE_VALIDATION) ||
			(metricType == apiv1.MetricType_METRIC_TYPE_UNSPECIFIED) {
			var metric apiv1.SummarizedMetric
			metric.Name = name
			metricMeasurements, err = a.m.db.ValidationMetricsSeries(
				trialID, startTime, name, startBatches, endBatches, xAxisLabelMetrics)
			if err != nil {
				return nil, errors.Wrapf(err, "error fetching time series of validation metrics")
			}
			metric.Type = apiv1.MetricType_METRIC_TYPE_VALIDATION

			metricSeriesTime = lttb.Downsample(metricMeasurements.Time, maxDatapoints, logScale)
			a.formatMetricsTime(&metric, metricSeriesTime)
			metricSeriesBatch = lttb.Downsample(metricMeasurements.Batches, maxDatapoints, logScale)
			a.formatMetricsBatch(&metric, metricSeriesBatch)
			metricSeriesEpoch = lttb.Downsample(metricMeasurements.AverageMetrics["epoch"],
				maxDatapoints, logScale)
			a.formatMetricsEpoch(&metric, metricSeriesEpoch)

			if len(metricSeriesBatch) > 0 || len(metricSeriesTime) > 0 {
				metrics = append(metrics, &metric)
			}
		}
	}
	return metrics, nil
}

func (a *apiServer) SummarizeTrial(ctx context.Context,
	req *apiv1.SummarizeTrialRequest,
) (*apiv1.SummarizeTrialResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	resp := &apiv1.SummarizeTrialResponse{Trial: &trialv1.Trial{}}
	if err := a.m.db.QueryProto("get_trial_basic", resp.Trial, req.TrialId); err != nil {
		return nil, errors.Wrapf(err, "failed to get trial %d", req.TrialId)
	}

	tsample, err := a.MultiTrialSample(req.TrialId, req.MetricNames, req.MetricType,
		int(req.MaxDatapoints), int(req.StartBatches), int(req.EndBatches),
		(req.Scale == apiv1.Scale_SCALE_LOG), apiv1.XAxis_X_AXIS_UNSPECIFIED)
	if err != nil {
		return nil, errors.Wrapf(err, "failed sampling")
	}
	resp.Metrics = tsample

	return resp, nil
}

func (a *apiServer) CompareTrials(ctx context.Context,
	req *apiv1.CompareTrialsRequest,
) (*apiv1.CompareTrialsResponse, error) {
	trials := make([]*apiv1.ComparableTrial, 0, len(req.TrialIds))
	for _, trialID := range req.TrialIds {
		if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(trialID),
			expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
			return nil, err
		}

		container := &apiv1.ComparableTrial{Trial: &trialv1.Trial{}}
		switch err := a.m.db.QueryProto("get_trial_basic", container.Trial, trialID); {
		case err == db.ErrNotFound:
			return nil, status.Errorf(codes.NotFound, "trial %d not found:", trialID)
		case err != nil:
			return nil, errors.Wrapf(err, "failed to get trial %d", trialID)
		}

		tsample, err := a.MultiTrialSample(trialID, req.MetricNames, req.MetricType,
			int(req.MaxDatapoints), int(req.StartBatches), int(req.EndBatches),
			(req.Scale == apiv1.Scale_SCALE_LOG), req.XAxis)
		if err != nil {
			return nil, errors.Wrapf(err, "failed sampling")
		}
		container.Metrics = tsample
		trials = append(trials, container)
	}
	return &apiv1.CompareTrialsResponse{Trials: trials}, nil
}

func (a *apiServer) GetTrialWorkloads(ctx context.Context, req *apiv1.GetTrialWorkloadsRequest) (
	*apiv1.GetTrialWorkloadsResponse, error,
) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}

	resp := &apiv1.GetTrialWorkloadsResponse{}
	limit := &req.Limit
	if *limit == 0 {
		limit = ptrs.Ptr[int32](-1)
	}

	sortCode := "total_batches"
	if req.SortKey != "" && req.SortKey != "batches" {
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
		req.MetricType.String(),
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
	labelsJSON, err := protojson.Marshal(req.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	var timeSinceLastAuth time.Time
	fetch := func(lr api.BatchRequest) (api.Batch, error) {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if err := a.canGetTrialsExperimentAndCheckCanDoAction(resp.Context(),
				int(req.Labels.TrialId),
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
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
		return a.m.db.GetTrialProfilerMetricsBatches(labelsJSON, lr.Offset, lr.Limit)
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
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if err := a.canGetTrialsExperimentAndCheckCanDoAction(resp.Context(),
				int(req.TrialId),
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return nil, err
			}
			timeSinceLastAuth = time.Now()
		}

		var labels apiv1.GetTrialProfilerAvailableSeriesResponse
		return api.ToBatchOfOne(&labels), a.m.db.QueryProto(
			"get_trial_available_series",
			&labels, fmt.Sprintf(`{"trialId": %d}`, req.TrialId),
		)
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
			if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, trialID,
				expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
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

func (a *apiServer) waitForAllocationToBeRestored(ctx context.Context, handler *actor.Ref) error {
	for i := 0; i < 60; i++ {
		var restoring bool
		if err := a.ask(handler.Address(), task.IsAllocationRestoring{}, &restoring); err != nil {
			return errors.Wrap(err, "failed to ask allocation actor about restoring status")
		}
		if !restoring {
			return nil
		}

		time.Sleep(time.Second)
	}
	return fmt.Errorf("allocation stuck restoring after one minute of retrying")
}

func (a *apiServer) AllocationPreemptionSignal(
	ctx context.Context,
	req *apiv1.AllocationPreemptionSignalRequest,
) (*apiv1.AllocationPreemptionSignalResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	allocationID := model.AllocationID(req.AllocationId)
	handler, err := a.m.rm.GetAllocationHandler(
		a.m.system,
		sproto.GetAllocationHandler{ID: allocationID},
	)
	if err != nil {
		return nil, err
	}
	if err := a.waitForAllocationToBeRestored(ctx, handler); err != nil {
		return nil, err
	}

	id := uuid.New()
	var w task.PreemptionWatcher
	if err := a.ask(handler.Address(), task.WatchPreemption{
		ID: id, AllocationID: allocationID,
	}, &w); err != nil {
		return nil, err
	}
	defer a.m.system.TellAt(handler.Address(), task.UnwatchPreemption{ID: id})

	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()
	select {
	case <-w.C:
		return &apiv1.AllocationPreemptionSignalResponse{Preempt: true}, nil
	case <-ctx.Done():
		return &apiv1.AllocationPreemptionSignalResponse{Preempt: false}, nil
	}
}

func (a *apiServer) AckAllocationPreemptionSignal(
	ctx context.Context, req *apiv1.AckAllocationPreemptionSignalRequest,
) (*apiv1.AckAllocationPreemptionSignalResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	allocationID := model.AllocationID(req.AllocationId)
	handler, err := a.m.rm.GetAllocationHandler(
		a.m.system,
		sproto.GetAllocationHandler{ID: allocationID},
	)
	if err != nil {
		return nil, err
	}
	if err := a.waitForAllocationToBeRestored(ctx, handler); err != nil {
		return nil, err
	}

	if err := a.ask(handler.Address(), task.AckPreemption{
		AllocationID: allocationID,
	}, nil); err != nil {
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
		a.m.system,
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
		a.m.system,
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
	allocationID := model.AllocationID(req.AllocationId)

	handler, err := a.m.rm.GetAllocationHandler(
		a.m.system,
		sproto.GetAllocationHandler{ID: allocationID},
	)
	if err != nil {
		return nil, err
	}
	if err := a.waitForAllocationToBeRestored(ctx, handler); err != nil {
		return nil, err
	}

	if err := a.ask(handler.Address(), task.MarkResourcesDaemon{
		AllocationID: allocationID,
		ResourcesID:  sproto.ResourcesID(req.ResourcesId),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.MarkAllocationResourcesDaemonResponse{}, nil
}

func (a *apiServer) GetCurrentTrialSearcherOperation(
	ctx context.Context, req *apiv1.GetCurrentTrialSearcherOperationRequest,
) (*apiv1.GetCurrentTrialSearcherOperationResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	if err != nil {
		return nil, err
	}
	exp := actor.Addr("experiments", eID)

	var resp trialSearcherState
	if err := a.ask(exp, trialGetSearcherState{
		requestID: rID,
	}, &resp); err != nil {
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
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	if err != nil {
		return nil, err
	}
	exp := actor.Addr("experiments", eID)

	if err = a.ask(exp, trialCompleteOperation{
		requestID: rID,
		metric:    req.CompletedOperation.SearcherMetric.AsInterface(),
		op:        searcher.NewValidateAfter(rID, req.CompletedOperation.Op.Length),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.CompleteTrialSearcherValidationResponse{}, nil
}

func (a *apiServer) ReportTrialSearcherEarlyExit(
	ctx context.Context, req *apiv1.ReportTrialSearcherEarlyExitRequest,
) (*apiv1.ReportTrialSearcherEarlyExitResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	if err != nil {
		return nil, err
	}
	trial := actor.Addr("experiments", eID, rID)

	if err = a.ask(trial, userInitiatedEarlyExit{
		requestID: rID,
		reason:    model.ExitedReasonFromProto(req.EarlyExit.Reason),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialSearcherEarlyExitResponse{}, nil
}

func (a *apiServer) ReportTrialProgress(
	ctx context.Context, req *apiv1.ReportTrialProgressRequest,
) (*apiv1.ReportTrialProgressResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	if err != nil {
		return nil, err
	}
	exp := actor.Addr("experiments", eID)

	if err = a.ask(exp, trialReportProgress{
		requestID: rID,
		progress:  searcher.PartialUnits(req.Progress),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialProgressResponse{}, nil
}

func (a *apiServer) ReportTrialTrainingMetrics(
	ctx context.Context, req *apiv1.ReportTrialTrainingMetricsRequest,
) (*apiv1.ReportTrialTrainingMetricsResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrainingMetrics.TrialId),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	if err := a.m.db.AddTrainingMetrics(ctx, req.TrainingMetrics); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialTrainingMetricsResponse{}, nil
}

func (a *apiServer) ReportTrialValidationMetrics(
	ctx context.Context, req *apiv1.ReportTrialValidationMetricsRequest,
) (*apiv1.ReportTrialValidationMetricsResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.ValidationMetrics.TrialId),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}
	if err := a.m.db.AddValidationMetrics(ctx, req.ValidationMetrics); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialValidationMetricsResponse{}, nil
}

func (a *apiServer) ReportCheckpoint(
	ctx context.Context, req *apiv1.ReportCheckpointRequest,
) (*apiv1.ReportCheckpointResponse, error) {
	if err := a.canDoActionsOnTask(ctx, model.TaskID(req.Checkpoint.TaskId),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
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

	if err := db.AddCheckpointMetadata(ctx, c); err != nil {
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

	c := &model.CheckpointV2{
		UUID:         conv.ToUUID(p.Uuid),
		TaskID:       model.TaskID(p.TaskId),
		AllocationID: model.AllocationID(p.AllocationId),
		ReportTime:   p.ReportTime.AsTime(),
		State:        conv.ToCheckpointState(p.State),
		Resources:    p.Resources,
		Metadata:     p.Metadata.AsMap(),
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

	allocationID := model.AllocationID(req.AllocationId)
	resourcesID := sproto.ResourcesID(req.ResourcesId)
	handler, err := a.m.rm.GetAllocationHandler(
		a.m.system,
		sproto.GetAllocationHandler{ID: allocationID},
	)
	if err != nil {
		return nil, err
	}
	if err := a.waitForAllocationToBeRestored(ctx, handler); err != nil {
		return nil, err
	}

	var w task.RendezvousWatcher
	if err = a.ask(handler.Address(), task.WatchRendezvousInfo{
		ResourcesID: resourcesID,
	}, &w); err != nil {
		return nil, err
	}
	defer a.m.system.TellAt(
		handler.Address(), task.UnwatchRendezvousInfo{
			ResourcesID: resourcesID,
		})

	select {
	case rsp := <-w.C:
		if rsp.Err != nil {
			return nil, rsp.Err
		}
		return &apiv1.AllocationRendezvousInfoResponse{RendezvousInfo: rsp.Info}, nil
	case <-ctx.Done():
		return nil, nil
	}
}

func (a *apiServer) PostTrialRunnerMetadata(
	ctx context.Context, req *apiv1.PostTrialRunnerMetadataRequest,
) (*apiv1.PostTrialRunnerMetadataResponse, error) {
	if err := a.canGetTrialsExperimentAndCheckCanDoAction(ctx, int(req.TrialId),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
		return nil, err
	}

	if err := a.m.db.UpdateTrialRunnerMetadata(int(req.TrialId), req.Metadata); err != nil {
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
