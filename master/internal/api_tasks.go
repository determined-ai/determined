package internal

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
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
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/internal/webhooks"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
	log "github.com/sirupsen/logrus"
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

func canAccessNTSCTask(ctx context.Context, curUser model.User, taskID model.TaskID) (bool, error) {
	spec, err := db.IdentifyTask(ctx, taskID)
	if errors.Is(err, db.ErrNotFound) {
		// Non NTSC case like checkpointGC case or the task just does not exist.
		// TODO(nick) eventually control access to checkpointGC.
		return true, nil
	} else if err != nil {
		return false, err
	}
	err = command.AuthZProvider.Get().CanGetNSC(
		ctx, curUser, spec.WorkspaceID)
	return !authz.IsPermissionDenied(err), err
}

func (a *apiServer) canDoActionsOnTask(
	ctx context.Context, taskID model.TaskID,
	actions ...func(context.Context, model.User, *model.Experiment) error,
) error {
	errTaskNotFound := api.NotFoundErrs("task", fmt.Sprint(taskID), true)
	t, err := a.m.db.TaskByID(taskID)
	if errors.Is(err, db.ErrNotFound) {
		return errTaskNotFound
	} else if err != nil {
		return err
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	switch t.TaskType {
	case model.TaskTypeTrial:
		isExp, exp, err := expFromTaskID(ctx, taskID)
		if !isExp {
			return fmt.Errorf("error we failed to look up an experiment "+
				"from taskID %s when we think it is a trial task", taskID)
		}
		if err != nil {
			return err
		}

		if err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp); err != nil {
			return authz.SubIfUnauthorized(err, errTaskNotFound)
		}
		for _, action := range actions {
			if err = action(ctx, *curUser, exp); err != nil {
				return status.Error(codes.PermissionDenied, err.Error())
			}
		}
	default: // NTSC case + checkpointGC.
		if ok, err := canAccessNTSCTask(ctx, *curUser, taskID); err != nil {
			if !ok || authz.IsPermissionDenied(err) {
				return errTaskNotFound
			}
			return err
		}
	}
	return nil
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
		if ok, err = canAccessNTSCTask(ctx, *curUser, model.TaskID(taskID)); err != nil {
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

func (a *apiServer) canEditAllocation(ctx context.Context, allocationID string) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	if !strings.Contains(allocationID, ".") {
		return status.Errorf(codes.InvalidArgument,
			"allocationID %s does not  contain at least '.'", allocationID)
	}

	taskID := model.AllocationID(allocationID).ToTaskID()
	isExp, exp, err := expFromTaskID(ctx, taskID)
	if err != nil {
		return err
	}
	if !isExp {
		var ok bool
		if ok, err = canAccessNTSCTask(ctx, *curUser, taskID); err != nil {
			return err
		} else if !ok {
			return api.NotFoundErrs("allocation", allocationID, true)
		}
		return nil
	}

	if err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp); err != nil {
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
	return nil, grpcutil.UnimplementedError
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

	var res []model.AcceleratorData
	err := db.Bun().NewSelect().
		ColumnExpr("alloc_acc.*").
		TableExpr("allocation_accelerators alloc_acc").
		Join("LEFT JOIN allocations alloc ON alloc_acc.allocation_id = alloc.allocation_id").
		Where("alloc.task_id = ?", req.TaskId).Scan(ctx, &res)
	if err != nil {
		return nil, fmt.Errorf("querying allocation accelerators: %w", err)
	}

	var accelerationData []*apiv1.AcceleratorData
	for _, r := range res {
		accelerationData = append(accelerationData, r.Proto())
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

func (a *apiServer) PostTaskLogs(
	ctx context.Context, req *apiv1.PostTaskLogsRequest,
) (*apiv1.PostTaskLogsResponse, error) {
	if len(req.Logs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "len logs must be greater than 0")
	}
	taskID := req.Logs[0].TaskId

	if err := a.canDoActionsOnTask(ctx, model.TaskID(taskID),
		expauth.AuthZProvider.Get().CanEditExperiment); err != nil {
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

	// TODO(ft) move this to a seperate function.
	// TODO(ft) do all logs have same taskID? Or should we just assume that they don't need to.
	// TODO(ft) do all logs stream through here? K8S I think we are planning to add log through
	// k8s api too.
	for _, l := range logs {
		if l.AgentID == nil {
			return nil, fmt.Errorf("agentID must be non nil") // TODO can we get away with this?
			// It feels kinda annoying to let our database have a possible null that theoretically
			// shouldn't happen.
		}

		regex := "testdisallow"
		if strings.Contains(l.Log, regex) { // TODO(ft) look up what regexes we care about per task.
			fmt.Println(regex)
			if err := logpattern.AddRetryOnDifferentNode(
				ctx, model.TaskID(l.TaskID), *l.AgentID, regex, l.Log,
			); err != nil {
				log.Errorf("error disallowing node") // Failing adding logs seems super bad.
			}
		}

		regex = "testdontretry"
		if strings.Contains(l.Log, regex) {
			fmt.Println(regex)
			if err := logpattern.AddDontRetry(
				ctx, model.TaskID(l.TaskID), *l.AgentID, regex, l.Log,
			); err != nil {
				log.Errorf("error disallowing node") // Failing adding logs seems super bad.
			}
		}

		regex = "testwebhook"
		if strings.Contains(l.Log, regex) {
			// TODO throw this in expconf..
			// TODO remove other thing in expcofn.
			fmt.Println(regex)
			url := "https://webhook.site/ce7c7e07-898a-4145-8ff4-84ef34f871fc"

			wt := webhooks.WebhookTypeDefault
			wt = webhooks.WebhookTypeSlack
			if err := logpattern.AddWebhookAlert(ctx,
				model.TaskID(l.TaskID), *l.AgentID, regex, l.Log, url, wt,
			); err != nil {
				log.Errorf("error disallowing node") // Failing adding logs seems super bad.
			}
		}

	}

	if err := a.m.taskLogBackend.AddTaskLogs(logs); err != nil {
		return nil, fmt.Errorf("adding task logs to task log backend: %w", err)
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
	req1 := &apiv1.GetNotebooksRequest{}
	resp1 := &apiv1.GetNotebooksResponse{}
	if err = a.ask(notebooksAddr, req1, &resp1); err != nil {
		return nil, err
	}
	for _, n := range resp1.Notebooks {
		if n.State == taskv1.State_STATE_RUNNING {
			finalResp.Notebooks++
		}
	}

	req2 := &apiv1.GetTensorboardsRequest{}
	resp2 := &apiv1.GetTensorboardsResponse{}
	if err = a.ask(tensorboardsAddr, req2, &resp2); err != nil {
		return nil, err
	}
	for _, tb := range resp2.Tensorboards {
		if tb.State == taskv1.State_STATE_RUNNING {
			finalResp.Tensorboards++
		}
	}

	req3 := &apiv1.GetCommandsRequest{}
	resp3 := &apiv1.GetCommandsResponse{}
	if err = a.ask(commandsAddr, req3, &resp3); err != nil {
		return nil, err
	}
	for _, c := range resp3.Commands {
		if c.State == taskv1.State_STATE_RUNNING {
			finalResp.Commands++
		}
	}

	req4 := &apiv1.GetShellsRequest{}
	resp4 := &apiv1.GetShellsResponse{}
	if err = a.ask(shellsAddr, req4, &resp4); err != nil {
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
	summary, err := a.m.rm.GetAllocationSummaries(a.m.system, sproto.GetAllocationSummaries{})
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

		var ok bool
		if !isExp {
			ok, err = canAccessNTSCTask(ctx, *curUser, summary[allocationID].TaskID)
		} else {
			err = expauth.AuthZProvider.Get().CanGetExperiment(ctx, *curUser, exp)
		}
		if !authz.IsPermissionDenied(err) || !ok {
			pbAllocationIDToSummary[string(allocationID)] = allocationSummary.Proto()
		} else if err != nil {
			return nil, err
		}
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
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if err = a.canDoActionsOnTask(ctx, taskID,
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
		filters = append(filters, api.Filter{
			Field:     "log",
			Operation: api.FilterOperationStringContainment,
			Values:    req.SearchText,
		})
	}
	return filters, nil
}

func (a *apiServer) TaskLogsFields(
	req *apiv1.TaskLogsFieldsRequest, resp apiv1.Determined_TaskLogsFieldsServer,
) error {
	taskID := model.TaskID(req.TaskId)

	var timeSinceLastAuth time.Time
	fetch := func(lr api.BatchRequest) (api.Batch, error) {
		if time.Now().Sub(timeSinceLastAuth) >= recheckAuthPeriod {
			if err := a.canDoActionsOnTask(resp.Context(), taskID,
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
