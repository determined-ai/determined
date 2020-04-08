package agent

import (
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	ws "github.com/determined-ai/determined/master/pkg/actor/api"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
)

type agent struct {
	address    string
	cluster    *actor.Ref
	socket     *actor.Ref
	slots      *actor.Ref
	containers map[container.ID]*actor.Ref
	label      string

	// uuid is an anonymous ID that is used when reporting telemetry
	// information to allow agent connection and disconnection events
	// to be correlated.
	uuid uuid.UUID
}

type agentSummary struct {
	ID             string       `json:"id"`
	RegisteredTime time.Time    `json:"registered_time"`
	Slots          slotsSummary `json:"slots"`
	Label          string       `json:"label"`
}

func (a *agent) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		a.uuid = uuid.New()
		a.slots, _ = ctx.ActorOf("slots", &slots{cluster: a.cluster})
		a.containers = make(map[container.ID]*actor.Ref)
	case agentSummary:
		ctx.Respond(a.summarize(ctx))
	case ws.WebSocketConnected:
		check.Panic(check.True(a.socket == nil, "websocket already connected"))
		socket, ok := msg.Accept(ctx, aproto.MasterMessage{}, true)
		check.Panic(check.True(ok, "failed to accept websocket connection"))
		a.socket = socket
		a.address = strings.SplitN(msg.Ctx.Request().RemoteAddr, ":", 2)[0]
	case aproto.SignalContainer:
		ctx.Ask(a.socket, ws.WriteMessage{Message: aproto.AgentMessage{SignalContainer: &msg}})
	case scheduler.StartTask:
		start := ws.WriteMessage{Message: aproto.AgentMessage{StartContainer: &msg.StartContainer}}
		ctx.Ask(a.socket, start)
		ctx.Tell(a.slots, msg.StartContainer)
		a.containers[msg.Container.ID] = msg.Task
	case aproto.MasterMessage:
		a.handleIncomingWSMessage(ctx, msg)
	case echo.Context:
		a.handleAPIRequest(ctx, msg)
	case actor.ChildFailed:
		telemetry.ReportAgentDisconnected(ctx.Self().System(), a.uuid)

		return errors.Wrapf(msg.Error, "child failed: %s", msg.Child.Address())
	case actor.PostStop:
		for cid := range a.containers {
			stopped := aproto.ContainerError(
				aproto.AgentFailed, errors.New("agent failed while container was running"))
			a.containerStateChanged(ctx, aproto.ContainerStateChanged{
				Container: container.Container{
					ID:    cid,
					State: container.Terminated,
				},
				ContainerStopped: &stopped,
			})
		}
		ctx.Tell(a.cluster, scheduler.RemoveAgent{Agent: ctx.Self()})
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agent) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, a.summarize(ctx)))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (a *agent) handleIncomingWSMessage(ctx *actor.Context, msg aproto.MasterMessage) {
	switch {
	case msg.AgentStarted != nil:
		telemetry.ReportAgentConnected(ctx.Self().System(), a.uuid, msg.AgentStarted.Devices)

		ctx.Tell(a.cluster, scheduler.AddAgent{Agent: ctx.Self(), Label: msg.AgentStarted.Label})
		ctx.Tell(a.slots, *msg.AgentStarted)
		a.label = msg.AgentStarted.Label
		for _, c := range msg.AgentStarted.RecoveredContainers {
			ctx.Log().Infof("attempting to recover container master state: %s", c.ID)
			if ref := ctx.Self().System().Get(c.Parent); ref != nil {
				a.containers[c.ID] = ref
			}
			kill := aproto.AgentMessage{SignalContainer: &aproto.SignalContainer{
				ContainerID: c.ID,
				Signal:      syscall.SIGKILL,
			}}
			ctx.Ask(a.socket, ws.WriteMessage{Message: kill})
		}
	case msg.ContainerStateChanged != nil:
		a.containerStateChanged(ctx, *msg.ContainerStateChanged)
	case msg.ContainerLog != nil:
		ref, ok := a.containers[msg.ContainerLog.Container.ID]
		if !ok {
			// We may get logs back for containers that were recovered but the task might already
			// be dead so we ignore the logs.
			break
		}
		ctx.Tell(ref, *msg.ContainerLog)
	default:
		check.Panic(errors.Errorf("error parsing incoming message"))
	}
}

func (a *agent) containerStateChanged(ctx *actor.Context, sc aproto.ContainerStateChanged) {
	task, ok := a.containers[sc.Container.ID]
	if !ok {
		ctx.Log().Warnf(
			"container transitioning to %s not assigned to agent: container %s",
			sc.Container.State, sc.Container.ID)
		return
	}
	switch sc.Container.State {
	case container.Running:
		if sc.ContainerStarted.ProxyAddress == "" {
			sc.ContainerStarted.ProxyAddress = a.address
		}
	case container.Terminated:
		delete(a.containers, sc.Container.ID)
	}
	ctx.Tell(task, sc)
	ctx.Tell(a.slots, sc)
	ctx.Tell(a.cluster, sc)
}

func (a *agent) summarize(ctx *actor.Context) agentSummary {
	return agentSummary{
		ID:             ctx.Self().Address().Local(),
		RegisteredTime: ctx.Self().RegisteredTime(),
		Slots:          ctx.Ask(a.slots, slotsSummary{}).Get().(slotsSummary),
		Label:          a.label,
	}
}
