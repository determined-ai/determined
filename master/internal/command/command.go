package command

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/logger"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
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
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// terminatedDuration defines the amount of time the command stays in a
// terminated state in the master before garbage collecting.
const terminatedDuration = 24 * time.Hour

// terminateForGC is an internal message indicating that the command actor
// should stop and garbage collect its state.
type terminateForGC struct{}

func createGenericCommandActor(
	ctx *actor.Context,
	db *db.PgDB,
	rm rm.ResourceManager,
	taskLogger *task.Logger,
	taskID model.TaskID,
	taskType model.TaskType,
	jobID model.JobID,
	jobType model.JobType,
	spec tasks.GenericCommandSpec,
) error {
	spec.TaskType = taskType
	cmd := &command{
		db:         db,
		rm:         rm,
		taskLogger: taskLogger,

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
	taskLogger *task.Logger,
	snapshot *CommandSnapshot,
) *command {
	taskID := snapshot.TaskID
	taskType := snapshot.Task.TaskType
	jobID := snapshot.Task.Job.JobID
	cmd := &command{
		db:         db,
		rm:         rm,
		taskLogger: taskLogger,

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
	taskLogger *task.Logger,
	taskType model.TaskType,
) ([]*command, error) {
	snapshots := []CommandSnapshot{}

	err := db.Bun().NewSelect().Model(&snapshots).
		Relation("Allocation").
		Relation("Task").
		Relation("Task.Job").
		Where("allocation.end_time IS NULL").
		Where("task.task_type = ?", taskType).
		Scan(context.TODO())
	if err != nil {
		ctx.Log().WithError(err).Warnf("failed to restore task type %s", taskType)
		return nil, err
	}

	results := []*command{}
	for i := range snapshots {
		cmd := commandFromSnapshot(ctx, pgDB, rm, taskLogger, &snapshots[i])
		results = append(results, cmd)
	}

	return results, nil
}

func restoreCommandsByType(
	ctx *actor.Context,
	pgDB *db.PgDB,
	rm rm.ResourceManager,
	taskLogger *task.Logger,
	taskType model.TaskType,
) error {
	commands, err := remakeCommandsByType(ctx, pgDB, rm, taskLogger, taskType)
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
	taskLogger *task.Logger,
	taskType model.TaskType,
) {
	if config.IsReattachEnabled() {
		err := restoreCommandsByType(ctx, pgDB, rm, taskLogger, taskType)
		if err != nil {
			ctx.Log().WithError(err).Warnf("failed to restoreCommandsByType: %s", taskType)
		}
	}
}

// command is executed in a containerized environment on a Determined cluster.
type command struct {
	db          *db.PgDB
	rm          rm.ResourceManager
	eventStream *actor.Ref
	taskLogger  *task.Logger

	tasks.GenericCommandSpec

	registeredTime time.Time
	taskID         model.TaskID
	taskType       model.TaskType
	jobType        model.JobType
	jobID          model.JobID
	allocationID   model.AllocationID
	allocation     *actor.Ref
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
		c.registeredTime = ctx.Self().RegisteredTime()
		if !c.restored {
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

		var proxyPortConf *sproto.ProxyPortConfig
		if c.GenericCommandSpec.Port != nil {
			proxyPortConf = &sproto.ProxyPortConfig{
				ServiceID:       string(c.taskID),
				Port:            *c.GenericCommandSpec.Port,
				ProxyTCP:        c.ProxyTCP,
				Unauthenticated: c.Unauthenticated,
			}
		}

		c.eventStream, _ = ctx.ActorOf("events", newEventManager(c.Config.Description))

		var eventStreamConfig *sproto.EventStreamConfig
		if c.eventStream != nil {
			eventStreamConfig = &sproto.EventStreamConfig{
				To: c.eventStream,
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

		allocation := task.NewAllocation(c.logCtx, sproto.AllocateRequest{
			AllocationID:      c.allocationID,
			TaskID:            c.taskID,
			JobID:             c.jobID,
			JobSubmissionTime: c.registeredTime,
			IsUserVisible:     true,
			Name:              c.Config.Description,
			AllocationRef:     ctx.Self(),
			Group:             ctx.Self(),

			SlotsNeeded:  c.Config.Resources.Slots,
			Label:        c.Config.Resources.AgentLabel,
			ResourcePool: c.Config.Resources.ResourcePool,
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},

			StreamEvents: eventStreamConfig,
			ProxyPort:    proxyPortConf,
			IdleTimeout:  idleWatcherConfig,
			Restore:      c.restored,
		}, c.db, c.rm, c.taskLogger)
		c.allocation, _ = ctx.ActorOf(c.allocationID, allocation)

		ctx.Self().System().TellAt(sproto.JobsActorAddr, sproto.RegisterJob{
			JobID:    c.jobID,
			JobActor: ctx.Self(),
		})

		ctx.Ask(c.allocation, actor.Ping{}).Get()
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
		ctx.Self().System().TellAt(sproto.JobsActorAddr, sproto.UnregisterJob{
			JobID: c.jobID,
		})
		if err := c.db.DeleteUserSessionByToken(c.GenericCommandSpec.Base.UserSessionToken); err != nil {
			ctx.Log().WithError(err).Errorf(
				"failure to delete user session for task: %v", c.taskID)
		}
	case actor.ChildStopped:
	case actor.ChildFailed:
		if msg.Child.Address().Local() == c.allocationID.String() && c.exitStatus == nil {
			c.exitStatus = &task.AllocationExited{
				FinalState: task.AllocationState{State: model.AllocationStateTerminated},
				Err:        errors.New("command allocation actor failed"),
			}
			if err := c.db.CompleteTask(c.taskID, time.Now().UTC()); err != nil {
				ctx.Log().WithError(err).Error("marking task complete")
			}
		}
	case task.BuildTaskSpec:
		if ctx.ExpectingResponse() {
			ctx.Respond(c.ToTaskSpec(c.GenericCommandSpec.Keys))
			// Evict the context from memory after starting the command as it is no longer needed. We
			// evict as soon as possible to prevent the master from hitting an OOM.
			// TODO: Consider not storing the userFiles in memory at all.
			c.UserFiles = nil
			c.AdditionalFiles = nil
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
	case *apiv1.IdleNotebookRequest:
		if !msg.Idle {
			ctx.Tell(c.allocation, task.IdleWatcherNoteActivity{LastActivity: time.Now()})
		}
		ctx.Respond(&apiv1.IdleNotebookResponse{})
	case *apiv1.KillNotebookRequest:
		// TODO(Brad): Do the same thing to allocations that we are doing to RMs.
		ctx.Tell(c.allocation, sproto.AllocationSignalWithReason{
			AllocationSignal:    sproto.KillAllocation,
			InformationalReason: "user requested kill",
		})
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
		ctx.Tell(c.allocation, sproto.AllocationSignalWithReason{
			AllocationSignal:    sproto.KillAllocation,
			InformationalReason: "user requested kill",
		})
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
		ctx.Tell(c.allocation, sproto.AllocationSignalWithReason{
			AllocationSignal:    sproto.KillAllocation,
			InformationalReason: "user requested kill",
		})
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
		ctx.Tell(c.allocation, sproto.AllocationSignalWithReason{
			AllocationSignal:    sproto.KillAllocation,
			InformationalReason: "user requested kill",
		})
		ctx.Respond(&apiv1.KillTensorboardResponse{Tensorboard: c.toTensorboard(ctx)})

	case *apiv1.SetTensorboardPriorityRequest:
		err := c.setPriority(ctx, int(msg.Priority), true)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(&apiv1.SetTensorboardPriorityResponse{Tensorboard: c.toTensorboard(ctx)})

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
		case rm.ErrUnsupported:
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
	case rm.ErrUnsupported:
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
	state := c.refreshAllocationState(ctx)
	return &notebookv1.Notebook{
		Id:             c.stringID(),
		State:          state.State.Proto(),
		Description:    c.Config.Description,
		Container:      state.FirstContainer().ToProto(),
		ServiceAddress: c.serviceAddress(),
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		JobId:          c.jobID.String(),
	}
}

func (c *command) toCommand(ctx *actor.Context) *commandv1.Command {
	state := c.refreshAllocationState(ctx)
	return &commandv1.Command{
		Id:           c.stringID(),
		State:        state.State.Proto(),
		Description:  c.Config.Description,
		Container:    state.FirstContainer().ToProto(),
		StartTime:    protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Username:     c.Base.Owner.Username,
		UserId:       int32(c.Base.Owner.ID),
		DisplayName:  c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool: c.Config.Resources.ResourcePool,
		ExitStatus:   c.exitStatus.String(),
		JobId:        c.jobID.String(),
	}
}

func (c *command) toShell(ctx *actor.Context) *shellv1.Shell {
	state := c.refreshAllocationState(ctx)
	return &shellv1.Shell{
		Id:             c.stringID(),
		State:          state.State.Proto(),
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Container:      state.FirstContainer().ToProto(),
		PrivateKey:     *c.Metadata.PrivateKey,
		PublicKey:      *c.Metadata.PublicKey,
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		Addresses:      toProto(state.FirstContainerAddresses()),
		AgentUserGroup: protoutils.ToStruct(c.Base.AgentUserGroup),
		JobId:          c.jobID.String(),
	}
}

func (c *command) toTensorboard(ctx *actor.Context) *tensorboardv1.Tensorboard {
	state := c.refreshAllocationState(ctx)
	return &tensorboardv1.Tensorboard{
		Id:             c.stringID(),
		State:          state.State.Proto(),
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Container:      state.FirstContainer().ToProto(),
		ServiceAddress: c.serviceAddress(),
		ExperimentIds:  c.Metadata.ExperimentIDs,
		TrialIds:       c.Metadata.TrialIDs,
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		JobId:          c.jobID.String(),
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

	resp, ok := ctx.Ask(c.allocation, task.AllocationState{}).GetOrTimeout(5 * time.Second)
	state, sOk := resp.(task.AllocationState)
	if !(ok && sOk) {
		ctx.Log().WithField("resp", resp).Warnf("getting allocation state")
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
