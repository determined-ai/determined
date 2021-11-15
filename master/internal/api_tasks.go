package internal

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/go-multierror"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	taskLogsBatchSize = 1000
)

var (
	taskReadyCheckLogs = "/run/determined/check_ready_logs.py"

	taskLogsBatchWaitTime       = 100 * time.Millisecond
	taskLogsBatchMissWaitTime   = time.Second
	taskLogsFieldsBatchWaitTime = 5 * time.Second

	// Common errors
	taskNotFound = status.Error(codes.NotFound, "task not found")
)

func (a *apiServer) AllocationReady(
	ctx context.Context, req *apiv1.AllocationReadyRequest,
) (*apiv1.AllocationReadyResponse, error) {
	handler, err := a.allocationHandlerByID(model.AllocationID(req.AllocationId))
	if err != nil {
		return nil, err
	}

	if err := a.ask(handler.Address(), task.AllocationReady{}, nil); err != nil {
		return nil, err
	}
	return &apiv1.AllocationReadyResponse{}, nil
}

func (a *apiServer) TaskLogs(
	req *apiv1.TaskLogsRequest, resp apiv1.Determined_TaskLogsServer,
) error {
	if err := grpcutil.ValidateRequest(
		grpcutil.ValidateLimit(req.Limit),
		grpcutil.ValidateFollow(req.Limit, req.Follow),
	); err != nil {
		return err
	}

	taskID := model.TaskID(req.TaskId)
	switch exists, err := a.m.db.CheckTaskExists(taskID); {
	case err != nil:
		return err
	case !exists:
		return taskNotFound
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult)
	go a.taskLogs(ctx, req, res)

	return processBatches(res, func(b api.Batch) error {
		return b.ForEach(func(i interface{}) error {
			pl, pErr := i.(*model.TaskLog).Proto()
			if pErr != nil {
				return pErr
			}
			return resp.Send(pl)
		})
	})
}

func (a *apiServer) taskLogs(
	ctx context.Context, req *apiv1.TaskLogsRequest, res chan api.BatchResult,
) {
	taskID := model.TaskID(req.TaskId)
	filters, err := constructTaskLogsFilters(req)
	if err != nil {
		res <- api.ErrBatchResult(
			status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported filter: %s", err)),
		)
		return
	}

	var followState interface{}
	fetch := func(r api.BatchRequest) (api.Batch, error) {
		switch {
		case r.Follow, r.Limit > taskLogsBatchSize:
			r.Limit = taskLogsBatchSize
		case r.Limit <= 0:
			return nil, nil
		}

		b, state, fErr := a.m.taskLogBackend.TaskLogs(
			taskID, r.Limit, filters, req.OrderBy, followState)
		if fErr != nil {
			return nil, fErr
		}
		followState = state

		return model.TaskLogBatch(b), nil
	}

	total, err := a.m.taskLogBackend.TaskLogsCount(taskID, filters)
	if err != nil {
		res <- api.ErrBatchResult(fmt.Errorf("getting log count from backend: %w", err))
		return
	}
	effectiveLimit := api.EffectiveLimit(int(req.Limit), 0, total)

	api.NewBatchStreamProcessor(
		api.BatchRequest{Limit: effectiveLimit, Follow: req.Follow},
		fetch,
		a.isTaskTerminalFunc(taskID, a.m.taskLogBackend.MaxTerminationDelay()),
		false,
		taskLogsBatchWaitTime,
		taskLogsBatchMissWaitTime,
	).Run(ctx, res)
}

func constructTaskLogsFilters(req *apiv1.TaskLogsRequest) ([]api.Filter, error) {
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

	addInFilter("allocation_id", req.AllocationIds, len(req.AllocationIds))
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

func (a *apiServer) TaskLogsFields(
	req *apiv1.TaskLogsFieldsRequest, resp apiv1.Determined_TaskLogsFieldsServer,
) error {
	taskID := model.TaskID(req.TaskId)
	fetch := func(lr api.BatchRequest) (api.Batch, error) {
		fields, err := a.m.taskLogBackend.TaskLogsFields(taskID)
		return api.ToBatchOfOne(fields), err
	}

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult)
	go api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: req.Follow},
		fetch,
		a.isTaskTerminalFunc(taskID, a.m.taskLogBackend.MaxTerminationDelay()),
		true,
		taskLogsFieldsBatchWaitTime,
		taskLogsFieldsBatchWaitTime,
	).Run(ctx, res)

	return processBatches(res, func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(r.(*apiv1.TaskLogsFieldsResponse))
		})
	})
}

// isTaskTerminalFunc returns an api.TerminationCheckFn that waits for a task to finish and
// optionally, additionally, waits some buffer duration to give trials a bit to finish sending
// stuff after termination.
func (a *apiServer) isTaskTerminalFunc(
	taskID model.TaskID, buffer time.Duration,
) api.TerminationCheckFn {
	return func() (bool, error) {
		switch task, err := a.m.db.TaskByID(taskID); {
		case err != nil:
			return true, err
		case task.EndTime != nil && task.EndTime.UTC().Add(buffer).Before(time.Now().UTC()):
			return true, nil
		default:
			return false, nil
		}
	}
}

func processBatches(res chan api.BatchResult, h func(api.Batch) error) error {
	var err *multierror.Error
	for r := range res {
		if r.Err() != nil {
			// Noting the failure but not exiting here will cause us to wait for the downstream
			// processor to fail from its error or continue.
			err = multierror.Append(err, r.Err())
			continue
		}

		hErr := h(r.Batch())
		if hErr != nil {
			// Since this is our failure, we fail and return. This should cause upstream
			// processses and cause downstream senders to cancel.
			return hErr
		}
	}
	return err.ErrorOrNil()
}
