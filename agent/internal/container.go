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
	containerIDEnvVar  = "DET_CONTAINER_ID"
	taskIDEnvVar       = "DET_TASK_ID"
	allocationIDEnvVar = "DET_ALLOCATION_ID"
)

type containerActor struct {
	cproto.Container
	spec          *cproto.Spec
	client        *client.Client
	docker        *actor.Ref
	containerInfo *types.ContainerJSON

	baseTaskLog model.TaskLog
	reattached  bool
}

type (
	getContainerSummary struct{}
	containerReady      struct{}
)

func newContainerActor(msg aproto.StartContainer, client *client.Client) actor.Actor {
	return &containerActor{Container: msg.Container, spec: &msg.Spec, client: client}
}

func reattachContainerActor(container cproto.Container, client *client.Client) actor.Actor {
	return &containerActor{Container: container, client: client, reattached: true}
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
		value := split[1]
		switch split[0] {
		case agentIDEnvVar:
			log.AgentID = &value
		case containerIDEnvVar:
			log.ContainerID = &value
		case taskIDEnvVar:
			log.TaskID = value
		case allocationIDEnvVar:
			log.AllocationID = ptrs.StringPtr(value)
		}
	}
	return log
}

func (c *containerActor) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		c.docker, _ = ctx.ActorOf("docker", &dockerActor{Client: c.client})
		if !c.reattached {
			c.transition(ctx, cproto.Pulling)
			pull := pullImage{PullSpec: c.spec.PullSpec, Name: c.spec.RunSpec.ContainerConfig.Image}
			ctx.Tell(c.docker, pull)
			c.baseTaskLog = getBaseTaskLog(c.spec)
		} else {
			ctx.Tell(c.docker, reattachContainer{ID: c.Container.ID})
		}
	case getContainerSummary:
		ctx.Respond(c.Container)

	case imagePulled:
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

	case containerReady:
		c.containerStarted(ctx, aproto.ContainerStarted{ContainerInfo: *c.containerInfo})

	case containerTerminated:
		c.containerStopped(ctx, aproto.ContainerExited(msg.ExitCode))
		ctx.Self().Stop()

	case aproto.SignalContainer:
		switch c.State {
		case cproto.Assigned, cproto.Pulling:
			switch msg.Signal {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL:
				ctx.Log().Infof("attempting to stop container while in [%s] state", c.State)
				ctx.Self().Stop()
				c.containerStopped(ctx, aproto.ContainerStopped{})
			default:
				ctx.Log().Warnf(
					"ignoring signal, signal not supported while container in [%s] state", c.State)
			}
		case cproto.Starting:
			// Delay signaling the container until the container is actually running.
			ctx.Tell(ctx.Self(), msg)
		case cproto.Running:
			ctx.Log().Infof("sending signal to container: %s", msg.Signal)
			ctx.Tell(c.docker, signalContainer{dockerID: c.containerInfo.ID, signal: msg.Signal})
		case cproto.Terminated:
			ctx.Log().Warnf("ignoring signal, container already terminated: %s", msg.Signal)
		}

	case aproto.ContainerLog:
		msg.Container = c.Container
		ctx.Log().Debug(msg)
		if c.spec.RunSpec.UseFluentLogging {
			ctx.Tell(ctx.Self().Parent(), c.makeTaskLog(msg))
		} else {
			ctx.Tell(ctx.Self().Parent(), msg)
		}
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
		Container: c.Container, ContainerStarted: &started})
}

func (c *containerActor) containerStopped(ctx *actor.Context, stopped aproto.ContainerStopped) {
	newState := cproto.Terminated
	if c.State == newState {
		ctx.Log().Infof("attempted transition of state from %s to %s", newState, newState)
		return
	}

	ctx.Log().Infof("transitioning state from %s to %s", c.State, newState)
	c.Container = c.Transition(newState)
	ctx.Tell(ctx.Self().Parent(), aproto.ContainerStateChanged{
		Container: c.Container, ContainerStopped: &stopped})
}
