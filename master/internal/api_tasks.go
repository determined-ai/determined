package internal

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/logv1"
)

const (
	taskLogsBatchSize = 1000
)

var (
	taskLogsBatchWaitTime       = 100 * time.Millisecond
	taskLogsBatchMissWaitTime   = time.Second
	taskLogsTerminationDelay    = 2 * time.Second
	taskLogsFieldsBatchWaitTime = 5 * time.Second

	// Common errors
	taskNotFound = status.Error(codes.NotFound, "task not found")
)

// TaskLogBackend is an interface trial log backends, such as elastic or postgres,
// must support to provide the features surfaced in API.
type TaskLogBackend interface {
	TaskLogs(
		taskID model.TaskID, limit int, filters []api.Filter, order apiv1.OrderBy, state interface{},
	) ([]*model.TaskLog, interface{}, error)
	AddTaskLogs([]*model.TaskLog) error
	TaskLogsCount(taskID model.TaskID, filters []api.Filter) (int, error)
	TaskLogsFields(taskID model.TaskID) (*apiv1.TaskLogsFieldsResponse, error)
	DeleteTaskLogs(taskIDs []model.TaskID) error
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

	return a.taskLogs(req, resp)
}

func (a *apiServer) taskLogs(
	ctx context.Context, req *apiv1.TaskLogsRequest, handler func(*model.TaskLog),
) error {
	taskID := model.TaskID(req.TaskId)
	filters, err := constructTaskLogsFilters(req)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("unsupported filter: %s", err))
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

	onBatch := func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			pl, pErr := r.(*model.TaskLog).Proto()
			if pErr != nil {
				return pErr
			}
			return resp.Send(pl)
		})
	}

	total, err := a.m.taskLogBackend.TaskLogsCount(taskID, filters)
	if err != nil {
		return fmt.Errorf("failed to get trial count from backend: %w", err)
	}
	effectiveLimit := api.EffectiveLimit(int(req.Limit), 0, total)

	return api.NewBatchStreamProcessor(
		api.BatchRequest{Limit: effectiveLimit, Follow: req.Follow},
		fetch,
		onBatch,
		a.isTaskTerminalFunc(taskID, taskLogsTerminationDelay),
		false,
		taskLogsBatchWaitTime,
		taskLogsBatchMissWaitTime,
	).Run(ctx)
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

func (a *apiServer) TaskLogsFields(
	req *apiv1.TaskLogsFieldsRequest, resp apiv1.Determined_TaskLogsFieldsServer,
) error {
	taskID := model.TaskID(req.TaskId)
	fetch := func(lr api.BatchRequest) (api.Batch, error) {
		fields, err := a.m.taskLogBackend.TaskLogsFields(taskID)
		return api.ToBatchOfOne(fields), err
	}

	onBatch := func(b api.Batch) error {
		return b.ForEach(func(r interface{}) error {
			return resp.Send(r.(*apiv1.TaskLogsFieldsResponse))
		})
	}

	return api.NewBatchStreamProcessor(
		api.BatchRequest{Follow: req.Follow},
		fetch,
		onBatch,
		a.isTaskTerminalFunc(taskID, taskLogsTerminationDelay),
		true,
		taskLogsFieldsBatchWaitTime,
		taskLogsFieldsBatchWaitTime,
	).Run(resp.Context())
}

// isTaskTerminalFunc returns an api.TerminationCheckFn that waits for a task to finish and
// optionally, additionally, waits some buffer duration to give trials a bit to finish sending
// stuff after termination.
func (a *apiServer) isTaskTerminalFunc(taskID model.TaskID, buffer time.Duration) api.TerminationCheckFn {
	return func() (bool, error) {
		switch task, err := a.m.db.TaskByID(taskID); {
		case err != nil:
			return true, err
		case task.EndTime != nil && task.EndTime.Add(buffer).After(time.Now().UTC()):
			return true, nil
		default:
			return false, nil
		}
	}
}
