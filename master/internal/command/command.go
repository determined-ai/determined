package command

import (
	"fmt"
	"net/url"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
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

// TODO: readinessCheck should be defined at the agent level. Temporarily we will use log
// messages as a proxy.
type readinessCheck func(sproto.ContainerLog) bool

// terminateForGC is an internal message indicating that the command actor
// should stop and garbage collect its state.
type terminateForGC struct{}

func createGenericCommandActor(
	ctx *actor.Context,
	db *db.PgDB,
	taskID model.TaskID,
	spec tasks.GenericCommandSpec,
	readinessCheck map[string]readinessCheck,
) error {
	serviceAddress := fmt.Sprintf("/proxy/%s/", taskID)

	cmd := &command{
		db:              db,
		readinessChecks: readinessCheck,

		GenericCommandSpec: spec,

		taskID:         taskID,
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

// DefaultConfig is the default configuration used by all
// commands (e.g., commands, notebooks, shells) if a request
// does not specify any configuration options.
func DefaultConfig(taskContainerDefaults *model.TaskContainerDefaultsConfig) model.CommandConfig {
	expConf := model.DefaultExperimentConfig(taskContainerDefaults)
	expConf.Resources.Slots = 1
	return model.CommandConfig{
		Resources:   expConf.Resources,
		Environment: expConf.Environment,
		BindMounts:  expConf.BindMounts,
	}
}

// command is executed in a containerized environment on a Determined cluster.
type command struct {
	db          *db.PgDB
	proxy       *actor.Ref
	eventStream *actor.Ref

	readinessChecks map[string]readinessCheck

	tasks.GenericCommandSpec

	taskID         model.TaskID
	allocationID   model.AllocationID
	serviceAddress *string

	readinessMessageSent bool
	registeredTime       time.Time
	task                 *sproto.AllocateRequest
	container            *container.Container
	allocation           sproto.Reservation
	proxyNames           []string
	exitStatus           *string
	addresses            []container.Address
}

// Receive implements the actor.Actor interface.
func (c *command) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		c.registeredTime = ctx.Self().RegisteredTime()
		// Initialize an event stream manager.
		c.eventStream, _ = ctx.ActorOf("events", newEventManager())
		// Schedule the command with the cluster.
		c.proxy = ctx.Self().System().Get(actor.Addr("proxy"))

		// Since command tasks a single allocation, allocation ID is just the taskID.
		c.allocationID = model.NewAllocationID(string(c.taskID))
		c.task = &sproto.AllocateRequest{
			AllocationID:   c.allocationID,
			Name:           c.Config.Description,
			SlotsNeeded:    c.Config.Resources.Slots,
			Label:          c.Config.Resources.AgentLabel,
			ResourcePool:   c.Config.Resources.ResourcePool,
			NonPreemptible: true,
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: true,
			},
			TaskActor: ctx.Self(),
		}
		if err := ctx.Ask(sproto.GetRM(ctx.Self().System()), *c.task).Error(); err != nil {
			return err
		}
		ctx.Tell(sproto.GetRM(ctx.Self().System()), sproto.SetGroupPriority{
			Priority: c.Config.Resources.Priority,
			Handler:  ctx.Self(),
		})
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ScheduledEvent: &c.allocationID})

	case actor.PostStop:
		c.terminate(ctx)

	case sproto.ResourcesAllocated:
		return c.receiveSchedulerMsg(ctx)

	case getSummary:
		if msg.userFilter == "" || c.Base.Owner.Username == msg.userFilter {
			ctx.Respond(newSummary(c))
		}

	case *notebookv1.Notebook:
		ctx.Respond(c.toNotebook(ctx))

	case *apiv1.GetNotebookRequest:
		ctx.Respond(&apiv1.GetNotebookResponse{
			Notebook: c.toNotebook(ctx),
			Config:   protoutils.ToStruct(c.Config),
		})

	case *apiv1.KillNotebookRequest:
		c.terminate(ctx)
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
		c.terminate(ctx)
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
		c.terminate(ctx)
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
		c.terminate(ctx)
		ctx.Respond(&apiv1.KillTensorboardResponse{Tensorboard: c.toTensorboard(ctx)})
	case *apiv1.SetTensorboardPriorityRequest:
		c.setPriority(ctx, int(msg.Priority))
		ctx.Respond(&apiv1.SetTensorboardPriorityResponse{Tensorboard: c.toTensorboard(ctx)})

	case sproto.TaskContainerStateChanged:
		c.container = &msg.Container

		switch {
		case msg.Container.State == container.Running:
			c.addresses = msg.ContainerStarted.Addresses
			assignedPort := c.GenericCommandSpec.Port
			// TODO(DET-5682): refactor this logic and the rendezvous info logic that does the same
			// thing into a helper function.
			var names []string
			for _, address := range c.addresses {
				// Only proxy the port we expect to proxy.  If a dockerfile uses an EXPOSE command,
				// additional addresses will appear her, but currently we only proxy one uuid to one
				// port, so it doesn't make sense to send multiple proxy.Register messages for a
				// single ServiceID (only the last one would work).
				if assignedPort == nil || address.ContainerPort != *assignedPort {
					continue
				}

				// We are keying on allocation id instead of container id. Revisit this when we need to
				// proxy multi-container tasks or when containers are created prior to being
				// assigned to an agent.
				ctx.Ask(c.proxy, proxy.Register{
					ServiceID: string(c.taskID),
					URL: &url.URL{
						Scheme: "http",
						Host:   fmt.Sprintf("%s:%d", address.HostIP, address.HostPort),
					},
					ProxyTCP: c.GenericCommandSpec.ProxyTCP,
				})
				names = append(names, string(c.taskID))
			}
			if assignedPort == nil && len(names) > 0 {
				ctx.Log().Error("expected to not proxy any ports but proxied one anyway")
			} else if len(names) != 1 {
				ctx.Log().Errorf(
					"expected to proxy exactly 1 port but proxied %v instead", len(names),
				)
			}
			c.proxyNames = names
			ctx.Tell(c.eventStream, event{
				Snapshot: newSummary(c), ContainerStartedEvent: msg.ContainerStarted,
			})

		case msg.Container.State == container.Terminated:
			for _, name := range c.proxyNames {
				ctx.Tell(c.proxy, proxy.Unregister{ServiceID: name})
			}
			c.proxyNames = make([]string, 0)

			exitStatus := "command exited successfully"
			if msg.ContainerStopped.Failure != nil {
				exitStatus = msg.ContainerStopped.Failure.Error()
			}

			c.exit(ctx, exitStatus)
		}

	case sproto.ContainerLog:
		if !c.readinessMessageSent && c.readinessChecksPass(ctx, msg) {
			c.readinessMessageSent = true
			ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ServiceReadyEvent: &msg})
		}
		log := msg.String()
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), LogEvent: &log})

	case terminateForGC:
		ctx.Self().Stop()

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (c *command) receiveSchedulerMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ResourcesAllocated:
		// Ignore this message if the command has exited.
		if c.task == nil || msg.ID != c.task.AllocationID {
			ctx.Log().Info("ignoring resource allocation since the command has exited.")
			return nil
		}

		check.Panic(check.Equal(len(msg.Reservations), 1,
			"Command should only receive an allocation of one container"))

		taskToken, err := c.db.StartAllocationSession(c.task.AllocationID)
		if err != nil {
			return errors.Wrap(err, "cannot start a new task session")
		}

		c.allocation = msg.Reservations[0]

		msg.Reservations[0].Start(ctx, c.ToTaskSpec(c.GenericCommandSpec.Keys, taskToken), 0)

		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), AssignedEvent: &msg})

		// Evict the context from memory after starting the command as it is no longer needed. We
		// evict as soon as possible to prevent the master from hitting an OOM.
		// TODO: Consider not storing the userFiles in memory at all.
		c.UserFiles = nil
		c.AdditionalFiles = nil

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// terminate handles the following cases of command termination:
// 1. Command is aborted before being allocated.
// 2. Forcible terminating a command by killing containers.
func (c *command) terminate(ctx *actor.Context) {
	if msg, ok := ctx.Message().(sproto.ReleaseResources); ok {
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), TerminateRequestEvent: &msg})
	}

	if c.allocation == nil {
		c.exit(ctx, "task is aborted without being scheduled")
	} else {
		ctx.Log().Info("task forcible terminating")
		c.allocation.Kill(ctx)
	}
}

// exit handles the following cases of command exiting:
// 1. Command is aborted before being allocated.
// 2. Forcible terminating a command by killing containers.
// 3. The command container exits itself.
func (c *command) exit(ctx *actor.Context, exitStatus string) {
	c.exitStatus = &exitStatus
	ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ExitedEvent: c.exitStatus})

	ctx.Tell(
		sproto.GetRM(ctx.Self().System()),
		sproto.ResourcesReleased{TaskActor: ctx.Self()},
	)
	actors.NotifyAfter(ctx, terminatedDuration, terminateForGC{})

	if c.task != nil {
		if err := c.db.DeleteAllocationSession(c.task.AllocationID); err != nil {
			ctx.Log().WithError(err).Error("cannot delete task session for a command")
		}
	}
}

func (c *command) setPriority(ctx *actor.Context, priority int) {
	ctx.Tell(sproto.GetRM(ctx.Self().System()), sproto.SetGroupPriority{
		Priority: &priority,
		Handler:  ctx.Self(),
	})
}

func (c *command) readinessChecksPass(ctx *actor.Context, log sproto.ContainerLog) bool {
	for name, check := range c.readinessChecks {
		if check(log) {
			delete(c.readinessChecks, name)
			ctx.Log().Infof("readiness check passed: %s", name)
		}
	}
	return len(c.readinessChecks) == 0
}

// State returns the command's state. This mirros the associated container's state
// if available.
func (c *command) State() State {
	state := Pending
	switch {
	case c.container != nil:
		switch c.container.State {
		case container.Assigned:
			state = Assigned
		case container.Pulling:
			state = Pulling
		case container.Starting:
			state = Starting
		case container.Running:
			state = Running
		case container.Terminated:
			state = Terminated
		}
	case c.exitStatus != nil:
		state = Terminated
	}
	return state
}

func (c *command) toNotebook(ctx *actor.Context) *notebookv1.Notebook {
	exitStatus := protoutils.DefaultStringValue
	if c.exitStatus != nil {
		exitStatus = *c.exitStatus
	}

	return &notebookv1.Notebook{
		Id:             ctx.Self().Address().Local(),
		State:          c.State().Proto(),
		Description:    c.Config.Description,
		Container:      c.container.Proto(),
		ServiceAddress: *c.serviceAddress,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Username:       c.Base.Owner.Username,
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     exitStatus,
	}
}

func (c *command) toCommand(ctx *actor.Context) *commandv1.Command {
	exitStatus := protoutils.DefaultStringValue
	if c.exitStatus != nil {
		exitStatus = *c.exitStatus
	}

	return &commandv1.Command{
		Id:           ctx.Self().Address().Local(),
		State:        c.State().Proto(),
		Description:  c.Config.Description,
		Container:    c.container.Proto(),
		StartTime:    protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Username:     c.Base.Owner.Username,
		ResourcePool: c.Config.Resources.ResourcePool,
		ExitStatus:   exitStatus,
	}
}

func (c *command) toShell(ctx *actor.Context) *shellv1.Shell {
	exitStatus := protoutils.DefaultStringValue
	if c.exitStatus != nil {
		exitStatus = *c.exitStatus
	}

	addresses := make([]*structpb.Struct, 0)
	for _, addr := range c.addresses {
		addresses = append(addresses, protoutils.ToStruct(addr))
	}

	return &shellv1.Shell{
		Id:             ctx.Self().Address().Local(),
		State:          c.State().Proto(),
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Container:      c.container.Proto(),
		PrivateKey:     c.Metadata["privateKey"].(string),
		PublicKey:      c.Metadata["publicKey"].(string),
		Username:       c.Base.Owner.Username,
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     exitStatus,
		Addresses:      addresses,
		AgentUserGroup: protoutils.ToStruct(c.Base.AgentUserGroup),
	}
}

func (c *command) toTensorboard(ctx *actor.Context) *tensorboardv1.Tensorboard {
	exitStatus := protoutils.DefaultStringValue
	if c.exitStatus != nil {
		exitStatus = *c.exitStatus
	}

	return &tensorboardv1.Tensorboard{
		Id:             ctx.Self().Address().Local(),
		State:          c.State().Proto(),
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Container:      c.container.Proto(),
		ServiceAddress: *c.serviceAddress,
		ExperimentIds:  c.Metadata["experiment_ids"].([]int32),
		TrialIds:       c.Metadata["trial_ids"].([]int32),
		Username:       c.Base.Owner.Username,
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     exitStatus,
	}
}
