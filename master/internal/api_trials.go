package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hashicorp/go-multierror"

	"github.com/determined-ai/determined/master/internal/protoutil"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/determined-ai/determined/proto/pkg/logv1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
	trialLogsBatchSize            = 1000
	trialProfilerMetricsBatchSize = 100
)

var (
	masterLogsBatchWaitTime           = 100 * time.Millisecond
	trialLogsBatchWaitTime            = 100 * time.Millisecond
	distinctFieldBatchWaitTime        = 5 * time.Second
	trialProfilerMetricsBatchWaitTime = 10 * time.Millisecond
	// TrialAvailableSeriesBatchWaitTime is exported to be changed by tests.
	TrialAvailableSeriesBatchWaitTime = 15 * time.Second

	// Common errors
	trialNotFound = status.Error(codes.NotFound, "trial not found")
)

// TrialLogBackend is an interface trial log backends, such as elastic or postgres,
// must support to provide the features surfaced in API.
type TrialLogBackend interface {
	TrialLogs(
		trialID, limit int, filters []api.Filter, order apiv1.OrderBy, state interface{},
	) ([]*model.TrialLog, interface{}, error)
	AddTrialLogs([]*model.TrialLog) error
	TrialLogsCount(trialID int, filters []api.Filter) (int, error)
	TrialLogsFields(trialID int) (*apiv1.TrialLogsFieldsResponse, error)
	DeleteTrialLogs(trialIDs []int) error
}

func (a *apiServer) TrialLogs(
	req *apiv1.TrialLogsRequest, resp apiv1.Determined_TrialLogsServer) error {
	if err := grpcutil.ValidateRequest(
		grpcutil.ValidateLimit(req.Limit),
		grpcutil.ValidateFollow(req.Limit, req.Follow),
	); err != nil {
		return err
	}

	switch exists, err := a.m.db.CheckTrialExists(int(req.TrialId)); {
	case err != nil:
		return err
	case !exists:
		return status.Error(codes.NotFound, "trial not found")
	}

	filters, err := constructTrialLogsFilters(req)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported filter: %s", err))
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

	onBatch := func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			pl, pErr := r.(*model.TrialLog).Proto()
			if pErr != nil {
				return pErr
			}
			return resp.Send(pl)
		})
	}

	total, err := a.m.trialLogBackend.TrialLogsCount(int(req.TrialId), filters)
	if err != nil {
		return fmt.Errorf("failed to get trial count from backend: %w", err)
	}
	effectiveLimit := api.EffectiveLimit(int(req.Limit), 0, total)

	return api.NewBatchStreamProcessor(
		api.BatchRequest{Limit: effectiveLimit, Follow: req.Follow},
		fetch,
		onBatch,
		a.isTrialTerminalFunc(int(req.TrialId), 20*time.Second),
		trialLogsBatchWaitTime,
	).Run(resp.Context())
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
			switch l {
			case logv1.LogLevel_LOG_LEVEL_UNSPECIFIED:
				levels = append(levels, "DEBUG")
			case logv1.LogLevel_LOG_LEVEL_TRACE:
				levels = append(levels, "TRACE")
			case logv1.LogLevel_LOG_LEVEL_DEBUG:
				levels = append(levels, "DEBUG")
			case logv1.LogLevel_LOG_LEVEL_INFO:
				levels = append(levels, "INFO")
			case logv1.LogLevel_LOG_LEVEL_WARNING:
				levels = append(levels, "WARNING")
			case logv1.LogLevel_LOG_LEVEL_ERROR:
				levels = append(levels, "ERROR")
			case logv1.LogLevel_LOG_LEVEL_CRITICAL:
				levels = append(levels, "CRITICAL")
			}
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

	onBatch := func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(
				r.(*apiv1.TrialLogsFieldsResponse))
		})
	}

	return api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: req.Follow},
		fetch,
		onBatch,
		nil,
		distinctFieldBatchWaitTime,
	).Run(resp.Context())
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
	ok, err := a.m.db.CheckTrialExists(int(req.Id))
	switch {
	case err != nil:
		return nil, status.Errorf(codes.Internal, "failed to check if trial exists: %s", err)
	case !ok:
		return nil, status.Errorf(codes.NotFound, "trial %d not found", req.Id)
	}

	resp := apiv1.KillTrialResponse{}
	addr := actor.Addr("trials", req.Id).String()
	err = a.actorRequest(addr, req, &resp)
	if status.Code(err) == codes.NotFound {
		return &apiv1.KillTrialResponse{}, nil
	}
	return &resp, err
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

	labelsParam, err := protojson.Marshal(req.Labels)
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

		var batchOfBatches []*trialv1.TrialProfilerMetricsBatch
		return model.TrialProfilerMetricsBatchBatch(batchOfBatches), a.m.db.QueryProto(
			"get_trial_profiler_metrics",
			&batchOfBatches, labelsParam, lr.Offset, lr.Limit,
		)
	}

	onBatch := func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(&apiv1.GetTrialProfilerMetricsResponse{
				Batch: r.(*trialv1.TrialProfilerMetricsBatch),
			})
		})
	}

	return api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: true},
		fetch,
		onBatch,
		a.isTrialTerminalFunc(int(req.Labels.TrialId), -1),
		trialProfilerMetricsBatchWaitTime,
	).Run(resp.Context())
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

	onBatch := func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(r.(*apiv1.GetTrialProfilerAvailableSeriesResponse))
		})
	}

	return api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: true},
		fetch,
		onBatch,
		nil,
		TrialAvailableSeriesBatchWaitTime,
	).Run(resp.Context())
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

		timestamps, err := protoutil.TimeSliceFromProto(batch.Timestamps)
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

func (a *apiServer) TrialPreemptionSignal(
	req *apiv1.TrialPreemptionSignalRequest,
	resp apiv1.Determined_TrialPreemptionSignalServer,
) error {
	trial, err := a.trialActorFromID(int(req.TrialId))
	if err != nil {
		return err
	}

	id := uuid.New()
	ret, err := a.askAtDefaultSystem(trial, trialWatchPreemption{id: id})
	if err != nil {
		return err
	}
	defer a.m.system.TellAt(trial, trialUnwatchPreemption{id: id})

	signal, ok := ret.(<-chan bool)
	if !ok {
		return unexpectedMessageError(trial, ret)
	}

	preempt := <-signal
	switch err := resp.Send(&apiv1.TrialPreemptionSignalResponse{Preempt: preempt}); {
	case err != nil:
		return err
	case preempt:
		return nil
	default:
		select {
		case preempt = <-signal:
			return resp.Send(&apiv1.TrialPreemptionSignalResponse{Preempt: preempt})
		case <-resp.Context().Done():
			return nil
		}
	}
}

func (a *apiServer) askAtDefaultSystem(
	addr actor.Address, msg interface{},
) (interface{}, error) {
	switch resp := a.m.system.AskAt(addr, msg); {
	case resp.Source() == nil, resp.Empty(), resp.Get() == nil:
		return nil, status.Errorf(
			codes.NotFound,
			"actor %s could not be found or the actor did not respond", addr,
		)
	case resp.Error() != nil:
		return nil, status.Errorf(
			codes.Internal,
			"actor %s returned error resp %s", addr, resp.Error(),
		)
	default:
		return resp.Get(), nil
	}
}

func unexpectedMessageError(addr actor.Address, resp interface{}) error {
	return status.Errorf(
		codes.Internal,
		"actor %s returned unexpected message (%T): %v", addr, resp, resp,
	)
}

func (a *apiServer) trialActorFromID(trialID int) (actor.Address, error) {
	switch eID, rID, err := a.m.db.TrialExperimentAndRequestID(trialID); {
	case errors.Is(err, db.ErrNotFound):
		return actor.Address{}, trialNotFound
	case err != nil:
		return actor.Address{}, err
	default:
		return actor.Addr("experiments", eID, rID), nil
	}
}

// isTrialTerminalFunc returns an api.TerminationCheckFn that waits for a trial to finish and
// optionally, additionally, waits some buffer duration to give trials a bit to finish sending
// stuff after termination.
func (a *apiServer) isTrialTerminalFunc(trialID int, buffer time.Duration) api.TerminationCheckFn {
	return func() (bool, error) {
		state, endTime, err := a.m.db.TrialStatus(trialID)
		if err != nil ||
			(model.TerminalStates[state] && endTime.Before(time.Now().Add(-buffer))) {
			return true, err
		}
		return false, nil
	}
}
