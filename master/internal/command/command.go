package command

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/determined-ai/determined/master/internal/proxy"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/archive"
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
// termianted state in the master before garbage collecting.
const terminatedDuration = 24 * time.Hour

// TODO: readinessCheck should be defined at the agent level. Temporarily we will use log
// messages as a proxy.
type readinessCheck func(sproto.ContainerLog) bool

// terminateForGC is an internal message indicating that the command actor
// should stop and garbage collect its state.
type terminateForGC struct{}

// commandOwner describes the owner of a command.
type commandOwner struct {
	ID       model.UserID `json:"id"`
	Username string       `json:"username"`
}

// DefaultConfig is the default configuration used by all
// commands (e.g., commands, notebooks, shells) if a request
// does not specify any configuration options.
func DefaultConfig(taskContainerDefaults *model.TaskContainerDefaultsConfig) model.CommandConfig {
	environment := model.DefaultExperimentConfig(taskContainerDefaults).Environment
	return model.CommandConfig{
		Resources: model.ResourcesConfig{
			Slots:  1,
			Weight: 1,
			// SlotsPerTrial is not used by commands. They prefer Slots instead.
			// It is only defined here to pass check.Validate.
			SlotsPerTrial: 1,
		},
		Environment: environment,
	}
}

// command is executed in a containerized environment on a Determined cluster.
type command struct {
	config model.CommandConfig

	owner          commandOwner
	agentUserGroup *model.AgentUserGroup

	taskID               scheduler.TaskID
	userFiles            archive.Archive
	additionalFiles      archive.Archive
	harnessPath          string
	readinessChecks      map[string]readinessCheck
	readinessMessageSent bool
	metadata             map[string]interface{}
	serviceAddress       *string

	registeredTime time.Time
	container      *container.Container
	assignment     scheduler.Assignment
	proxyNames     []string
	exitStatus     *string
	addresses      []scheduler.Address

	proxy       *actor.Ref
	rps         *actor.Ref
	eventStream *actor.Ref
}

// Receive implements the actor.Actor interface.
func (c *command) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		c.registeredTime = ctx.Self().RegisteredTime()
		// Initialize an event stream manager.
		c.eventStream, _ = ctx.ActorOf("events", newEventManager())
		// Schedule the command with the cluster.
		c.rps = ctx.Self().System().Get(actor.Addr("resourceProviders"))
		c.proxy = ctx.Self().System().Get(actor.Addr("proxy"))
		ctx.Tell(c.rps, scheduler.AssignResource{
			ID:           &c.taskID,
			Name:         c.config.Description,
			SlotsNeeded:  c.config.Resources.Slots,
			Label:        c.config.Resources.AgentLabel,
			CanTerminate: true,
			FittingRequirements: scheduler.FittingRequirements{
				SingleAgent: true,
			},
			TaskHandler: ctx.Self(),
		})
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ScheduledEvent: &c.taskID})

	case actor.PostStop:
		c.terminate(ctx)

	case getSummary:
		if msg.userFilter == "" || c.owner.Username == msg.userFilter {
			ctx.Respond(newSummary(c))
		}

	case *notebookv1.Notebook:
		notebook, err := c.toNotebook(ctx)
		switch {
		case err != nil:
			ctx.Log().Error(err)
		default:
			ctx.Respond(notebook)
		}

	case *apiv1.GetNotebookRequest:
		notebook, err := c.toNotebook(ctx)
		switch {
		case err != nil:
			ctx.Log().Error(err)
		default:
			ctx.Respond(&apiv1.GetNotebookResponse{
				Notebook: notebook,
				Config:   protoutils.ToStruct(c.config),
			})
		}

	case *apiv1.KillNotebookRequest:
		notebook, err := c.toNotebook(ctx)
		switch {
		case err != nil:
			ctx.Log().Error(err)
		default:
			c.terminate(ctx)
			ctx.Respond(&apiv1.KillNotebookResponse{Notebook: notebook})
		}

	case *commandv1.Command:
		ctx.Respond(c.toCommand(ctx))

	case *apiv1.GetCommandRequest:
		ctx.Respond(&apiv1.GetCommandResponse{
			Command: c.toCommand(ctx),
			Config:  protoutils.ToStruct(c.config),
		})

	case *apiv1.KillCommandRequest:
		c.terminate(ctx)
		ctx.Respond(&apiv1.KillCommandResponse{Command: c.toCommand(ctx)})

	case *shellv1.Shell:
		ctx.Respond(c.toShell(ctx))

	case *apiv1.GetShellRequest:
		ctx.Respond(&apiv1.GetShellResponse{
			Shell:  c.toShell(ctx),
			Config: protoutils.ToStruct(c.config),
		})

	case *apiv1.KillShellRequest:
		c.terminate(ctx)
		ctx.Respond(&apiv1.KillShellResponse{Shell: c.toShell(ctx)})

	case *tensorboardv1.Tensorboard:
		ctx.Respond(c.toTensorboard(ctx))

	case *apiv1.GetTensorboardRequest:
		ctx.Respond(&apiv1.GetTensorboardResponse{Tensorboard: c.toTensorboard(ctx)})

	case *apiv1.KillTensorboardRequest:
		c.terminate(ctx)
		ctx.Respond(&apiv1.KillTensorboardResponse{Tensorboard: c.toTensorboard(ctx)})

	case sproto.ContainerStateChanged:
		c.container = &msg.Container
		if msg.Container.State == container.Terminated {
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

	case scheduler.ResourceAssigned:
		c.assignment = msg.Assignments[0]
		msg.Assignments[0].StartContainer(tasks.TaskSpec{
			StartCommand: &tasks.StartCommand{
				AgentUserGroup:  c.agentUserGroup,
				Config:          c.config,
				UserFiles:       c.userFiles,
				AdditionalFiles: c.additionalFiles,
			},
			HarnessPath: c.harnessPath,
		})

		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), AssignedEvent: &msg})

		// Evict the context from memory after starting the command as it is no longer needed. We
		// evict as soon as possible to prevent the master from hitting an OOM.
		// TODO: Consider not storing the userFiles in memory at all.
		c.userFiles = nil
		c.additionalFiles = nil

	case scheduler.ReleaseResource:
		c.terminate(ctx)

	case scheduler.ContainerStarted:
		c.addresses = msg.Container.Addresses()

		names := make([]string, 0, len(msg.Container.Addresses()))
		for _, address := range msg.Container.Addresses() {
			// We are keying on task ID instead of container ID. Revisit this when we need to
			// proxy multi-container tasks or when containers are created prior to being
			// assigned to an agent.
			ctx.Ask(c.proxy, proxy.Register{
				ServiceID: string(c.taskID),
				URL: &url.URL{
					Scheme: "http",
					Host:   fmt.Sprintf("%s:%d", address.HostIP, address.HostPort),
				},
			})
			names = append(names, string(c.taskID))
		}
		c.proxyNames = names

		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ContainerStartedEvent: &msg})

	case scheduler.TaskTerminated:
		// This message is being deprecated; ignore it.

	case sproto.ContainerLog:
		if !c.readinessMessageSent && c.readinessChecksPass(ctx, msg) {
			c.readinessMessageSent = true
			ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ServiceReadyEvent: &msg})
		}
		log := msg.String()
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), LogEvent: &log})

	case terminateForGC:
		ctx.Self().Stop()

	case echo.Context:
		c.handleAPIRequest(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// handleAPIRequest handles API requests inbound to this actor.
func (c *command) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, newSummary(c)))
	case echo.DELETE:
		c.terminate(ctx)
		ctx.Respond(apiCtx.NoContent(http.StatusAccepted))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (c *command) terminate(ctx *actor.Context) {
	ctx.Log().Info("terminating")
	ctx.Ask(c.rps, scheduler.TerminateTask{TaskID: c.taskID, Forcible: true}).Get()
	if c.assignment != nil {
		c.assignment.KillContainer()
	} else {
		ctx.Log().Warn("found no started container")
	}
	if msg, ok := ctx.Message().(scheduler.ReleaseResource); ok {
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), TerminateRequestEvent: &msg})
	}
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

func (c *command) exit(ctx *actor.Context, exitStatus string) {
	c.exitStatus = &exitStatus
	ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ExitedEvent: c.exitStatus})
	actors.NotifyAfter(ctx, terminatedDuration, terminateForGC{})
}

func (c *command) toNotebook(ctx *actor.Context) (*notebookv1.Notebook, error) {
	serviceAddress, err := generateServiceAddress(string(c.taskID))
	if err != nil {
		return nil, errors.Wrapf(err, "generating service address for %s", c.taskID)
	}
	return &notebookv1.Notebook{
		Id:             ctx.Self().Address().Local(),
		Description:    c.config.Description,
		Container:      c.container.Proto(),
		ServiceAddress: serviceAddress,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Username:       c.owner.Username,
	}, nil
}

func (c *command) toCommand(ctx *actor.Context) *commandv1.Command {
	return &commandv1.Command{
		Id:          ctx.Self().Address().Local(),
		Description: c.config.Description,
		Container:   c.container.Proto(),
		StartTime:   protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Username:    c.owner.Username,
	}
}

func (c *command) toShell(ctx *actor.Context) *shellv1.Shell {
	return &shellv1.Shell{
		Id:          ctx.Self().Address().Local(),
		Description: c.config.Description,
		StartTime:   protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Container:   c.container.Proto(),
		PrivateKey:  c.metadata["privateKey"].(string),
		PublicKey:   c.metadata["publicKey"].(string),
		Username:    c.owner.Username,
	}
}

func (c *command) toTensorboard(ctx *actor.Context) *tensorboardv1.Tensorboard {
	var eids []int32
	for _, id := range c.metadata["experiment_ids"].([]int) {
		eids = append(eids, int32(id))
	}
	var tids []int32
	for _, id := range c.metadata["trial_ids"].([]int) {
		tids = append(tids, int32(id))
	}
	return &tensorboardv1.Tensorboard{
		Id:             ctx.Self().Address().Local(),
		Description:    c.config.Description,
		StartTime:      protoutils.ToTimestamp(ctx.Self().RegisteredTime()),
		Container:      c.container.Proto(),
		ServiceAddress: fmt.Sprintf(tensorboardServiceAddress, c.taskID),
		ExperimentIds:  eids,
		TrialIds:       tids,
		Username:       c.owner.Username,
	}
}

func getPort(min, max int) int {
	// Set the seed here or else the compiler will generate a random number at
	// compile time and each invocation of this function will return the same
	// number.
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func setPodSpec(
	config *model.CommandConfig,
	taskContainerDefaults model.TaskContainerDefaultsConfig,
) {
	if config.Environment.PodSpec != nil {
		return
	}

	if config.Resources.Slots == 0 {
		config.Environment.PodSpec = taskContainerDefaults.CPUPodSpec
	} else {
		config.Environment.PodSpec = taskContainerDefaults.GPUPodSpec
	}
}
