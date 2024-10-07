package internal

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	expauth "github.com/determined-ai/determined/master/internal/experiment"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/internal/webhooks"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

const (
	taskLogsChanBuffer = 5
	taskLogsBatchSize  = 1000
)

var (
	taskReadyCheckLogs = "/run/determined/check_ready_logs.py"

	taskLogsBatchMissWaitTime   = time.Second
	taskLogsFieldsBatchWaitTime = 5 * time.Second
)

func expFromTaskID(
	ctx context.Context, taskID model.TaskID,
) (isExperiment bool, exp *model.Experiment, err error) {
	expID, err := experimentIDFromTrialTaskID(taskID)
	if errors.Is(err, errIsNotTrialTaskID) {
		return false, nil, nil
	} else if err != nil {
		return false, nil, err
	}

	exp, err = db.ExperimentByID(ctx, expID)
	if err != nil {
		return false, nil, err
	}
	return true, exp, nil
}

func canAccessNTSCTask(
	ctx context.Context, curUser model.User, taskID model.TaskID,
) (bool, model.AccessScopeID, error) {
	spec, err := command.IdentifyTask(ctx, taskID)
	if errors.Is(err, db.ErrNotFound) {
		// Non NTSC case like checkpointGC case or the task just does not exist.
		// TODO(nick) eventually control access to checkpointGC.
		return true, spec.WorkspaceID, nil
	} else if err != nil {
		return false, spec.WorkspaceID, err
	}
	err = command.AuthZProvider.Get().CanGetNSC(
		ctx, curUser, spec.WorkspaceID)
	return !authz.IsPermissionDenied(err), spec.WorkspaceID, err
}

func (a *apiServer) canDoActionsOnTask(
	ctx context.Context, taskID model.TaskID,
	actions ...func(context.Context, model.User, *model.Experiment) error,
) (*model.AccessScopeID, *int, error) {
	errTaskNotFound := api.NotFoundErrs("task", fmt.Sprint(taskID), true)
	t, err := db.TaskByID(ctx, taskID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, nil, errTaskNotFound
	} else if err != nil {
		return nil, nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, nil, err
	}

	switch t.TaskType {
	case model.TaskTypeTrial:
		isExp, exp, err := expFromTaskID(ctx, taskID)
		if !isExp {
			return nil, nil, fmt.Errorf("error we failed to look up an experiment "+
				"from taskID %s when we think it is a trial task", taskID)
		}
		if err != nil {
			return nil, nil, err
		}

		if err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp); err != nil {
			return nil, nil, authz.SubIfUnauthorized(err, errTaskNotFound)
		}
		for _, action := range actions {
			if err = action(ctx, *curUser, exp); err != nil {
				return nil, nil, status.Error(codes.PermissionDenied, err.Error())
			}
		}
		workspaceID, err := expauth.GetWorkspaceFromExperiment(ctx, exp)
		if err != nil {
			return nil, nil, err
		}
		return ptrs.Ptr(model.AccessScopeID(workspaceID)), ptrs.Ptr(exp.ID), nil
	default: // NTSC case + checkpointGC.
		ok, workspaceID, err := canAccessNTSCTask(ctx, *curUser, taskID)
		if err != nil {
			if !ok || authz.IsPermissionDenied(err) {
				return nil, nil, errTaskNotFound
			}
			return nil, nil, err
		}
		// When error is nil, workspaceID is guaranteed not nil
		return &workspaceID, nil, nil
	}
}

func (a *apiServer) canGetTaskAcceleration(ctx context.Context, taskID string) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}
	isExp, exp, err := expFromTaskID(ctx, model.TaskID(taskID))
	if err != nil {
		return err
	}
	if !isExp {
		var ok bool
		if ok, _, err = canAccessNTSCTask(ctx, *curUser, model.TaskID(taskID)); err != nil {
			return err
		} else if !ok {
			return api.NotFoundErrs("task", taskID, true)
		}
		return nil
	}

	if err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp); err != nil {
		return authz.SubIfUnauthorized(err, api.NotFoundErrs("task", taskID, true))
	}

	return nil
}

func (a *apiServer) canGetAllocation(ctx context.Context, allocationID string) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	if !strings.Contains(allocationID, ".") {
		return status.Errorf(codes.InvalidArgument, "allocationID %s does not contain at least ','", allocationID)
	}

	taskID := model.AllocationID(allocationID).ToTaskID()
	isExp, exp, err := expFromTaskID(ctx, taskID)
	if err != nil {
		return err
	}
	if !isExp {
		var ok bool
		if ok, _, err = canAccessNTSCTask(ctx, *curUser, taskID); err != nil {
			return err
		} else if !ok {
			return api.NotFoundErrs("allocation", allocationID, true)
		}
		return nil
	}

	if err = expauth.AuthZProvider.Get().CanGetExperimentArtifacts(ctx, *curUser, exp); err != nil {
		return authz.SubIfUnauthorized(err, api.NotFoundErrs("allocation", allocationID, true))
	}

	return nil
}

func (a *apiServer) canEditAllocation(ctx context.Context, allocationID string) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	if !strings.Contains(allocationID, ".") {
		return status.Errorf(codes.InvalidArgument,
			"allocationID %s does not contain at least '.'", allocationID)
	}

	taskID := model.AllocationID(allocationID).ToTaskID()
	isExp, exp, err := expFromTaskID(ctx, taskID)
	if err != nil {
		return err
	}
	if !isExp {
		var ok bool
		if ok, _, err = canAccessNTSCTask(ctx, *curUser, taskID); err != nil {
			return err
		} else if !ok {
			return api.NotFoundErrs("allocation", allocationID, true)
		}
		return nil
	}

	if err = expauth.AuthZProvider.Get().CanGetExperimentArtifacts(ctx, *curUser, exp); err != nil {
		return authz.SubIfUnauthorized(err, api.NotFoundErrs("allocation", allocationID, true))
	}
	if err = expauth.AuthZProvider.Get().CanEditExperiment(ctx, *curUser, exp); err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}

	return nil
}

func (a *apiServer) AllocationReady(
	ctx context.Context, req *apiv1.AllocationReadyRequest,
) (*apiv1.AllocationReadyResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	err := task.DefaultService.SetReady(ctx, model.AllocationID(req.AllocationId))
	if err != nil {
		return nil, err
	}
	return &apiv1.AllocationReadyResponse{}, nil
}

func (a *apiServer) AllocationWaiting(
	ctx context.Context, req *apiv1.AllocationWaitingRequest,
) (*apiv1.AllocationWaitingResponse, error) {
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	err := task.DefaultService.SetWaiting(ctx, model.AllocationID(req.AllocationId))
	if err != nil {
		return nil, err
	}
	return &apiv1.AllocationWaitingResponse{}, nil
}

func (a *apiServer) AllocationAllGather(
	ctx context.Context, req *apiv1.AllocationAllGatherRequest,
) (*apiv1.AllocationAllGatherResponse, error) {
	if req.AllocationId == "" {
		return nil, status.Error(codes.InvalidArgument, "allocation ID missing")
	}
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(req.RequestUuid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	data, err := task.DefaultService.AllGather(
		ctx,
		model.AllocationID(req.AllocationId),
		id,
		int(req.NumPeers),
		req.Data,
	)
	if err != nil {
		return nil, err
	}

	var out []*structpb.Struct
	for _, d := range data {
		out = append(out, d.(*structpb.Struct))
	}
	return &apiv1.AllocationAllGatherResponse{Data: out}, nil
}

func (a *apiServer) GetAllocation(
	ctx context.Context,
	req *apiv1.GetAllocationRequest,
) (*apiv1.GetAllocationResponse, error) {
	if req.AllocationId == "" {
		return nil, status.Error(codes.InvalidArgument, "allocation ID missing")
	}

	if err := a.canGetAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	allocation, err := task.DefaultService.GetAllocation(ctx, req.AllocationId)
	if err != nil {
		return nil, fmt.Errorf("querying allocation %s: %w", req.AllocationId, err)
	}
	return &apiv1.GetAllocationResponse{
		Allocation: allocation.Proto(),
	}, nil
}

func (a *apiServer) PostAllocationProxyAddress(
	ctx context.Context, req *apiv1.PostAllocationProxyAddressRequest,
) (*apiv1.PostAllocationProxyAddressResponse, error) {
	if req.AllocationId == "" {
		return nil, status.Error(codes.InvalidArgument, "allocation ID missing")
	}
	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	err := task.DefaultService.SetProxyAddress(
		ctx,
		model.AllocationID(req.AllocationId),
		req.ProxyAddress,
	)
	if err != nil {
		return nil, err
	}
	return &apiv1.PostAllocationProxyAddressResponse{}, nil
}

func (a *apiServer) GetTaskAcceleratorData(
	ctx context.Context,
	req *apiv1.GetTaskAcceleratorDataRequest,
) (*apiv1.GetTaskAcceleratorDataResponse, error) {
	if req.TaskId == "" {
		return nil, status.Error(codes.InvalidArgument, "task ID missing")
	}

	if err := a.canGetTaskAcceleration(ctx, req.TaskId); err != nil {
		return nil, err
	}

	res := []struct {
		model.AcceleratorData
		ResourcePool string
	}{}

	err := db.Bun().NewSelect().ColumnExpr("alloc_acc.*").
		Column("resource_pool").
		TableExpr("allocation_accelerators alloc_acc").
		Join("LEFT JOIN allocations alloc ON alloc_acc.allocation_id = alloc.allocation_id").
		Order("start_time DESC").
		Where("alloc.task_id = ?", req.TaskId).Scan(ctx, &res)
	if err != nil {
		return nil, fmt.Errorf("querying allocation accelerators: %w", err)
	}

	var accelerationData []*apiv1.AcceleratorData
	for _, r := range res {
		accelerationData = append(accelerationData, &apiv1.AcceleratorData{
			ContainerId:      r.ContainerID,
			AllocationId:     string(r.AllocationID),
			NodeName:         r.NodeName,
			AcceleratorType:  r.AcceleratorType,
			AcceleratorUuids: r.AcceleratorUuids,
			ResourcePool:     r.ResourcePool,
		})
	}
	return &apiv1.GetTaskAcceleratorDataResponse{AcceleratorData: accelerationData}, nil
}

func (a *apiServer) PostAllocationAcceleratorData(
	ctx context.Context,
	req *apiv1.PostAllocationAcceleratorDataRequest,
) (*apiv1.PostAllocationAcceleratorDataResponse, error) {
	if req.AllocationId == "" {
		return nil, status.Error(codes.InvalidArgument, "allocation ID missing")
	}

	if err := a.canEditAllocation(ctx, req.AllocationId); err != nil {
		return nil, err
	}

	accData := &model.AcceleratorData{
		ContainerID:      req.AcceleratorData.ContainerId,
		AllocationID:     model.AllocationID(req.AllocationId),
		NodeName:         req.AcceleratorData.NodeName,
		AcceleratorType:  req.AcceleratorData.AcceleratorType,
		AcceleratorUuids: req.AcceleratorData.AcceleratorUuids,
	}
	err := task.DefaultService.SetAcceleratorData(
		ctx,
		*accData,
	)
	if err != nil {
		return nil, err
	}

	return &apiv1.PostAllocationAcceleratorDataResponse{}, nil
}

// TaskLogBackend is an interface task log backends, such as elastic or postgres,
// must support to provide the features surfaced in our API.
type TaskLogBackend interface {
	TaskLogs(
		taskID model.TaskID, limit int, filters []api.Filter, order apiv1.OrderBy, state interface{},
	) ([]*model.TaskLog, interface{}, error)
	AddTaskLogs([]*model.TaskLog) error
	TaskLogsCount(taskID model.TaskID, filters []api.Filter) (int, error)
	TaskLogsFields(taskID model.TaskID) (*apiv1.TaskLogsFieldsResponse, error)
	DeleteTaskLogs(taskIDs []model.TaskID) error
	// MaxTerminationDelay is the max delay before a consumer can be sure all logs have been
	// recevied. A better interface may be an interface for streaming, rather than helper
	// interfaces to aid streaming, but it's not bad enough to motivate changing it.
	MaxTerminationDelay() time.Duration
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

	ctx, cancel := context.WithCancel(resp.Context())
	defer cancel()

	res := make(chan api.BatchResult, taskLogsChanBuffer)
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

func (a *apiServer) monitor(ctx context.Context, taskID model.TaskID, logs []*model.TaskLog) error {
	isExp, exp, err := expFromTaskID(ctx, taskID)
	if err != nil {
		return err
	}
	if !isExp {
		return nil
	}

	policies, err := db.ActiveLogPolicies(ctx, exp.ID)
	if err != nil {
		return err
	}

	if err := logpattern.Monitor(ctx, taskID, logs, policies); err != nil {
		return err
	}

	return nil
}

func (a *apiServer) PostTaskLogs(
	ctx context.Context, req *apiv1.PostTaskLogsRequest,
) (*apiv1.PostTaskLogsResponse, error) {
	if len(req.Logs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "len logs must be greater than 0")
	}
	taskID := req.Logs[0].TaskId

	workspaceID, expID, err := a.canDoActionsOnTask(ctx, model.TaskID(taskID),
		expauth.AuthZProvider.Get().CanEditExperiment)
	if err != nil {
		return nil, err
	}

	logs := make([]*model.TaskLog, len(req.Logs))
	for i := range req.Logs {
		if req.Logs[i].Id != nil {
			return nil, status.Errorf(codes.InvalidArgument,
				"ID must be nil on logs got %d instead", *req.Logs[i].Id)
		}
		if req.Logs[i].TaskId != taskID {
			// There isn't a hard reason for this requirement other than we would have to RBAC
			// against all provided taskIDs. This usecase seems pretty unlikely.
			return nil, status.Errorf(codes.InvalidArgument,
				"can only post logs of a single taskID per task log request got '%s' and '%s'",
				taskID, req.Logs[i].TaskId)
		}

		logs[i] = model.TaskLogFromProto(req.Logs[i])
	}

	if err := a.m.taskLogBackend.AddTaskLogs(logs); err != nil {
		return nil, fmt.Errorf("adding task logs to task log backend: %w", err)
	}

	switch err := webhooks.ScanLogs(ctx, logs, *workspaceID, expID); {
	case err != nil && errors.Is(err, context.Canceled):
		return nil, err
	case err != nil:
		log.Errorf("scanning logs for webhook triggers: %v", err)
	}

	switch err := a.monitor(ctx, model.TaskID(taskID), logs); {
	case err != nil && errors.Is(err, context.Canceled):
		return nil, err
	case err != nil:
		log.Errorf("monitor logs against log pattern policies: %s", err)
	}

	return &apiv1.PostTaskLogsResponse{}, nil
}

func (a *apiServer) GetActiveTasksCount(
	ctx context.Context, req *apiv1.GetActiveTasksCountRequest,
) (resp *apiv1.GetActiveTasksCountResponse, err error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	if err = command.AuthZProvider.Get().CanGetActiveTasksCount(ctx, *curUser); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	finalResp := &apiv1.GetActiveTasksCountResponse{}

	resp1, err := command.DefaultCmdService.GetNotebooks(&apiv1.GetNotebooksRequest{})
	if err != nil {
		return nil, err
	}
	for _, n := range resp1.Notebooks {
		if n.State == taskv1.State_STATE_RUNNING {
			finalResp.Notebooks++
		}
	}

	resp2, err := command.DefaultCmdService.GetTensorboards(&apiv1.GetTensorboardsRequest{})
	if err != nil {
		return nil, err
	}
	for _, tb := range resp2.Tensorboards {
		if tb.State == taskv1.State_STATE_RUNNING {
			finalResp.Tensorboards++
		}
	}

	resp3, err := command.DefaultCmdService.GetCommands(&apiv1.GetCommandsRequest{})
	if err != nil {
		return nil, err
	}
	for _, c := range resp3.Commands {
		if c.State == taskv1.State_STATE_RUNNING {
			finalResp.Commands++
		}
	}

	resp4, err := command.DefaultCmdService.GetShells(&apiv1.GetShellsRequest{})
	if err != nil {
		return nil, err
	}
	for _, s := range resp4.Shells {
		if s.State == taskv1.State_STATE_RUNNING {
			finalResp.Shells++
		}
	}

	return finalResp, err
}

func (a *apiServer) GetTasks(
	ctx context.Context, req *apiv1.GetTasksRequest,
) (resp *apiv1.GetTasksResponse, err error) {
	summary, err := a.m.rm.GetAllocationSummaries()
	if err != nil {
		return nil, err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	pbAllocationIDToSummary := make(map[string]*taskv1.AllocationSummary)
	for allocationID, allocationSummary := range summary {
		isExp, exp, err := expFromTaskID(ctx, allocationSummary.TaskID)
		if err != nil {
			return nil, err
		}

		if !isExp {
			_, _, err = canAccessNTSCTask(ctx, *curUser, summary[allocationID].TaskID)
		} else {
			err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp)
		}
		if authz.IsPermissionDenied(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		pbAllocationIDToSummary[string(allocationID)] = allocationSummary.Proto()
	}

	return &apiv1.GetTasksResponse{AllocationIdToSummary: pbAllocationIDToSummary}, nil
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
	var timeSinceLastAuth time.Time
	fetch := func(r api.BatchRequest) (api.Batch, error) {
		if time.Since(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err = a.canDoActionsOnTask(ctx, taskID,
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return nil, err
			}

			timeSinceLastAuth = time.Now()
		}

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
		nil,
		&taskLogsBatchMissWaitTime,
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

	// Allow a value in a list of numbers, or a NULL represented as -1.
	addNullInclusiveFilter := func(field string, values []int32) {
		if values == nil || !slices.Contains(values, -1) {
			addInFilter(field, values, len(values))
			return
		}
		filters = append(filters, api.Filter{
			Field:     field,
			Operation: api.FilterOperationInOrNull,
			Values:    values,
		})
	}

	addNullInclusiveFilter("rank_id", req.RankIds)

	addInFilter("allocation_id", req.AllocationIds, len(req.AllocationIds))
	addInFilter("agent_id", req.AgentIds, len(req.AgentIds))
	addInFilter("container_id", req.ContainerIds, len(req.ContainerIds))
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

	if req.SearchText != "" {
		if req.EnableRegex {
			filters = append(filters, api.Filter{
				Field:     "log",
				Operation: api.FilterOperationRegexContainment,
				Values:    req.SearchText,
			})
		} else {
			filters = append(filters, api.Filter{
				Field:     "log",
				Operation: api.FilterOperationStringContainment,
				Values:    req.SearchText,
			})
		}
	}
	return filters, nil
}

func (a *apiServer) TaskLogsFields(
	req *apiv1.TaskLogsFieldsRequest, resp apiv1.Determined_TaskLogsFieldsServer,
) error {
	taskID := model.TaskID(req.TaskId)

	var timeSinceLastAuth time.Time
	fetch := func(lr api.BatchRequest) (api.Batch, error) {
		if time.Since(timeSinceLastAuth) >= recheckAuthPeriod {
			if _, _, err := a.canDoActionsOnTask(resp.Context(), taskID,
				expauth.AuthZProvider.Get().CanGetExperimentArtifacts); err != nil {
				return nil, err
			}

			timeSinceLastAuth = time.Now()
		}

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
		&taskLogsFieldsBatchWaitTime,
		&taskLogsFieldsBatchWaitTime,
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
		switch task, err := db.TaskByID(context.TODO(), taskID); {
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

func zipBatches(res1, res2 chan api.BatchResult, z func(api.Batch, api.Batch) error) error {
	var err *multierror.Error
	for {
		b1, ok := <-res1
		switch {
		case !ok:
			return err.ErrorOrNil()
		case b1.Err() != nil:
			// Noting the failure but not exiting here will cause us to wait for the downstream
			// processor to fail from its error or continue.
			err = multierror.Append(err, b1.Err())
			continue
		}

		b2, ok := <-res2
		switch {
		case !ok:
			return err.ErrorOrNil()
		case b2.Err() != nil:
			// Noting the failure but not exiting here will cause us to wait for the downstream
			// processor to fail from its error or continue.
			err = multierror.Append(err, b2.Err())
			continue
		}

		if zErr := z(b1.Batch(), b2.Batch()); zErr != nil {
			// Since this is our failure, we fail and return. This should cause upstream
			// processses and cause downstream senders to cancel.
			return zErr
		}
	}
}
