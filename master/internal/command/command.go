package command

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// terminatedDuration defines the amount of time the command stays in a
// termianted state in the master before garbage collecting.
const terminatedDuration = 24 * time.Hour

const containerLostDuration = 5 * time.Minute

// TODO: readinessCheck should be defined at the agent level. Temporarily we will use log
// messages as a proxy.
type readinessCheck func(agent.ContainerLog) bool

// terminateForGC is an internal message indicating that the command actor
// should stop and garbage collect its state.
type terminateForGC struct{}

type lostID uuid.UUID

// commandOwner describes the owner of a command.
type commandOwner struct {
	ID       model.UserID `json:"id"`
	Username string       `json:"username"`
}

// DefaultConfig is the default configuration used by all
// commands (e.g., commands, notebooks, shells) if a request
// does not specify any configuration options.
func DefaultConfig() model.CommandConfig {
	environment := model.DefaultExperimentConfig().Environment
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
	exitStatus     *string
	addresses      []scheduler.Address

	cluster     *actor.Ref
	eventStream *actor.Ref

	containerLost *lostID
}

// Receive implements the actor.Actor interface.
func (c *command) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		c.registeredTime = ctx.Self().RegisteredTime()
		// Initialize an event stream manager.
		c.eventStream, _ = ctx.ActorOf("events", newEventManager())
		// Schedule the command with the cluster.
		c.cluster = ctx.Self().System().Get(actor.Addr("cluster"))
		ctx.Tell(c.cluster, scheduler.AddTask{
			ID:           &c.taskID,
			Name:         c.config.Description,
			SlotsNeeded:  c.config.Resources.Slots,
			Label:        c.config.Resources.AgentLabel,
			CanTerminate: true,
			FittingRequirements: scheduler.FittingRequirements{
				SingleAgent:    true,
				DedicatedAgent: false,
			},
		})
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ScheduledEvent: &c.taskID})

	case getSummary:
		if msg.userFilter == "" || c.owner.Username == msg.userFilter {
			ctx.Respond(newSummary(c))
		}

	case agent.ContainerStateChanged:
		c.container = &msg.Container
		switch {
		case msg.Container.State == container.Terminated &&
			msg.ContainerStopped.Failure != nil &&
			msg.ContainerStopped.Failure.FailureType == agent.AgentFailed:
			id := lostID(uuid.New())
			c.containerLost = &id
			actors.NotifyAfter(ctx, containerLostDuration, id)
		case msg.Container.State == container.Terminated:
			exitStatus := "command exited successfully"
			if msg.ContainerStopped.Failure != nil {
				exitStatus = msg.ContainerStopped.Failure.Error()
			}
			c.exit(ctx, exitStatus)
		}

	case scheduler.Assigned:
		msg.StartTask(tasks.TaskSpec{
			StartCommand: &tasks.StartCommand{
				AgentUserGroup:  c.agentUserGroup,
				Config:          c.config,
				UserFiles:       c.userFiles,
				AdditionalFiles: c.additionalFiles,
			},
			HarnessPath: c.harnessPath,
			Recoverable: true,
		})
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), AssignedEvent: &msg})

		// Evict the context from memory after starting the command as it is no longer needed. We
		// evict as soon as possible to prevent the master from hitting an OOM.
		// TODO: Consider not storing the userFiles in memory at all.
		c.userFiles = nil
		c.additionalFiles = nil

	case scheduler.ContainerStarted:
		c.addresses = msg.Container.Addresses()
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ContainerStartedEvent: &msg})

	case scheduler.TerminateRequest:
		c.terminate(ctx)

	case scheduler.TaskAborted:
		c.exit(ctx, "command terminated without being scheduled")

	case scheduler.TaskTerminated:
		// This message is being deprecated; ignore it.

	case agent.ContainerLog:
		if !c.readinessMessageSent && c.readinessChecksPass(ctx, msg) {
			c.readinessMessageSent = true
			ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), ServiceReadyEvent: &msg})
		}
		log := msg.String()
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), LogEvent: &log})

	case lostID:
		if c.exitStatus == nil && c.containerLost != nil && *c.containerLost == msg {
			c.exit(ctx, "container was lost on agent")
		}

	case agent.ContainerRecovered:
		if c.exitStatus == nil {
			c.container = &msg.Container
			c.containerLost = nil
		}

	case terminateForGC:
		ctx.Self().Stop()

	case echo.Context:
		c.handleAPIRequest(ctx, msg)
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
	if c.containerLost != nil {
		c.exit(ctx, "container was asked to terminate while lost")
		return
	}
	ctx.Ask(c.cluster, scheduler.TerminateTask{TaskID: c.taskID, Forcible: true}).Get()
	if msg, ok := ctx.Message().(scheduler.TerminateRequest); ok {
		ctx.Tell(c.eventStream, event{Snapshot: newSummary(c), TerminateRequestEvent: &msg})
	}
}

func (c *command) readinessChecksPass(ctx *actor.Context, log agent.ContainerLog) bool {
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
