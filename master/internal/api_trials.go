package internal

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
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
	trialProfilerMetricsBatchWaitTime     = 100 * time.Millisecond
	trialProfilerMetricsBatchMissWaitTime = 5 * time.Second

	// TrialAvailableSeriesBatchWaitTime is exported to be changed by tests.
	TrialAvailableSeriesBatchWaitTime = 15 * time.Second

	// Common errors
	trialNotFound = status.Error(codes.NotFound, "trial not found")
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

func (a *apiServer) TrialLogs(
	req *apiv1.TrialLogsRequest, resp apiv1.Determined_TrialLogsServer,
) error {
	var taskID model.TaskID
	switch t, err := a.m.db.TrialByID(int(req.TrialId)); {
	case errors.Is(err, sql.ErrNoRows), errors.Is(err, db.ErrNotFound):
		return trialNotFound
	case err != nil:
		return err
	default:
		taskID = t.TaskID
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult, 1)
	switch t, err := a.m.db.TaskByID(taskID); {
	case errors.Is(err, sql.ErrNoRows), t.LogVersion == 0:
		go a.legacyTrialLogs(ctx, req, res)
		return processBatches(res, func(b api.Batch) error {
			return b.ForEach(func(i interface{}) error {
				l, err := i.(*model.TrialLog).Proto()
				if err != nil {
					return err
				}
				return resp.Send(l)
			})
		})
	default:
		// Translate the request.
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
		}, res)
		return processBatches(res, func(b api.Batch) error {
			return b.ForEach(func(i interface{}) error {
				l, err := i.(*model.TaskLog).Proto()
				if err != nil {
					return err
				}
				return resp.Send(&apiv1.TrialLogsResponse{
					Id:        l.Id,
					Timestamp: l.Timestamp,
					Message:   l.Message,
					Level:     l.Level,
				})
			})
		})
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
	fetch := func(r api.BatchRequest) (api.Batch, error) {
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
		taskLogsBatchWaitTime,
		taskLogsBatchMissWaitTime,
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
	req *apiv1.TrialLogsFieldsRequest, resp apiv1.Determined_TrialLogsFieldsServer) error {
	fetch := func(lr api.BatchRequest) (api.Batch, error) {
		fields, err := a.m.trialLogBackend.TrialLogsFields(int(req.TrialId))
		return api.ToBatchOfOne(fields), err
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult)
	go api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: req.Follow},
		fetch,
		a.isTrialTerminalFunc(int(req.TrialId), a.m.taskLogBackend.MaxTerminationDelay()),
		true,
		taskLogsFieldsBatchWaitTime,
		taskLogsFieldsBatchWaitTime,
	).Run(ctx, res)

	return processBatches(res, func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(r.(*apiv1.TrialLogsFieldsResponse))
		})
	})
}

func (a *apiServer) GetTrialCheckpoints(
	_ context.Context, req *apiv1.GetTrialCheckpointsRequest,
) (*apiv1.GetTrialCheckpointsResponse, error) {
	switch exists, err := a.m.db.CheckTrialExists(int(req.Id)); {
	case err != nil:
		return nil, err
	case !exists:
		return nil, status.Error(codes.NotFound, "trial not found")
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

		found = false
		for _, state := range req.ValidationStates {
			if state == v.ValidationState {
				found = true
				break
			}
		}

		if len(req.ValidationStates) != 0 && !found {
			return false
		}

		return true
	})

	a.sort(
		resp.Checkpoints, req.OrderBy, req.SortBy, apiv1.GetTrialCheckpointsRequest_SORT_BY_BATCH_NUMBER)

	return resp, a.paginate(&resp.Pagination, &resp.Checkpoints, req.Offset, req.Limit)
}

func (a *apiServer) KillTrial(
	ctx context.Context, req *apiv1.KillTrialRequest,
) (*apiv1.KillTrialResponse, error) {
	t, err := a.m.db.TrialByID(int(req.Id))
	switch {
	case errors.Cause(err) == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "trial %d not found", req.Id)
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to get trial: %s", err)
	}

	tr := actor.Addr("experiments", t.ExperimentID, t.RequestID)
	if err = a.ask(tr, model.StoppingKilledState, nil); err != nil {
		return nil, err
	}
	return &apiv1.KillTrialResponse{}, nil
}

func (a *apiServer) GetExperimentTrials(
	_ context.Context, req *apiv1.GetExperimentTrialsRequest,
) (*apiv1.GetExperimentTrialsResponse, error) {
	ok, err := a.m.db.CheckExperimentExists(int(req.ExperimentId))
	switch {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to check if experiment exists: %s", err)
	case !ok:
		return nil, status.Errorf(codes.NotFound, "experiment %d not found", req.ExperimentId)
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

	resp := &apiv1.GetExperimentTrialsResponse{}
	switch err := a.m.db.QueryProtof(
		"proto_get_trial_ids_for_experiment",
		[]interface{}{orderExpr},
		resp,
		req.ExperimentId,
		stateFilterExpr,
		req.Offset,
		req.Limit,
	); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "experiment %d not found:", req.ExperimentId)
	case err != nil:
		return nil, errors.Wrapf(err, "failed to get trials for experiment %d", req.ExperimentId)
	}

	trialIds := make([]string, 0)
	for _, trial := range resp.Trials {
		trialIds = append(trialIds, strconv.Itoa(int(trial.Id)))
	}

	switch err := a.m.db.QueryProto(
		"proto_get_trials_plus",
		&resp.Trials,
		"{"+strings.Join(trialIds, ",")+"}",
	); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "trials %v not found:", trialIds)
	case err != nil:
		return nil, errors.Wrapf(err, "failed to get trials for experiment %d", req.ExperimentId)
	}

	return resp, nil
}

func (a *apiServer) GetTrial(_ context.Context, req *apiv1.GetTrialRequest) (
	*apiv1.GetTrialResponse, error,
) {
	resp := &apiv1.GetTrialResponse{Trial: &trialv1.Trial{}}
	switch err := a.m.db.QueryProto(
		"proto_get_trials_plus",
		resp.Trial,
		"{"+strconv.Itoa(int(req.TrialId))+"}",
	); {
	case err == db.ErrNotFound:
		return nil, status.Errorf(codes.NotFound, "trial %d not found:", req.TrialId)
	case err != nil:
		return nil, errors.Wrapf(err, "failed to get trial %d", req.TrialId)
	}

	switch err := a.m.db.QueryProto(
		"proto_get_trial_workloads",
		&resp.Workloads,
		req.TrialId,
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
	switch exists, err := a.m.db.CheckTrialExists(int(req.Labels.TrialId)); {
	case err != nil:
		return err
	case !exists:
		return status.Error(codes.NotFound, "trial not found")
	}

	labelsJSON, err := protojson.Marshal(req.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	fetch := func(lr api.BatchRequest) (api.Batch, error) {
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
		trialProfilerMetricsBatchWaitTime,
		trialProfilerMetricsBatchMissWaitTime,
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
	switch exists, err := a.m.db.CheckTrialExists(int(req.TrialId)); {
	case err != nil:
		return err
	case !exists:
		return trialNotFound
	}

	fetch := func(_ api.BatchRequest) (api.Batch, error) {
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
		TrialAvailableSeriesBatchWaitTime,
		TrialAvailableSeriesBatchWaitTime,
	).Run(ctx, res)

	return processBatches(res, func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(r.(*apiv1.GetTrialProfilerAvailableSeriesResponse))
		})
	})
}

func (a *apiServer) PostTrialProfilerMetricsBatch(
	_ context.Context,
	req *apiv1.PostTrialProfilerMetricsBatchRequest,
) (*apiv1.PostTrialProfilerMetricsBatchResponse, error) {
	var errs *multierror.Error
	existingTrials := map[int]bool{}
	for _, batch := range req.Batches {
		trialID := int(batch.Labels.TrialId)
		if !existingTrials[trialID] {
			switch exists, err := a.m.db.CheckTrialExists(trialID); {
			case err != nil:
				errs = multierror.Append(errs, err)
				continue
			case !exists:
				errs = multierror.Append(errs, status.Error(codes.NotFound, "trial not found"))
				continue
			default:
				existingTrials[trialID] = true
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
	allocationID := model.AllocationID(req.AllocationId)
	handler, err := a.allocationHandlerByID(allocationID)
	if err != nil {
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
	_ context.Context, req *apiv1.AckAllocationPreemptionSignalRequest,
) (*apiv1.AckAllocationPreemptionSignalResponse, error) {
	handler, err := a.allocationHandlerByID(model.AllocationID(req.AllocationId))
	if err != nil {
		return nil, err
	}

	if err := a.ask(handler.Address(), task.AckPreemption{
		AllocationID: model.AllocationID(req.AllocationId),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.AckAllocationPreemptionSignalResponse{}, nil
}

func (a *apiServer) MarkAllocationReservationDaemon(
	_ context.Context, req *apiv1.MarkAllocationReservationDaemonRequest,
) (*apiv1.MarkAllocationReservationDaemonResponse, error) {
	handler, err := a.allocationHandlerByID(model.AllocationID(req.AllocationId))
	if err != nil {
		return nil, err
	}

	if err := a.ask(handler.Address(), task.MarkReservationDaemon{
		AllocationID: model.AllocationID(req.AllocationId),
		ContainerID:  cproto.ID(req.ContainerId),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.MarkAllocationReservationDaemonResponse{}, nil
}

func (a *apiServer) GetCurrentTrialSearcherOperation(
	_ context.Context, req *apiv1.GetCurrentTrialSearcherOperationRequest,
) (*apiv1.GetCurrentTrialSearcherOperationResponse, error) {
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	switch {
	case errors.Is(err, db.ErrNotFound):
		return nil, trialNotFound
	case err != nil:
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
		Op: &experimentv1.SearcherOperation{
			Union: &experimentv1.SearcherOperation_ValidateAfter{
				ValidateAfter: resp.Op.ToProto(),
			},
		},
		Completed: resp.Complete,
	}, nil
}

func (a *apiServer) CompleteTrialSearcherValidation(
	_ context.Context, req *apiv1.CompleteTrialSearcherValidationRequest,
) (*apiv1.CompleteTrialSearcherValidationResponse, error) {
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	switch {
	case errors.Is(err, db.ErrNotFound):
		return nil, trialNotFound
	case err != nil:
		return nil, err
	}
	exp := actor.Addr("experiments", eID)

	if err = a.ask(exp, trialCompleteOperation{
		requestID: rID,
		metric:    req.CompletedOperation.SearcherMetric,
		op:        searcher.ValidateAfterFromProto(rID, req.CompletedOperation.Op),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.CompleteTrialSearcherValidationResponse{}, nil
}

func (a *apiServer) ReportTrialSearcherEarlyExit(
	_ context.Context, req *apiv1.ReportTrialSearcherEarlyExitRequest,
) (*apiv1.ReportTrialSearcherEarlyExitResponse, error) {
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	switch {
	case errors.Is(err, db.ErrNotFound):
		return nil, trialNotFound
	case err != nil:
		return nil, err
	}
	exp := actor.Addr("experiments", eID)

	if err = a.ask(exp, trialReportEarlyExit{
		requestID: rID,
		reason:    model.ExitedReasonFromProto(req.EarlyExit.Reason),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialSearcherEarlyExitResponse{}, nil
}

func (a *apiServer) ReportTrialProgress(
	_ context.Context, req *apiv1.ReportTrialProgressRequest,
) (*apiv1.ReportTrialProgressResponse, error) {
	eID, rID, err := a.m.db.TrialExperimentAndRequestID(int(req.TrialId))
	switch {
	case errors.Is(err, db.ErrNotFound):
		return nil, trialNotFound
	case err != nil:
		return nil, err
	}
	exp := actor.Addr("experiments", eID)

	if err = a.ask(exp, trialReportProgress{
		requestID: rID,
		progress:  model.PartialUnits(req.Progress),
	}, nil); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialProgressResponse{}, nil
}

func (a *apiServer) ReportTrialTrainingMetrics(
	ctx context.Context, req *apiv1.ReportTrialTrainingMetricsRequest,
) (*apiv1.ReportTrialTrainingMetricsResponse, error) {
	if err := a.checkTrialExists(int(req.TrainingMetrics.TrialId)); err != nil {
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
	if err := a.checkTrialExists(int(req.ValidationMetrics.TrialId)); err != nil {
		return nil, err
	}
	if err := a.m.db.AddValidationMetrics(ctx, req.ValidationMetrics); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialValidationMetricsResponse{}, nil
}

func (a *apiServer) ReportTrialCheckpointMetadata(
	ctx context.Context, req *apiv1.ReportTrialCheckpointMetadataRequest,
) (*apiv1.ReportTrialCheckpointMetadataResponse, error) {
	if err := a.checkTrialExists(int(req.CheckpointMetadata.TrialId)); err != nil {
		return nil, err
	}
	if err := a.m.db.AddCheckpointMetadata(ctx, req.CheckpointMetadata); err != nil {
		return nil, err
	}
	return &apiv1.ReportTrialCheckpointMetadataResponse{}, nil
}

func (a *apiServer) AllocationRendezvousInfo(
	ctx context.Context, req *apiv1.AllocationRendezvousInfoRequest,
) (*apiv1.AllocationRendezvousInfoResponse, error) {
	if req.AllocationId == "" {
		return nil, status.Error(codes.InvalidArgument, "allocation ID missing")
	}

	handler, err := a.allocationHandlerByID(model.AllocationID(req.AllocationId))
	if err != nil {
		return nil, err
	}

	var w task.RendezvousWatcher
	if err = a.ask(handler.Address(), task.WatchRendezvousInfo{
		AllocationID: model.AllocationID(req.AllocationId),
		ContainerID:  cproto.ID(req.ContainerId),
	}, &w); err != nil {
		return nil, err
	}
	defer a.m.system.TellAt(
		handler.Address(), task.UnwatchRendezvousInfo{ID: cproto.ID(req.ContainerId)})

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

func (a *apiServer) allocationHandlerByID(id model.AllocationID) (*actor.Ref, error) {
	var handler *actor.Ref
	if err := a.ask(a.m.rm.Address(), sproto.GetTaskHandler{ID: id}, &handler); err != nil {
		return nil, err
	}
	return handler, nil
}

func (a *apiServer) PostTrialRunnerMetadata(
	_ context.Context, req *apiv1.PostTrialRunnerMetadataRequest,
) (*apiv1.PostTrialRunnerMetadataResponse, error) {
	if err := a.checkTrialExists(int(req.TrialId)); err != nil {
		return nil, err
	}

	if err := a.m.db.UpdateTrialRunnerMetadata(int(req.TrialId), req.Metadata); err != nil {
		return nil, err
	}

	return &apiv1.PostTrialRunnerMetadataResponse{}, nil
}

func (a *apiServer) checkTrialExists(id int) error {
	ok, err := a.m.db.CheckTrialExists(id)
	switch {
	case err != nil:
		return status.Errorf(codes.Internal, "failed to check if trial exists: %s", err)
	case !ok:
		return status.Errorf(codes.NotFound, "trial %d not found", id)
	default:
		return nil
	}
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
