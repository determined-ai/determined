package command

import (
	"context"
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/internal/job/jobservice"

	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/logger"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// terminatedDuration defines the amount of time the command stays in a
// terminated state in the master before garbage collecting.
const terminatedDuration = 24 * time.Hour

// terminateForGC is an internal message indicating that the command actor
// should stop and garbage collect its state.
type terminateForGC struct{}

// queueStates are allocation states which the API and UI will show as "Queued".
var queueStates = []model.AllocationState{
	model.AllocationStatePending,
	model.AllocationStateAssigned,
}

func enrichState(state model.AllocationState) taskv1.State {
	if slices.Contains(queueStates, state) {
		return taskv1.State_STATE_QUEUED
	}
	return state.Proto()
}

func createGenericCommandActor(
	ctx *actor.Context,
	db *db.PgDB,
	rm rm.ResourceManager,
	taskID model.TaskID,
	taskType model.TaskType,
	jobID model.JobID,
	jobType model.JobType,
	spec tasks.GenericCommandSpec,
) error {
	spec.TaskType = taskType
	cmd := &command{
		db: db,
		rm: rm,

		GenericCommandSpec: spec,

		taskID:   taskID,
		taskType: taskType,
		jobType:  jobType,
		jobID:    jobID,

		logCtx: logger.Context{
			"job-id":    jobID,
			"task-id":   taskID,
			"task-type": taskType,
		},
	}

	a, _ := ctx.ActorOf(cmd.taskID, cmd)
	summaryFut := ctx.Ask(a, getSummary{})
	if err := summaryFut.Error(); err != nil {
		return errors.Wrap(err, "failed to create generic command")
	}
	// Sync with the actor, but we don't really need the summary. actor.Ping works too,
	// but this makes sure it can form some sort of useful response (ping doesn't actually
	// hit the receive block).
	summaryFut.Get()
	return nil
}

func commandFromSnapshot(
	ctx *actor.Context,
	db *db.PgDB,
	rm rm.ResourceManager,
	snapshot *CommandSnapshot,
) *command {
	taskID := snapshot.TaskID
	taskType := snapshot.Task.TaskType
	jobID := snapshot.Task.Job.JobID
	cmd := &command{
		db:             db,
		rm:             rm,
		registeredTime: snapshot.RegisteredTime,

		GenericCommandSpec: snapshot.GenericCommandSpec,

		taskID:   taskID,
		taskType: taskType,
		jobType:  snapshot.Task.Job.JobType,
		jobID:    jobID,

		logCtx: logger.Context{
			"job-id":    jobID,
			"task-id":   taskID,
			"task-type": taskType,
		},

		restored: true,
	}

	return cmd
}

func remakeCommandsByType(
	ctx *actor.Context,
	pgDB *db.PgDB,
	rm rm.ResourceManager,
	taskType model.TaskType,
) ([]*command, error) {
	snapshots := []CommandSnapshot{}

	err := db.Bun().NewSelect().Model(&snapshots).
		Relation("Allocation").
		Relation("Task").
		Relation("Task.Job").
		Where("allocation.end_time IS NULL").
		Where("allocation.state != ?", model.AllocationStateTerminated).
		Where("task.task_type = ?", taskType).
		Scan(context.TODO())
	if err != nil {
		ctx.Log().WithError(err).Warnf("failed to restore task type %s", taskType)
		return nil, err
	}

	results := []*command{}
	for i := range snapshots {
		if rm.IsReattachEnabledForRP(ctx,
			snapshots[i].GenericCommandSpec.Config.Resources.ResourcePool) {
			cmd := commandFromSnapshot(ctx, pgDB, rm, &snapshots[i])
			results = append(results, cmd)
		}
	}

	return results, nil
}

func restoreCommandsByType(
	ctx *actor.Context,
	pgDB *db.PgDB,
	rm rm.ResourceManager,
	taskType model.TaskType,
) error {
	commands, err := remakeCommandsByType(ctx, pgDB, rm, taskType)
	if err != nil {
		return err
	}

	for _, cmd := range commands {
		a, ok := ctx.ActorOf(cmd.taskID, cmd)
		if !ok {
			return fmt.Errorf("failed to recreate restored generic command actor %s", cmd.taskID)
		}

		ctx.Ask(a, actor.Ping{}).Get()
		ctx.Log().Debugf("restored generic command %s", cmd.taskID)
	}

	return nil
}

func tryRestoreCommandsByType(
	ctx *actor.Context,
	pgDB *db.PgDB,
	rm rm.ResourceManager,
	taskType model.TaskType,
) {
	if rm.IsReattachEnabled(ctx) {
		err := restoreCommandsByType(ctx, pgDB, rm, taskType)
		if err != nil {
			ctx.Log().WithError(err).Warnf("failed to restoreCommandsByType: %s", taskType)
		}
	}
}

// command is executed in a containerized environment on a Determined cluster.
type command struct {
	db *db.PgDB
	rm rm.ResourceManager

	tasks.GenericCommandSpec

	registeredTime time.Time
	taskID         model.TaskID
	taskType       model.TaskType
	jobType        model.JobType
	jobID          model.JobID
	allocationID   model.AllocationID
	lastState      task.AllocationState
	exitStatus     *task.AllocationExited
	restored       bool

	logCtx logger.Context
}

// Receive implements the actor.Actor interface.
func (c *command) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabels(c.logCtx)
		c.allocationID = model.AllocationID(fmt.Sprintf("%s.%d", c.taskID, 1))
		if !c.restored {
			c.registeredTime = ctx.Self().RegisteredTime().Truncate(time.Millisecond)
			if err := c.db.AddJob(&model.Job{
				JobID:   c.jobID,
				JobType: c.jobType,
				OwnerID: &c.Base.Owner.ID,
			}); err != nil {
				return errors.Wrapf(err, "persisting job %v", c.taskID)
			}

			if err := c.db.AddTask(&model.Task{
				TaskID:     c.taskID,
				TaskType:   c.taskType,
				StartTime:  c.registeredTime,
				JobID:      &c.jobID,
				LogVersion: model.CurrentTaskLogVersion,
			}); err != nil {
				return errors.Wrapf(err, "persisting task %v", c.taskID)
			}
		}

		priority := c.Config.Resources.Priority
		if priority != nil {
			if err := c.setPriority(ctx, *priority, true); err != nil {
				return errors.Wrapf(err, "setting priority of task %v", c.taskID)
			}
		}

		var idleWatcherConfig *sproto.IdleTimeoutConfig
		if c.Config.IdleTimeout != nil && (c.WatchProxyIdleTimeout || c.WatchRunnerIdleTimeout) {
			idleWatcherConfig = &sproto.IdleTimeoutConfig{
				ServiceID:       string(c.taskID),
				UseProxyState:   c.WatchProxyIdleTimeout,
				UseRunnerState:  c.WatchRunnerIdleTimeout,
				TimeoutDuration: time.Duration(*c.Config.IdleTimeout),
				Debug:           c.Config.Debug,
			}
		}

		err := task.DefaultService.StartAllocation(c.logCtx, sproto.AllocateRequest{
			AllocationID:      c.allocationID,
			TaskID:            c.taskID,
			JobID:             c.jobID,
			JobSubmissionTime: c.registeredTime,
			IsUserVisible:     true,
			Name:              c.Config.Description,
			Group:             ctx.Self(),

			SlotsNeeded:  c.Config.Resources.Slots,
			ResourcePool: c.Config.Resources.ResourcePool,
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},

			ProxyPorts:  sproto.NewProxyPortConfig(c.GenericCommandSpec.ProxyPorts(), c.taskID),
			IdleTimeout: idleWatcherConfig,
			Restore:     c.restored,
			ProxyTLS:    c.TaskType == model.TaskTypeNotebook,
		}, c.db, c.rm, c.GenericCommandSpec, ctx.Self().System(), func(ae *task.AllocationExited) {
			ctx.Tell(ctx.Self(), ae)
		})
		if err != nil {
			return err
		}

		jobservice.Default.RegisterJob(c.jobID, ctx.Self())

		if err := c.persist(); err != nil {
			ctx.Log().WithError(err).Warnf("command persist failure")
		}
	case sproto.GetJob:
		ctx.Respond(c.toV1Job())

	case actor.PostStop:
		if c.exitStatus == nil {
			if err := c.db.CompleteTask(c.taskID, time.Now().UTC()); err != nil {
				ctx.Log().WithError(err).Error("marking task complete")
			}
		}
		jobservice.Default.UnregisterJob(c.jobID)
		if err := c.db.DeleteUserSessionByToken(c.GenericCommandSpec.Base.UserSessionToken); err != nil {
			ctx.Log().WithError(err).Errorf(
				"failure to delete user session for task: %v", c.taskID)
		}
	case *task.AllocationExited:
		c.exitStatus = msg
		if err := c.db.CompleteTask(c.taskID, time.Now().UTC()); err != nil {
			ctx.Log().WithError(err).Error("marking task complete")
		}
		if err := c.db.DeleteUserSessionByToken(c.GenericCommandSpec.Base.UserSessionToken); err != nil {
			ctx.Log().WithError(err).Errorf(
				"failure to delete user session for task: %v", c.taskID)
		}
		actors.NotifyAfter(ctx, terminatedDuration, terminateForGC{})
	case getSummary:
		if msg.userFilter == "" || c.Base.Owner.Username == msg.userFilter {
			ctx.Respond(c.summary(ctx))
		}

	case *notebookv1.Notebook:
		ctx.Respond(c.toNotebook(ctx))

	case *apiv1.GetNotebookRequest:
		ctx.Respond(&apiv1.GetNotebookResponse{
			Notebook: c.toNotebook(ctx),
			Config:   protoutils.ToStruct(c.Config),
		})
	case *apiv1.KillNotebookRequest:
		// TODO(Brad): Do the same thing to allocations that we are doing to RMs.
		err := task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
		if err != nil {
			ctx.Log().WithError(err).Warn("failed to kill allocation")
		}
		ctx.Respond(&apiv1.KillNotebookResponse{Notebook: c.toNotebook(ctx)})
	case *apiv1.SetNotebookPriorityRequest:
		err := c.setPriority(ctx, int(msg.Priority), true)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(&apiv1.SetNotebookPriorityResponse{Notebook: c.toNotebook(ctx)})

	case *commandv1.Command:
		ctx.Respond(c.toCommand(ctx))

	case *apiv1.GetCommandRequest:
		ctx.Respond(&apiv1.GetCommandResponse{
			Command: c.toCommand(ctx),
			Config:  protoutils.ToStruct(c.Config),
		})

	case *apiv1.KillCommandRequest:
		err := task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
		if err != nil {
			ctx.Log().WithError(err).Warn("failed to kill allocation")
		}
		ctx.Respond(&apiv1.KillCommandResponse{Command: c.toCommand(ctx)})

	case *apiv1.SetCommandPriorityRequest:
		err := c.setPriority(ctx, int(msg.Priority), true)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(&apiv1.SetCommandPriorityResponse{Command: c.toCommand(ctx)})

	case *shellv1.Shell:
		ctx.Respond(c.toShell(ctx))

	case *apiv1.GetShellRequest:
		ctx.Respond(&apiv1.GetShellResponse{
			Shell:  c.toShell(ctx),
			Config: protoutils.ToStruct(c.Config),
		})

	case *apiv1.KillShellRequest:
		err := task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
		if err != nil {
			ctx.Log().WithError(err).Warn("failed to kill allocation")
		}
		ctx.Respond(&apiv1.KillShellResponse{Shell: c.toShell(ctx)})

	case *apiv1.SetShellPriorityRequest:
		err := c.setPriority(ctx, int(msg.Priority), true)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(&apiv1.SetShellPriorityResponse{Shell: c.toShell(ctx)})

	case *tensorboardv1.Tensorboard:
		ctx.Respond(c.toTensorboard(ctx))

	case *apiv1.GetTensorboardRequest:
		ctx.Respond(&apiv1.GetTensorboardResponse{
			Tensorboard: c.toTensorboard(ctx),
			Config:      protoutils.ToStruct(c.Config),
		})

	case *apiv1.KillTensorboardRequest:
		err := task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
		if err != nil {
			ctx.Log().WithError(err).Warn("failed to kill allocation")
		}
		ctx.Respond(&apiv1.KillTensorboardResponse{Tensorboard: c.toTensorboard(ctx)})

	case *apiv1.SetTensorboardPriorityRequest:
		err := c.setPriority(ctx, int(msg.Priority), true)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(&apiv1.SetTensorboardPriorityResponse{Tensorboard: c.toTensorboard(ctx)})

	case *apiv1.DeleteWorkspaceRequest:
		if c.Metadata.WorkspaceID == model.AccessScopeID(msg.Id) {
			err := task.DefaultService.Signal(
				c.allocationID,
				task.KillAllocation,
				"user requested workspace delete",
			)
			if err != nil {
				ctx.Log().WithError(err).Warn("failed to kill allocation while deleting workspace")
			}
		}

	case sproto.NotifyRMPriorityChange:
		ctx.Respond(c.setPriority(ctx, msg.Priority, false))

	case sproto.ContainerLog:

	case terminateForGC:
		ctx.Self().Stop()

	case sproto.SetGroupWeight:
		err := c.setWeight(ctx, msg.Weight)
		if err != nil {
			ctx.Log().WithError(err).Info("setting command job weight")
		}
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}

	case sproto.SetGroupPriority:
		err := c.setPriority(ctx, msg.Priority, true)
		if err != nil {
			ctx.Log().WithError(err).Info("setting command job priority")
		}
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}

	case sproto.RegisterJobPosition:
		err := c.db.UpdateJobPosition(msg.JobID, msg.JobPosition)
		if err != nil {
			ctx.Log().WithError(err).Errorf("persisting position for job %s failed", msg.JobID)
		}

	case sproto.SetResourcePool:
		ctx.Respond(fmt.Errorf("setting resource pool for job type %s is not supported", c.jobType))

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (c *command) setPriority(ctx *actor.Context, priority int, forward bool) error {
	if forward {
		switch err := c.rm.SetGroupPriority(ctx, sproto.SetGroupPriority{
			Priority: priority,
			Handler:  ctx.Self(),
		}).(type) {
		case nil:
		case rmerrors.ErrUnsupported:
			ctx.Log().WithError(err).Debug("ignoring unsupported call to set group priority")
		default:
			return fmt.Errorf("setting group priority for command: %w", err)
		}
	}

	c.Config.Resources.Priority = &priority

	return nil
}

func (c *command) setWeight(ctx *actor.Context, weight float64) error {
	switch err := c.rm.SetGroupWeight(ctx, sproto.SetGroupWeight{
		Weight:  weight,
		Handler: ctx.Self(),
	}).(type) {
	case nil:
	case rmerrors.ErrUnsupported:
		ctx.Log().WithError(err).Debug("ignoring unsupported call to set group weight")
	default:
		return fmt.Errorf("setting group weight for command: %w", err)
	}

	c.Config.Resources.Weight = weight
	return nil
}

func (c *command) stringID() string {
	return c.taskID.String()
}

func (c *command) serviceAddress() string {
	return fmt.Sprintf("/proxy/%s/", c.taskID)
}

func (c *command) toNotebook(ctx *actor.Context) *notebookv1.Notebook {
	allo := c.refreshAllocationState(ctx)
	state := enrichState(allo.State)

	return &notebookv1.Notebook{
		Id:             c.stringID(),
		State:          state,
		Description:    c.Config.Description,
		Container:      allo.SingleContainer().ToProto(),
		ServiceAddress: c.serviceAddress(),
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}

func (c *command) toCommand(ctx *actor.Context) *commandv1.Command {
	allo := c.refreshAllocationState(ctx)
	state := enrichState(allo.State)
	return &commandv1.Command{
		Id:           c.stringID(),
		State:        state,
		Description:  c.Config.Description,
		Container:    allo.SingleContainer().ToProto(),
		StartTime:    protoutils.ToTimestamp(c.registeredTime),
		Username:     c.Base.Owner.Username,
		UserId:       int32(c.Base.Owner.ID),
		DisplayName:  c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool: c.Config.Resources.ResourcePool,
		ExitStatus:   c.exitStatus.String(),
		JobId:        c.jobID.String(),
		WorkspaceId:  int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}

func (c *command) toShell(ctx *actor.Context) *shellv1.Shell {
	allo := c.refreshAllocationState(ctx)
	state := enrichState(allo.State)
	return &shellv1.Shell{
		Id:             c.stringID(),
		State:          state,
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Container:      allo.SingleContainer().ToProto(),
		PrivateKey:     *c.Metadata.PrivateKey,
		PublicKey:      *c.Metadata.PublicKey,
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		Addresses:      toProto(allo.SingleContainerAddresses()),
		AgentUserGroup: protoutils.ToStruct(c.Base.AgentUserGroup),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}

func (c *command) toTensorboard(ctx *actor.Context) *tensorboardv1.Tensorboard {
	allo := c.refreshAllocationState(ctx)
	state := enrichState(allo.State)
	return &tensorboardv1.Tensorboard{
		Id:             c.stringID(),
		State:          state,
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Container:      allo.SingleContainer().ToProto(),
		ServiceAddress: c.serviceAddress(),
		ExperimentIds:  c.Metadata.ExperimentIDs,
		TrialIds:       c.Metadata.TrialIDs,
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}

// Refresh our view of the allocation state. If the allocation has sent us an exit status,
// we don't ask for a refresh because it won't respond. Otherwise, ask with a timeout
// since there is another ask in the opposite direction, and even though it's probably
// 1 in a million runs, we don't want to deadlock.
func (c *command) refreshAllocationState(ctx *actor.Context) task.AllocationState {
	if c.exitStatus != nil {
		return c.exitStatus.FinalState
	}

	state, err := task.DefaultService.State(c.allocationID)
	if err != nil {
		ctx.Log().WithError(err).Warn("refreshing allocation state")
	} else {
		c.lastState = state
	}
	return c.lastState
}

func toProto(as []cproto.Address) []*structpb.Struct {
	res := make([]*structpb.Struct, 0, len(as))
	for _, a := range as {
		res = append(res, protoutils.ToStruct(a))
	}
	return res
}

func (c *command) toV1Job() *jobv1.Job {
	j := jobv1.Job{
		JobId:          c.jobID.String(),
		EntityId:       string(c.taskID),
		Type:           c.jobType.Proto(),
		SubmissionTime: timestamppb.New(c.registeredTime),
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		Weight:         c.Config.Resources.Weight,
		Name:           c.Config.Description,
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}

	j.IsPreemptible = false
	j.Priority = int32(config.ReadPriority(j.ResourcePool, &c.Config))
	j.Weight = config.ReadWeight(j.ResourcePool, &c.Config)

	j.ResourcePool = c.Config.Resources.ResourcePool

	return &j
}

func (c *command) snapshot() *CommandSnapshot {
	res := CommandSnapshot{
		TaskID:             c.taskID,
		RegisteredTime:     c.registeredTime,
		AllocationID:       c.allocationID,
		GenericCommandSpec: c.GenericCommandSpec,
	}
	return &res
}

func (c *command) persist() error {
	snapshot := c.snapshot()
	_, err := db.Bun().NewInsert().Model(snapshot).
		On("CONFLICT (task_id) DO UPDATE").
		Exec(context.TODO())
	return err
}
