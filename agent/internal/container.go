package internal

import (
	"fmt"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

const (
	agentIDEnvVar      = "DET_AGENT_ID"
	taskIDEnvVar       = "DET_TASK_ID"
	allocationIDEnvVar = "DET_ALLOCATION_ID"
	containerIDEnvVar  = "DET_CONTAINER_ID"
)

type containerActor struct {
	cproto.Container
	spec          *cproto.Spec
	client        *client.Client
	docker        *actor.Ref
	containerInfo *types.ContainerJSON

	// Keeps track of why we exited. Always valid with a terminated state.
	stop *aproto.ContainerStopped

	baseTaskLog model.TaskLog
	reattached  bool
}

type (
	getContainerSummary struct{}
	containerReady      struct{}
)

func newContainerActor(msg aproto.StartContainer, client *client.Client) actor.Actor {
	return &containerActor{
		Container: msg.Container, spec: &msg.Spec, client: client,
	}
}

func reattachContainerActor(
	container cproto.Container, client *client.Client,
) actor.Actor {
	return &containerActor{
		Container: container, client: client, reattached: true,
	}
}

// getBaseTaskLog computes the container-specific extra fields to be injected into each Fluent
// log entry. We configure Docker to send these fields itself, but we need to compute and add them
// ourselves for agent-inserted logs.
func getBaseTaskLog(spec *cproto.Spec) model.TaskLog {
	level := "INFO"
	stdtype := "stdout"
	log := model.TaskLog{
		Level:   &level,
		StdType: &stdtype,
	}
	for _, env := range spec.RunSpec.ContainerConfig.Env {
		split := strings.SplitN(env, "=", 2)
		// For container logging config, ignore environment variables of
		// form 'DET_TASK_ID' when they should be 'DET_TASK_ID=x'.
		if len(split) < 2 {
			continue
		}

		value := split[1]
		switch split[0] {
		case agentIDEnvVar:
			log.AgentID = &value
		case containerIDEnvVar:
			log.ContainerID = &value
		case taskIDEnvVar:
			log.TaskID = value
		case allocationIDEnvVar:
			log.AllocationID = ptrs.Ptr(value)
		}
	}
	return log
}

func (c *containerActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if !c.reattached {
			c.docker, _ = ctx.ActorOf("docker", &dockerActor{Client: c.client})
			taskLog := getBaseTaskLog(c.spec)
			c.transition(ctx, cproto.Pulling)
			pull := pullImage{
				PullSpec:     c.spec.PullSpec,
				Name:         c.spec.RunSpec.ContainerConfig.Image,
				TaskType:     c.spec.TaskType,
				AllocationID: model.AllocationID(*taskLog.AllocationID),
			}
			ctx.Tell(c.docker, pull)
			c.baseTaskLog = taskLog
		} else {
			c.docker, _ = ctx.ActorOf(
				"docker",
				&dockerActor{
					Client:              c.client,
					reattachContainerID: &c.Container.ID,
				})
			ctx.Ask(c.docker, actor.Ping{}).Get()
		}
	case getContainerSummary:
		ctx.Respond(c.Container)

	case imagePulled:
		if c.State == cproto.Terminated {
			// This can happen if we are terminated while pulling, since `ctx.Self().Stop()` still
			// allows us to read all our messages.
			ctx.Log().Warn("ignoring pull complete message for terminated container")
			return nil
		}

		c.transition(ctx, cproto.Starting)
		ctx.Tell(c.docker, runContainer{c.spec.RunSpec})

	case containerStarted:
		c.containerInfo = &msg.containerInfo

		if len(c.spec.RunSpec.ChecksConfig.Checks) == 0 {
			ctx.Tell(ctx.Self(), containerReady{})
			return nil
		}

		checker, err := newCheckerActor(c.spec.RunSpec.ChecksConfig, msg.containerInfo)
		if err != nil {
			return errors.Wrap(err, "failed to set up readiness check")
		}
		ctx.ActorOf("checker", checker)

		// Evict the spec from memory due to their potentially massive memory consumption.
		c.spec = nil

	case containerReattached:
		c.containerInfo = &msg.containerInfo
		// TODO(ilia): When do we need to start a checker for these containers?

	case containerReady:
		c.containerStarted(ctx, aproto.ContainerStarted{ContainerInfo: *c.containerInfo})

	case containerTerminated:
		ctx.Log().Debug("containerTerminated")
		c.containerStopped(ctx, aproto.ContainerExited(msg.ExitCode))
		ctx.Self().Stop()

	case aproto.ContainerStatsRecord:
		ctx.Tell(ctx.Self().Parent(), msg)

	case aproto.SignalContainer:
		switch c.State {
		case cproto.Assigned, cproto.Pulling:
			switch msg.Signal {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL:
				msg := "attempting to stop container %s while in [%s] state"
				ctx.Log().Infof(msg, c.ID, c.State)
				ctx.Self().Stop()
				c.containerStopped(ctx,
					aproto.ContainerError(aproto.ContainerAborted, fmt.Errorf(msg, c.ID, c.State)))
			default:
				ctx.Log().Warnf(
					"ignoring signal, signal not supported while container %s in [%s] state",
					c.ID, c.State)
			}
		case cproto.Starting:
			// Delay signaling the container until the container is actually running.
			ctx.Tell(ctx.Self(), msg)
		case cproto.Running:
			ctx.Log().Infof("sending signal to container: %s", msg.Signal)
			ctx.Log().Debugf("docker: %v, contInfo: %v", c.docker, c.containerInfo)
			ctx.Tell(c.docker, signalContainer{dockerID: c.containerInfo.ID, signal: msg.Signal})
		case cproto.Terminated:
			err := fmt.Errorf("re-acknowledging signal, container already terminated: %s", msg.Signal)
			ctx.Log().Warnf(err.Error())
			c.containerStopped(ctx, aproto.ContainerError(aproto.AgentFailed, err))
		}

	case aproto.ContainerLog:
		msg.Container = c.Container
		ctx.Log().Debug(msg)
		ctx.Tell(ctx.Self().Parent(), c.makeTaskLog(msg))
	case actor.ChildStopped:

	case actor.ChildFailed:
		c.containerStopped(ctx, aproto.ContainerError(aproto.ContainerFailed, msg.Error))
		return msg.Error

	case dockerErr:
		c.containerStopped(ctx, aproto.ContainerError(aproto.ContainerFailed, msg.Error))
		return msg.Error

	case echo.Context:
		c.handleAPIRequest(ctx, msg)

	case actor.PostStop:
		if c.State == cproto.Running {
			ctx.Log().Infof("disconnecting from container: %s", c.containerInfo.ID)
		} else {
			ctx.Log().Info("container stopped")
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (c *containerActor) makeTaskLog(log aproto.ContainerLog) model.TaskLog {
	l := c.baseTaskLog
	timestamp := time.Now().UTC()
	l.Timestamp = &timestamp
	if log.Level != nil {
		l.Level = log.Level
	}

	var source string
	var msg string
	switch {
	case log.AuxMessage != nil:
		source = "agent"
		msg = *log.AuxMessage
	case log.PullMessage != nil:
		msg = *log.PullMessage
	case log.RunMessage != nil:
		panic(fmt.Sprintf("unexpected run message from container on Fluent logging: %v", log.RunMessage))
	default:
		panic("unknown log message received")
	}
	msg += "\n"

	l.Log = msg
	l.Source = &source

	return l
}

func (c *containerActor) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, c.Container))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (c *containerActor) transition(ctx *actor.Context, newState cproto.State) {
	ctx.Log().Infof("transitioning state from %s to %s", c.State, newState)
	c.Container = c.Transition(newState)
	ctx.Tell(ctx.Self().Parent(), aproto.ContainerStateChanged{Container: c.Container})
}

func (c *containerActor) containerStarted(ctx *actor.Context, started aproto.ContainerStarted) {
	newState := cproto.Running
	ctx.Log().Infof("transitioning state from %s to %s", c.State, newState)
	c.Container = c.Transition(newState)
	ctx.Tell(ctx.Self().Parent(), aproto.ContainerStateChanged{
		Container: c.Container, ContainerStarted: &started,
	})
}

// containerStopped transitions the container and sets the reason for stop. If called multiple
// times, it just respects and resends the first reason.
func (c *containerActor) containerStopped(ctx *actor.Context, msg aproto.ContainerStopped) {
	switch c.State {
	case cproto.Terminated:
		ctx.Log().
			WithField("called-because", msg).
			WithField("why", c.stop).
			Infof("retransmitting actions for transition to %s", cproto.Terminated)
	default:
		ctx.Log().
			WithField("why", msg).
			Infof("transitioning state from %s to %s", c.State, cproto.Terminated)
		c.Container = c.Transition(cproto.Terminated)
		c.stop = &msg
	}

	ctx.Tell(ctx.Self().Parent(), aproto.ContainerStateChanged{
		Container:        c.Container,
		ContainerStopped: c.stop,
	})
}
