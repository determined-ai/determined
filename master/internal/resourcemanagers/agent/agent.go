package agent

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	ws "github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	proto "github.com/determined-ai/determined/proto/pkg/apiv1"
)

type (
	agent struct {
		address          string
		resourcePool     *actor.Ref
		socket           *actor.Ref
		slots            *actor.Ref
		resourcePoolName string
		// started tracks if we have received the AgentStarted message.
		started bool
		version string

		// TODO(ilia): Maybe maxZeroSlotContainers should be an attribute of a resource pool,
		// and not be copied to agents.
		maxZeroSlotContainers int
		agentReconnectWait    time.Duration
		agentReattachEnabled  bool
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
		reconnectTimers   []*actor.Ref
		// On disconnect, we stash the state here and become "draining + disabled". Upon reconnect, we
		// pop back to our previous state.
		preDisconnectEnabled  bool
		preDisconnectDraining bool

		// opts are additional agent options the master sends to the agent.
		opts *aproto.MasterSetAgentOptions

		agentState *AgentState
	}

	reconnectTimeout struct{}

	// GetAgentState response is agent.agentState.
	GetAgentState struct{}
	// PatchAllSlotsState updates the state of all slots.
	PatchAllSlotsState struct {
		Enabled *bool
		Drain   *bool
	}
	// PatchSlotState updates the state of the target slot.
	PatchSlotState struct {
		ID      device.ID
		Enabled *bool
		Drain   *bool
	}
	// AllocateFreeDevices calls agentState.AllocateFreeDevices.
	AllocateFreeDevices struct {
		Slots       int
		ContainerID cproto.ID
	}
	// AllocateFreeDevicesResponse is a response to AllocateFreeDevices.
	AllocateFreeDevicesResponse struct {
		Devices []device.Device
	}
	// DeallocateContainer calls agentState.DeallocateContainer.
	DeallocateContainer struct {
		ContainerID cproto.ID
	}
)

var errRecovering = errors.New("agent disconnected, wait for recovery")

func (a *agent) Receive(ctx *actor.Context) error {
	return a.receive(ctx, ctx.Message())
}

func (a *agent) receive(ctx *actor.Context, msg interface{}) error {
	switch msg := msg.(type) {
	case actor.PreStart:
		if a.agentState != nil { // not nil agentState on PreStart means it's restored.
			a.started = true
			a.agentState.Handler = ctx.Self()
			// Update maxZeroSlotContainers config setting.
			a.agentState.maxZeroSlotContainers = a.maxZeroSlotContainers
			a.socketDisconnected(ctx)
			// TODO(ilia): Adding restored agent here will overcount AgentStarts by maximum
			// agentReconnectWait if it never reconnects.
			ctx.Tell(a.resourcePool, sproto.AddAgent{Agent: ctx.Self(), Label: a.agentState.Label})
		}
		a.slots, _ = ctx.ActorOf("slots", &slots{})
	case model.AgentSummary:
		ctx.Respond(a.summarize(ctx))
	case ws.WebSocketConnected:
		check.Panic(check.True(a.socket == nil, "websocket already connected"))
		socket, ok := msg.Accept(ctx, aproto.MasterMessage{}, true)
		check.Panic(check.True(ok, "failed to accept websocket connection"))
		a.socket = socket
		a.version = msg.Ctx.QueryParam("version")

		lastColonIndex := strings.LastIndex(msg.Ctx.Request().RemoteAddr, ":")
		if lastColonIndex == -1 {
			a.address = msg.Ctx.Request().RemoteAddr
		} else {
			a.address = msg.Ctx.Request().RemoteAddr[0:lastColonIndex]
		}

		var masterSetAgentOptions aproto.AgentMessage
		fmt.Println("websocketconnected", a.awaitingReconnect, a.agentReattachEnabled)
		// For reconnecting containers, or when reattach is enabled, try to synchronize containers.
		// Flush them otherwise.
		reconnect, _ := msg.IsReconnect()
		if a.awaitingReconnect && (a.agentReattachEnabled || reconnect) {
			optsCopy := *a.opts
			optsCopy.ContainersToReattach = a.gatherContainersToReattach(ctx)
			fmt.Println("containersToReattach", optsCopy.ContainersToReattach)
			masterSetAgentOptions = aproto.AgentMessage{MasterSetAgentOptions: &optsCopy}
		} else {
			masterSetAgentOptions = aproto.AgentMessage{MasterSetAgentOptions: a.opts}
		}

		wsm := ws.WriteMessage{Message: masterSetAgentOptions}
		if err := ctx.Ask(a.socket, wsm).Error(); err != nil {
			ctx.Log().WithError(err).Error("failed to write master set agent options")
		}

		if a.awaitingReconnect {
			ctx.Log().Info("agent reconnected")
			a.awaitingReconnect = false

			// Cancel reconnect timers.
			for _, timerActor := range a.reconnectTimers {
				timerActor.Stop()
			}
			a.reconnectTimers = nil

			// Re-propagate our old state back on successful recovery.
			if a.preDisconnectEnabled {
				a.agentState.Enable(ctx)
			} else {
				a.agentState.Disable(ctx, a.preDisconnectDraining)
			}
			a.agentState.patchAllSlotsState(ctx, PatchAllSlotsState{
				Enabled: &a.agentState.enabled,
				Drain:   &a.agentState.draining,
			})

			if len(a.reconnectBacklog) > 0 {
				for i, msg := range a.reconnectBacklog {
					ctx.Log().Debugf("will replay reconnectBacklog %d %s", i, reflect.TypeOf(msg))
				}
			}

			for _, msg := range a.reconnectBacklog {
				if err := a.receive(ctx, msg); err != nil {
					ctx.Log().WithError(err).WithField("msg", msg).Errorf("replaying backlog")
					return errors.Wrapf(err, "replaying backlog")
				}
			}
			a.reconnectBacklog = nil
			ctx.Tell(a.resourcePool, sproto.UpdateAgent{Agent: ctx.Self()})
		}
	case sproto.KillTaskContainer:
		if a.awaitingReconnect {
			a.bufferForRecovery(ctx, msg)
			return nil
		}

		log := ctx.Log().
			WithFields(msg.LogContext.Fields()).
			WithField("container-id", msg.ContainerID)
		log.Infof("killing container")

		killMsg := aproto.SignalContainer{
			ContainerID: msg.ContainerID, Signal: syscall.SIGKILL,
		}
		wsm := ws.WriteMessage{Message: aproto.AgentMessage{SignalContainer: &killMsg}}
		if err := ctx.Ask(a.socket, wsm).Error(); err != nil {
			log.WithError(err).Error("failed to write kill task message")
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
		log := ctx.Log().
			WithFields(msg.LogContext.Fields()).
			WithField("container-id", msg.StartContainer.Container.ID).
			WithField("slots", len(msg.StartContainer.Container.Devices))
		log.Infof("starting container")

		wsm := ws.WriteMessage{Message: aproto.AgentMessage{StartContainer: &msg.StartContainer}}
		if err := ctx.Ask(a.socket, wsm).Error(); err != nil {
			// TODO(DET-5862): After push arch, return and handle this error when starting allocations.
			log.WithError(err).Error("failed to write start container message")
			ctx.Respond(sproto.NewResourcesFailure(sproto.AgentError, err.Error(), nil))
		}

		if err := a.agentState.startContainer(ctx, msg); err != nil {
			log.WithError(err).Error("failed to update agent state")
		}
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

		if !a.started {
			ctx.Respond(errors.New("can't enable agent: agent not started"))
			return nil
		}

		a.agentState.Enable(ctx)
		a.agentState.patchAllSlotsState(ctx, PatchAllSlotsState{
			Enabled: &a.agentState.enabled,
			Drain:   &a.agentState.draining,
		})
		ctx.Respond(&proto.EnableAgentResponse{Agent: a.summarize(ctx).ToProto()})
		ctx.Tell(a.resourcePool, sproto.UpdateAgent{Agent: ctx.Self()})
	case *proto.DisableAgentRequest:
		if a.awaitingReconnect {
			ctx.Respond(errRecovering)
			return nil
		}

		if !a.started {
			ctx.Respond(errors.New("can't disable agent: agent not started"))
			return nil
		}

		// Mark current agent as disabled with RP.
		a.agentState.Disable(ctx, msg.Drain)
		// Update individual slot state.
		a.agentState.patchAllSlotsState(ctx, PatchAllSlotsState{
			Enabled: &a.agentState.enabled,
			Drain:   &a.agentState.draining,
		})
		// Kill both slotted and zero-slot tasks, unless draining.
		if !msg.Drain {
			for cid := range a.agentState.containerAllocation {
				ctx.Tell(a.agentState.containerAllocation[cid], task.Kill)
			}
		}
		ctx.Respond(&proto.DisableAgentResponse{Agent: a.summarize(ctx).ToProto()})
		ctx.Tell(a.resourcePool, sproto.UpdateAgent{Agent: ctx.Self()})
	case echo.Context:
		a.handleAPIRequest(ctx, msg)
	case actor.ChildFailed:
		if !a.started {
			// If we happen to fail before the agent has started and been registered with
			// the resource manager, then nothing can be running on it. In this case we
			// just fail outright and make it restart.
			return errors.Wrapf(msg.Error, "child failed: %s", msg.Child.Address())
		}

		ctx.Log().WithError(msg.Error).Errorf("child failed, awaiting reconnect: %s", msg.Child.Address())

		a.socketDisconnected(ctx)
		ctx.Tell(a.resourcePool, sproto.UpdateAgent{Agent: ctx.Self()})
	case reconnectTimeout:
		// Re-enter from actor.ChildFailed.
		if a.awaitingReconnect {
			return errors.New("agent failed to reconnect by deadline")
		}
	case GetAgentState:
		if !a.started {
			ctx.Respond(errors.New("agent state is not available: agent not started"))
			return nil
		}

		ctx.Respond(a.agentState.DeepCopy())
	case PatchSlotState:
		if !a.started {
			ctx.Respond(errors.New("can't patch slot state: agent not started"))
			return nil
		}

		result, err := a.agentState.patchSlotState(ctx, msg)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(result)
	case PatchAllSlotsState:
		if !a.started {
			ctx.Respond(errors.New("can't patch slots state: agent not started"))
			return nil
		}

		ctx.Respond(a.agentState.patchAllSlotsState(ctx, msg))
	case AllocateFreeDevices:
		if !a.started {
			ctx.Log().Debugf("received AllocateFreeDevices on non-started agent")
			ctx.Respond(errors.New("can't allocate free devices: agent not started"))
			return nil
		}
		devices, err := a.agentState.AllocateFreeDevices(msg.Slots, msg.ContainerID)
		if err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(AllocateFreeDevicesResponse{
				Devices: devices,
			})
		}
	case DeallocateContainer:
		if !a.started {
			ctx.Respond(errors.New("can't deallocate container: agent not started"))
			return nil
		}
		a.agentState.DeallocateContainer(msg.ContainerID)
	case model.SlotsSummary:
		if !a.started {
			ctx.Respond(model.SlotsSummary{})
			return nil
		}

		ctx.Respond(a.agentState.getSlotsSummary(ctx))
	case actor.ChildStopped:
		ctx.Self().Stop()
	case actor.PostStop:
		ctx.Log().Infof("agent disconnected")
		if a.started {
			for cid := range a.agentState.containerAllocation {
				stopped := aproto.ContainerError(
					aproto.AgentFailed, errors.New("agent closed with allocated containers"))
				a.containerStateChanged(ctx, aproto.ContainerStateChanged{
					Container: cproto.Container{
						ID:    cid,
						State: cproto.Terminated,
					},
					ContainerStopped: &stopped,
				})
			}

			if err := a.agentState.delete(); err != nil {
				ctx.Log().WithError(err).Warnf("failed to delete agent state")
			}
		}
		ctx.Tell(a.resourcePool, sproto.RemoveAgent{Agent: ctx.Self()})
	default:
		fmt.Println("UNEXPECTED MESSAGE", msg, reflect.TypeOf(msg))
		debug.PrintStack()
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agent) bufferForRecovery(ctx *actor.Context, msg interface{}) {
	// This explodes the debug logs when msg is big
	// ctx.Log().WithField("msg", msg).Debugf("buffering message until agent reconnects")
	ctx.Log().WithField("msg-type", reflect.TypeOf(msg)).
		Debugf("buffering message until agent reconnects")
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
		ctx.Log().Infof("agent connected ip: %v resource pool: %s slots: %d",
			a.address, a.resourcePoolName, len(msg.AgentStarted.Devices))

		// TODO(ilia): Error out on a change in devices.
		if !a.started {
			a.agentStarted(ctx, msg.AgentStarted)
		}

		a.started = true

		if err := a.handleContainersReattached(ctx, msg.AgentStarted); err != nil {
			ctx.Log().
				WithError(err).
				WithField("agent-id", ctx.Self().Address().Local()).
				Error("failure in handleContainersReattached")
		}
	case msg.ContainerStateChanged != nil:
		a.containerStateChanged(ctx, *msg.ContainerStateChanged)
	case msg.ContainerLog != nil:
		ref, ok := a.agentState.containerAllocation[msg.ContainerLog.Container.ID]
		if !ok {
			containerID := msg.ContainerLog.Container.ID
			ctx.Log().WithField("container-id", containerID).Warnf(
				"received ContainerLog from container not allocated to agent: "+
					"container %s, message: %v", containerID, msg.ContainerLog)
			return
		}
		ctx.Tell(ref, sproto.ContainerLog{
			Container:   msg.ContainerLog.Container,
			Timestamp:   msg.ContainerLog.Timestamp,
			PullMessage: msg.ContainerLog.PullMessage,
			RunMessage:  msg.ContainerLog.RunMessage,
			AuxMessage:  msg.ContainerLog.AuxMessage,
		})
	case msg.ContainerStatsRecord != nil:
		if a.taskNeedsRecording(msg.ContainerStatsRecord) {
			var err error
			if msg.ContainerStatsRecord.EndStats {
				err = db.RecordTaskEndStatsBun(msg.ContainerStatsRecord.Stats)
			} else {
				err = db.RecordTaskStatsBun(msg.ContainerStatsRecord.Stats)
			}
			if err != nil {
				ctx.Log().Errorf("Error record task stats %s", err)
			}
		}

	default:
		check.Panic(errors.Errorf("error parsing incoming message"))
	}
}

func (a *agent) taskNeedsRecording(record *aproto.ContainerStatsRecord) bool {
	return record.TaskType == model.TaskTypeTrial
}

func (a *agent) agentStarted(ctx *actor.Context, agentStarted *aproto.AgentStarted) {
	a.agentState = NewAgentState(
		sproto.AddAgent{Agent: ctx.Self(), Label: agentStarted.Label},
		a.maxZeroSlotContainers)
	a.agentState.resourcePoolName = a.resourcePoolName // TODO that's where it gets set
	a.agentState.agentStarted(ctx, agentStarted)
	ctx.Tell(a.resourcePool, sproto.AddAgent{
		Agent: ctx.Self(),
		Label: agentStarted.Label,
		Slots: a.agentState.NumSlots()})

	// TODO(ilia): Deprecate together with the old slots API.
	ctx.Tell(a.slots, *agentStarted)
}

func (a *agent) containerStateChanged(ctx *actor.Context, sc aproto.ContainerStateChanged) {
	taskActor, ok := a.agentState.containerAllocation[sc.Container.ID]

	if !ok {
		// We may receieve late terminations when reconnected agent is cleaning up
		// terminated containers.
		if sc.Container.State != cproto.Terminated {
			containerID := sc.Container.ID
			ctx.Log().WithField("container-id", containerID).Warnf(
				"received ContainerStateChanged from container not allocated to agent: "+
					"container %s, message: %v", containerID, sc)
		}
		return
	}

	switch sc.Container.State {
	case cproto.Running:
		if sc.ContainerStarted.ProxyAddress == "" {
			sc.ContainerStarted.ProxyAddress = a.address
		}
	case cproto.Terminated:
		ctx.Log().
			WithError(sc.ContainerStopped.Failure).
			Infof("container %s terminated", sc.Container.ID)
		delete(a.agentState.containerAllocation, sc.Container.ID)
	}

	ctx.Tell(taskActor, sproto.FromContainerStateChanged(sc))
	a.agentState.containerStateChanged(ctx, sc)
}

func (a *agent) summarize(ctx *actor.Context) model.AgentSummary {
	if a.agentState != nil {
		fmt.Println("agent summarize debug")
		fmt.Println(maps.Keys(a.agentState.containerAllocation))
		fmt.Println(maps.Keys(a.agentState.containerState))
	}
	result := model.AgentSummary{
		ID:             ctx.Self().Address().Local(),
		RegisteredTime: ctx.Self().RegisteredTime(),
		ResourcePool:   a.resourcePoolName,
		Addresses:      []string{a.address},
		// Default dummy values if the AgentStarted hasn't been processed yet.
		// Client code expects `Slots` to always be present.
		Slots:         model.SlotsSummary{},
		Enabled:       true,
		Draining:      false,
		NumContainers: 0,
		Version:       a.version,
	}

	if a.agentState != nil {
		result.Slots = a.agentState.getSlotsSummary(ctx)
		result.Label = a.agentState.Label
		result.Enabled = a.agentState.enabled
		result.Draining = a.agentState.draining
		result.NumContainers = len(a.agentState.containerAllocation)
	}

	return result
}

func (a *agent) gatherContainersToReattach(ctx *actor.Context) []aproto.ContainerReattach {
	err := a.agentState.restoreContainersField()
	if err != nil {
		ctx.Log().WithError(err).Warn("failed restoreContainersField in gatherContainersToReattach")
	}
	result := make([]aproto.ContainerReattach, 0, len(a.agentState.containerAllocation))

	for _, container := range a.agentState.containerState {
		result = append(result, aproto.ContainerReattach{Container: *container})
	}

	fmt.Println("gatherContainersToReattach: ", result)
	return result
}

func (a *agent) handleContainersReattached(
	ctx *actor.Context, agentStarted *aproto.AgentStarted) error {
	ctx.Log().Debugf("agent ContainersRestored ip: %v , reattached: %v, allocations: %v",
		a.address, agentStarted.ContainersReattached, maps.Keys(a.agentState.containerState))

	recovered := map[cproto.ID]aproto.ContainerReattachAck{}

	for _, containerRestored := range agentStarted.ContainersReattached {
		if containerRestored.Failure != nil {
			ctx.Log().Infof(
				"agent failed to restore container: %v: %v",
				containerRestored.Container.ID, containerRestored.Failure.ErrMsg)
			continue
		}

		cid := containerRestored.Container.ID
		_, ok := a.agentState.containerAllocation[cid]
		if !ok {
			continue
		}

		if a.agentState.containerState[cid].State != containerRestored.Container.State {
			ctx.Log().Warnf(
				"reattached container %s has changed state: %s to %s",
				cid, a.agentState.containerState[cid].State,
				containerRestored.Container.State)
			continue
		}

		recovered[cid] = containerRestored
	}

	// Mark the rest as dead.
	return a.clearNonReattachedContainers(ctx, recovered)
}

func (a *agent) clearNonReattachedContainers(
	ctx *actor.Context, recovered map[cproto.ID]aproto.ContainerReattachAck) error {
	for cid, allocation := range a.agentState.containerAllocation {
		_, ok := recovered[cid]

		if !ok {
			errorMsg := "container cleaned up on reconnect"
			if a.agentReattachEnabled {
				errorMsg = "failed to reattach container on reconnect"
			}

			stopped := aproto.ContainerError(aproto.AgentFailed, errors.New(errorMsg))
			ctx.Log().Infof("killing container that didn't restore: %s", cid.String())

			resp := ctx.Ask(allocation, sproto.GetResourcesContainerState{
				ResourcesID: sproto.ResourcesID(cid),
			})
			switch {
			case resp.Error() != nil:
				ctx.Log().Warnf(
					"allocation GetTaskContainerState id: %s, got error: %s", cid, resp.Error())
			case resp.Get() == nil:
				ctx.Log().Warnf("allocation GetTaskContainerState id: %s, is nil", cid)
			default:
				containerState := resp.Get().(cproto.Container)
				containerState.State = cproto.Terminated

				a.containerStateChanged(ctx, aproto.ContainerStateChanged{
					Container:        containerState,
					ContainerStopped: &stopped,
				})
			}
		}
	}

	return a.agentState.clearUnlessRecovered(recovered)
}

func (a *agent) socketDisconnected(ctx *actor.Context) {
	a.socket = nil
	a.awaitingReconnect = true

	timerActor, _ := actors.NotifyAfter(ctx, a.agentReconnectWait, reconnectTimeout{})
	a.reconnectTimers = append(a.reconnectTimers, timerActor)

	a.preDisconnectEnabled = a.agentState.enabled
	a.preDisconnectDraining = a.agentState.draining
	// Mark ourselves as draining to avoid action on ourselves while we recover. While the
	// system is technically correct without this, it's better because we avoid any waste
	// effort scheduling things only to have them suffer AgentErrors later.
	a.agentState.Disable(ctx, true)
	a.agentState.patchAllSlotsState(ctx, PatchAllSlotsState{
		Enabled: &a.agentState.enabled,
		Drain:   &a.agentState.draining,
	})
	ctx.Tell(a.resourcePool, sproto.UpdateAgent{Agent: ctx.Self()})
}
