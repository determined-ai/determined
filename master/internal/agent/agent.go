package agent

import (
	"net/http"
	"sort"
	"strings"
	"syscall"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	ws "github.com/determined-ai/determined/master/pkg/actor/api"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	proto "github.com/determined-ai/determined/proto/pkg/apiv1"
)

type (
	agent struct {
		address          string
		resourcePool     *actor.Ref
		socket           *actor.Ref
		slots            *actor.Ref
		containers       map[container.ID]*actor.Ref
		resourcePoolName string
		label            string
		// enabled and draining are duplicated in resourcepool agentState.
		// TODO(ilia): refactor/dedupe it.
		enabled  bool
		draining bool

		// awaitingReconnect et al contain reconnect related state. The pattern for
		// reconnecting agents is
		//  * They have a small window to reconnect.
		//  * In the meantime, we store up the messages it still can receive. We buffer and replay
		//    such that things go out in the order they always would have. This is critical since
		//    Start/Kill messages don't commute.
		//  * We deny state changes while in recovery for simplicity. Within a bounded time, it will
		//    recover or die.
		//  * If the agent reconnects within the deadline, great. We replay the messages and move on.
		//  * If it doesn't we stop with an error. If it comes back to reconnect later (only by some
		//    monumental clock skew), the agent manager shoos it away, telling it to restart.
		// Because of all this, for future developers: messages must be replay-able and writes must
		// get buffered while down.
		awaitingReconnect bool
		reconnectBacklog  []interface{}

		// uuid is an anonymous ID that is used when reporting telemetry
		// information to allow agent connection and disconnection events
		// to be correlated.
		uuid uuid.UUID

		// opts are additional agent options the master sends to the agent.
		opts *aproto.MasterSetAgentOptions
	}

	reconnectTimeout struct{}
)

var errRecovering = errors.New("agent disconnected, wait for recovery")

func (a *agent) Receive(ctx *actor.Context) error {
	return a.receive(ctx, ctx.Message())
}

func (a *agent) receive(ctx *actor.Context, msg interface{}) error {
	switch msg := msg.(type) {
	case actor.PreStart:
		a.uuid = uuid.New()
		a.slots, _ = ctx.ActorOf("slots", &slots{resourcePool: a.resourcePool})
		a.containers = make(map[container.ID]*actor.Ref)
	case model.AgentSummary:
		ctx.Respond(a.summarize(ctx))
	case ws.WebSocketConnected:
		check.Panic(check.True(a.socket == nil, "websocket already connected"))
		socket, ok := msg.Accept(ctx, aproto.MasterMessage{}, true)
		check.Panic(check.True(ok, "failed to accept websocket connection"))
		a.socket = socket
		lastColonIndex := strings.LastIndex(msg.Ctx.Request().RemoteAddr, ":")
		if lastColonIndex == -1 {
			a.address = msg.Ctx.Request().RemoteAddr
		} else {
			a.address = msg.Ctx.Request().RemoteAddr[0:lastColonIndex]
		}
		wsm := ws.WriteMessage{Message: aproto.AgentMessage{MasterSetAgentOptions: a.opts}}
		if err := ctx.Ask(a.socket, wsm).Error(); err != nil {
			ctx.Log().WithError(err).Error("failed to write master set agent options")
		}
		if a.awaitingReconnect {
			ctx.Log().Info("agent reconnected")
			a.awaitingReconnect = false
			for msg := range a.reconnectBacklog {
				if err := a.receive(ctx, msg); err != nil {
					return errors.Wrapf(err, "replaying backlog")
				}
			}
			a.reconnectBacklog = nil
		}
	case sproto.KillTaskContainer:
		if a.awaitingReconnect {
			a.bufferForRecovery(ctx, msg)
			return nil
		}

		ctx.Log().Infof("killing container id: %s", msg.ContainerID)
		killMsg := aproto.SignalContainer{
			ContainerID: msg.ContainerID, Signal: syscall.SIGKILL,
		}
		wsm := ws.WriteMessage{Message: aproto.AgentMessage{SignalContainer: &killMsg}}
		if err := ctx.Ask(a.socket, wsm).Error(); err != nil {
			ctx.Log().WithError(err).Error("failed to write kill task message")
		}
	case aproto.SignalContainer:
		if a.awaitingReconnect {
			a.bufferForRecovery(ctx, msg)
			return nil
		}

		wsm := ws.WriteMessage{Message: aproto.AgentMessage{SignalContainer: &msg}}
		if err := ctx.Ask(a.socket, wsm).Error(); err != nil {
			ctx.Log().WithError(err).Error("failed to write signal container message")
		}
	case sproto.StartTaskContainer:
		if a.awaitingReconnect {
			a.bufferForRecovery(ctx, msg)
			return nil
		}

		ctx.Log().Infof("starting container id: %s slots: %d task handler: %s",
			msg.StartContainer.Container.ID, len(msg.StartContainer.Container.Devices),
			msg.TaskActor.Address())

		wsm := ws.WriteMessage{Message: aproto.AgentMessage{StartContainer: &msg.StartContainer}}
		if err := ctx.Ask(a.socket, wsm).Error(); err != nil {
			// TODO(DET-5862): After push arch, return and handle this error when starting allocations.
			ctx.Log().WithError(err).Error("failed to write start container message")
		}
		ctx.Tell(a.slots, msg.StartContainer)
		a.containers[msg.Container.ID] = msg.TaskActor
	case aproto.MasterMessage:
		a.handleIncomingWSMessage(ctx, msg)
	case *proto.GetAgentRequest:
		ctx.Respond(&proto.GetAgentResponse{Agent: a.summarize(ctx).ToProto()})
	case *proto.GetSlotsRequest:
		var slots []*agentv1.Slot
		for _, s := range a.summarize(ctx).Slots {
			slots = append(slots, s.ToProto())
		}
		sort.Slice(slots, func(i, j int) bool { return slots[i].Id < slots[j].Id })
		ctx.Respond(&proto.GetSlotsResponse{Slots: slots})
	case *proto.EnableAgentRequest:
		if a.awaitingReconnect {
			ctx.Respond(errRecovering)
			return nil
		}

		ctx.Tell(a.slots, patchSlot{Enabled: true})
		ctx.Tell(a.resourcePool, sproto.EnableAgent{Agent: ctx.Self()})
		ctx.Respond(&proto.EnableAgentResponse{Agent: a.summarize(ctx).ToProto()})
	case *proto.DisableAgentRequest:
		if a.awaitingReconnect {
			ctx.Respond(errRecovering)
			return nil
		}

		// Mark current agent as disabled with RP.
		ctx.Tell(a.resourcePool, sproto.DisableAgent{Agent: ctx.Self(), Drain: msg.Drain})
		// Update individual slot state.
		ctx.Tell(a.slots, patchSlot{Enabled: false, Drain: msg.Drain})
		// Kill both slotted and zero-slot tasks, unless draining.
		if !msg.Drain {
			for cid := range a.containers {
				// TODO(DET-5916): This kill should not count towards max_restarts.
				ctx.Tell(ctx.Self(), sproto.KillTaskContainer{ContainerID: cid})
			}
		}
		ctx.Respond(&proto.DisableAgentResponse{Agent: a.summarize(ctx).ToProto()})
	case echo.Context:
		a.handleAPIRequest(ctx, msg)
	case actor.ChildFailed:
		ctx.Log().WithError(msg.Error).Errorf("child failed, awaiting reconnect: %s", msg.Child.Address())
		a.socket = nil
		a.awaitingReconnect = true
		actors.NotifyAfter(ctx, aproto.AgentReconnectWait, reconnectTimeout{})
	case reconnectTimeout:
		// Re-enter from actor.ChildFailed.
		if a.awaitingReconnect {
			telemetry.ReportAgentDisconnected(ctx.Self().System(), a.uuid)
			return errors.New("agent failed to reconnect by deadline")
		}
	case actor.ChildStopped:
		telemetry.ReportAgentDisconnected(ctx.Self().System(), a.uuid)
		ctx.Self().Stop()
	case actor.PostStop:
		ctx.Log().Infof("agent disconnected")
		for cid := range a.containers {
			stopped := aproto.ContainerError(
				aproto.AgentFailed, errors.New("agent closed with allocated containers"))
			a.containerStateChanged(ctx, aproto.ContainerStateChanged{
				Container: container.Container{
					ID:    cid,
					State: container.Terminated,
				},
				ContainerStopped: &stopped,
			})
		}
		ctx.Tell(a.resourcePool, sproto.RemoveAgent{Agent: ctx.Self()})
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agent) bufferForRecovery(ctx *actor.Context, msg interface{}) {
	ctx.Log().WithField("msg", msg).Debugf("buffering message until agent reconnects")
	a.reconnectBacklog = append(a.reconnectBacklog, msg)
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
		ctx.Log().Infof("agent connected ip: %v resource pool: %s slots: %d",
			a.address, a.resourcePoolName, len(msg.AgentStarted.Devices))

		ctx.Tell(a.resourcePool, sproto.AddAgent{Agent: ctx.Self(), Label: msg.AgentStarted.Label})
		ctx.Tell(a.slots, *msg.AgentStarted)
		a.label = msg.AgentStarted.Label
	case msg.ContainerStateChanged != nil:
		a.containerStateChanged(ctx, *msg.ContainerStateChanged)
	case msg.ContainerLog != nil:
		ref, ok := a.containers[msg.ContainerLog.Container.ID]
		check.Panic(check.True(ok,
			"container not allocated to agent: container %s", msg.ContainerLog.Container.ID))
		ctx.Tell(ref, sproto.ContainerLog{
			Container:   msg.ContainerLog.Container,
			Timestamp:   msg.ContainerLog.Timestamp,
			PullMessage: msg.ContainerLog.PullMessage,
			RunMessage:  msg.ContainerLog.RunMessage,
			AuxMessage:  msg.ContainerLog.AuxMessage,
		})
	default:
		check.Panic(errors.Errorf("error parsing incoming message"))
	}
}

func (a *agent) containerStateChanged(ctx *actor.Context, sc aproto.ContainerStateChanged) {
	taskActor, ok := a.containers[sc.Container.ID]
	check.Panic(check.True(ok, "container not allocated to agent: container %s", sc.Container.ID))

	rsc := sproto.TaskContainerStateChanged{Container: sc.Container}
	switch sc.Container.State {
	case container.Running:
		if sc.ContainerStarted.ProxyAddress == "" {
			sc.ContainerStarted.ProxyAddress = a.address
		}
		rsc.ContainerStarted = &sproto.TaskContainerStarted{
			Addresses: sc.ContainerStarted.Addresses(),
		}
	case container.Terminated:
		ctx.Log().
			WithError(sc.ContainerStopped.Failure).
			Infof("container %s terminated", sc.Container.ID)
		delete(a.containers, sc.Container.ID)
		rsc.ContainerStopped = &sproto.TaskContainerStopped{
			ContainerStopped: *sc.ContainerStopped,
		}
	}

	ctx.Tell(taskActor, rsc)
	ctx.Tell(a.slots, sc)
}

func (a *agent) summarize(ctx *actor.Context) model.AgentSummary {
	return model.AgentSummary{
		ID:             ctx.Self().Address().Local(),
		RegisteredTime: ctx.Self().RegisteredTime(),
		Slots:          ctx.Ask(a.slots, model.SlotsSummary{}).Get().(model.SlotsSummary),
		NumContainers:  len(a.containers),
		ResourcePool:   a.resourcePoolName,
		Label:          a.label,
		Addresses:      []string{a.address},
		Enabled:        a.enabled,
		Draining:       a.draining,
	}
}
