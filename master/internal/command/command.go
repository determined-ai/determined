package command

import (
	"fmt"
	"time"

	cproto "github.com/determined-ai/determined/master/pkg/container"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
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
	taskID model.TaskID,
	taskType model.TaskType,
	spec tasks.GenericCommandSpec,
) error {
	serviceAddress := fmt.Sprintf("/proxy/%s/", taskID)

	cmd := &command{
		db: db,

		GenericCommandSpec: spec,

		taskID:         taskID,
		taskType:       taskType,
		serviceAddress: &serviceAddress,
	}

	a, _ := ctx.ActorOf(cmd.taskID, cmd)
	summaryFut := ctx.Ask(a, getSummary{})
	if err := summaryFut.Error(); err != nil {
		ctx.Respond(errors.Wrap(err, "failed to create generic command"))
		return nil
	}
	summary := summaryFut.Get().(summary)
	ctx.Respond(summary.ID)
	return nil
}

// command is executed in a containerized environment on a Determined cluster.
type command struct {
	db          *db.PgDB
	eventStream *actor.Ref

	tasks.GenericCommandSpec

	registeredTime time.Time
	taskID         model.TaskID
	taskType       model.TaskType
	allocationID   model.AllocationID
	allocation     *actor.Ref
	serviceAddress *string
	lastState      task.AllocationState
	exitStatus     *task.AllocationExited
}

// Receive implements the actor.Actor interface.
func (c *command) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		c.allocationID = model.NewAllocationID(fmt.Sprintf("%s.%d", c.taskID, 1))
		c.registeredTime = ctx.Self().RegisteredTime()
		if err := c.db.AddTask(&model.Task{
			TaskID:    c.taskID,
			TaskType:  c.taskType,
			StartTime: c.registeredTime,
		}); err != nil {
			return errors.Wrapf(err, "persisting task %v", c.taskID)
		}

		c.eventStream, _ = ctx.ActorOf("events", newEventManager(c.Config.Description))

		ctx.Tell(sproto.GetRM(ctx.Self().System()), sproto.SetGroupPriority{
			Priority: c.Config.Resources.Priority,
			Handler:  ctx.Self(),
		})

		var portProxyConf *sproto.PortProxyConfig
		if c.GenericCommandSpec.Port != nil {
			portProxyConf = &sproto.PortProxyConfig{
				ServiceID: string(c.taskID),
				Port:      *c.GenericCommandSpec.Port,
				ProxyTCP:  c.ProxyTCP,
			}
		}

		var eventStreamConfig *sproto.EventStreamConfig
		if c.eventStream != nil {
			eventStreamConfig = &sproto.EventStreamConfig{
				To: c.eventStream,
			}
		}

		var logBasedReadinessConfig *sproto.LogBasedReadinessConfig
		if c.LogReadinessCheck != nil {
			logBasedReadinessConfig = &sproto.LogBasedReadinessConfig{
				Pattern: c.LogReadinessCheck,
			}
		}

		var idleWatcherConfig *sproto.IdleTimeoutConfig
		if c.Config.IdleTimeout != nil && (c.WatchProxyIdleTimeout || c.WatchRunnerIdleTimeout) {
			idleWatcherConfig = &sproto.IdleTimeoutConfig{
				ServiceID:       string(c.taskID),
				UseProxyState:   c.WatchProxyIdleTimeout,
				UseRunnerState:  c.WatchRunnerIdleTimeout,
				TimeoutDuration: time.Duration(*c.Config.IdleTimeout),
			}
		}

		allocation := task.NewAllocation(sproto.AllocateRequest{
			AllocationID: c.allocationID,
			TaskID:       c.taskID,
			Name:         c.Config.Description,
			TaskActor:    ctx.Self(),
			Group:        ctx.Self(),

			SlotsNeeded:  c.Config.Resources.Slots,
			Label:        c.Config.Resources.AgentLabel,
			ResourcePool: c.Config.Resources.ResourcePool,
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},

			StreamEvents:  eventStreamConfig,
			ProxyPort:     portProxyConf,
			IdleTimeout:   idleWatcherConfig,
			LogBasedReady: logBasedReadinessConfig,
		}, c.db, sproto.GetRM(ctx.Self().System()))
		c.allocation, _ = ctx.ActorOf(c.allocationID, allocation)

	case actor.PostStop:
		if err := c.db.CompleteTask(c.taskID, time.Now()); err != nil {
			ctx.Log().WithError(err).Error("marking task complete")
		}
	case actor.ChildStopped:
	case actor.ChildFailed:
		if msg.Child.Address().Local() == c.allocationID.String() && c.exitStatus == nil {
			c.exitStatus = &task.AllocationExited{
				FinalState: task.AllocationState{State: model.AllocationStateTerminated},
				Err:        errors.New("command allocation actor failed"),
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
		ctx.Tell(c.allocation, task.Kill)
		ctx.Respond(&apiv1.KillNotebookResponse{Notebook: c.toNotebook(ctx)})
	case *apiv1.SetNotebookPriorityRequest:
		c.setPriority(ctx, int(msg.Priority))
		ctx.Respond(&apiv1.SetNotebookPriorityResponse{Notebook: c.toNotebook(ctx)})

	case *commandv1.Command:
		ctx.Respond(c.toCommand(ctx))

	case *apiv1.GetCommandRequest:
		ctx.Respond(&apiv1.GetCommandResponse{
			Command: c.toCommand(ctx),
			Config:  protoutils.ToStruct(c.Config),
		})

	case *apiv1.KillCommandRequest:
		ctx.Tell(c.allocation, task.Kill)
		ctx.Respond(&apiv1.KillCommandResponse{Command: c.toCommand(ctx)})
	case *apiv1.SetCommandPriorityRequest:
		c.setPriority(ctx, int(msg.Priority))
		ctx.Respond(&apiv1.SetCommandPriorityResponse{Command: c.toCommand(ctx)})

	case *shellv1.Shell:
		ctx.Respond(c.toShell(ctx))

	case *apiv1.GetShellRequest:
		ctx.Respond(&apiv1.GetShellResponse{
			Shell:  c.toShell(ctx),
			Config: protoutils.ToStruct(c.Config),
		})

	case *apiv1.KillShellRequest:
		ctx.Tell(c.allocation, task.Kill)
		ctx.Respond(&apiv1.KillShellResponse{Shell: c.toShell(ctx)})
	case *apiv1.SetShellPriorityRequest:
		c.setPriority(ctx, int(msg.Priority))
		ctx.Respond(&apiv1.SetShellPriorityResponse{Shell: c.toShell(ctx)})

	case *tensorboardv1.Tensorboard:
		ctx.Respond(c.toTensorboard(ctx))

	case *apiv1.GetTensorboardRequest:
		ctx.Respond(&apiv1.GetTensorboardResponse{
			Tensorboard: c.toTensorboard(ctx),
			Config:      protoutils.ToStruct(c.Config),
		})

	case *apiv1.KillTensorboardRequest:
		ctx.Tell(c.allocation, task.Kill)
		ctx.Respond(&apiv1.KillTensorboardResponse{Tensorboard: c.toTensorboard(ctx)})
	case *apiv1.SetTensorboardPriorityRequest:
		c.setPriority(ctx, int(msg.Priority))
		ctx.Respond(&apiv1.SetTensorboardPriorityResponse{Tensorboard: c.toTensorboard(ctx)})

	case sproto.ContainerLog:

	case terminateForGC:
		ctx.Self().Stop()

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (c *command) setPriority(ctx *actor.Context, priority int) {
	ctx.Tell(sproto.GetRM(ctx.Self().System()), sproto.SetGroupPriority{
		Priority: &priority,
		Handler:  ctx.Self(),
	})
}

func (c *command) toNotebook(ctx *actor.Context) *notebookv1.Notebook {
	state := c.refreshAllocationState(ctx)
	return &notebookv1.Notebook{
		Id:             ctx.Self().Address().Local(),
		State:          state.State.Proto(),
		Description:    c.Config.Description,
		Container:      state.FirstContainer().Proto(),
		ServiceAddress: *c.serviceAddress,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Username:       c.Base.Owner.Username,
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
	}
}

func (c *command) toCommand(ctx *actor.Context) *commandv1.Command {
	state := c.refreshAllocationState(ctx)
	return &commandv1.Command{
		Id:           ctx.Self().Address().Local(),
		State:        state.State.Proto(),
		Description:  c.Config.Description,
		Container:    state.FirstContainer().Proto(),
		StartTime:    protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Username:     c.Base.Owner.Username,
		ResourcePool: c.Config.Resources.ResourcePool,
		ExitStatus:   c.exitStatus.String(),
	}
}

func (c *command) toShell(ctx *actor.Context) *shellv1.Shell {
	state := c.refreshAllocationState(ctx)
	return &shellv1.Shell{
		Id:             ctx.Self().Address().Local(),
		State:          state.State.Proto(),
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Container:      state.FirstContainer().Proto(),
		PrivateKey:     c.Metadata["privateKey"].(string),
		PublicKey:      c.Metadata["publicKey"].(string),
		Username:       c.Base.Owner.Username,
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		Addresses:      toProto(state.FirstContainerAddresses()),
		AgentUserGroup: protoutils.ToStruct(c.Base.AgentUserGroup),
	}
}

func (c *command) toTensorboard(ctx *actor.Context) *tensorboardv1.Tensorboard {
	state := c.refreshAllocationState(ctx)
	return &tensorboardv1.Tensorboard{
		Id:             ctx.Self().Address().Local(),
		State:          state.State.Proto(),
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Container:      state.FirstContainer().Proto(),
		ServiceAddress: *c.serviceAddress,
		ExperimentIds:  c.Metadata["experiment_ids"].([]int32),
		TrialIds:       c.Metadata["trial_ids"].([]int32),
		Username:       c.Base.Owner.Username,
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
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
	res := make([]*structpb.Struct, 0)
	for _, a := range as {
		res = append(res, protoutils.ToStruct(a))
	}
	return res
}
