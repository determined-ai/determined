package agentrm

import (
	"fmt"
	"net"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
	"github.com/determined-ai/determined/master/pkg/ws"
	"github.com/determined-ai/determined/proto/pkg/agentv1"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

var errRecovering = errors.New("agent disconnected, wait for recovery")

type (
	agent struct {
		syslog     *logrus.Entry
		unregister func()

		mu sync.Mutex

		id               agentID
		registeredTime   time.Time
		address          string
		agentUpdates     *queue.Queue[agentUpdatedEvent]
		socket           *ws.WebSocket[*aproto.MasterMessage, aproto.AgentMessage]
		resourcePoolName string
		// started tracks if we have received the AgentStarted message.
		started bool
		version string

		// TODO(ilia): Maybe maxZeroSlotContainers should be an attribute of a resource pool,
		// and not be copied to agents.
		maxZeroSlotContainers int
		agentReconnectWait    time.Duration
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
		// awaitingRestore tracks the restoration of agentState.containerAllocation, which must be
		// restored after allocation refs start and register in allocationmap. It happens in during
		// websocket connection if the websocket does reconnect and in poststop if it does not.
		awaitingRestore  bool
		reconnectBacklog []interface{}
		reconnectTimers  []*time.Timer
		// On disconnect, we stash the state here and become "draining + disabled". Upon reconnect, we
		// pop back to our previous state.
		preDisconnectEnabled  bool
		preDisconnectDraining bool

		// opts are additional agent options the master sends to the agent.
		opts *aproto.MasterSetAgentOptions

		agentState *agentState
	}

	// patchAllSlotsState updates the state of all slots.
	patchAllSlotsState struct {
		enabled *bool
		drain   *bool
	}
	// patchSlotState updates the state of the target slot.
	patchSlotState struct {
		id      device.ID
		enabled *bool
		drain   *bool
	}
	// allocateFreeDevices calls agentState.allocateFreeDevices.
	allocateFreeDevices struct {
		slots       int
		containerID cproto.ID
	}
	// allocateFreeDevicesResponse is a response to allocateFreeDevices.
	allocateFreeDevicesResponse struct {
		devices []device.Device
	}
	// deallocateContainer calls agentState.deallocateContainer.
	deallocateContainer struct {
		containerID cproto.ID
	}
)

func newAgent(
	id agentID,
	agentUpdates *queue.Queue[agentUpdatedEvent],
	resourcePoolName string,
	rpConfig *config.ResourcePoolConfig,
	opts *aproto.MasterSetAgentOptions,
	restoredAgentState *agentState,
	unregister func(),
) *agent {
	a := &agent{
		syslog:                logrus.WithField("component", "agent").WithField("id", id),
		id:                    id,
		registeredTime:        time.Now(),
		agentUpdates:          agentUpdates,
		resourcePoolName:      resourcePoolName,
		maxZeroSlotContainers: rpConfig.MaxAuxContainersPerAgent,
		agentReconnectWait:    time.Duration(rpConfig.AgentReconnectWait),
		opts:                  opts,
		agentState:            restoredAgentState,
		unregister:            unregister,
	}

	if restoring := a.agentState != nil; restoring {
		a.started = true
		a.awaitingRestore = true
		a.agentState.handler = a
		// Update maxZeroSlotContainers config setting.
		a.agentState.maxZeroSlotContainers = a.maxZeroSlotContainers
		// TODO(ilia): Adding restored agent here will overcount AgentStarts by maximum
		// agentReconnectWait if it never reconnects.
		// Ensure RP is aware of the agent.
		a.syslog.Infof("adding agent: %s", a.agentState.agentID())
		err := a.updateAgentStartStats(a.resourcePoolName, string(a.id), a.agentState.numSlots())
		if err != nil {
			a.syslog.WithError(err).Error("failed to update agent start stats")
		}
		a.notifyListeners()
		a.socketDisconnected()
	}

	return a
}

func (a *agent) AllocateFreeDevices(msg allocateFreeDevices) (allocateFreeDevicesResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		a.syslog.Debugf("received allocateFreeDevices on non-started agent")
		return allocateFreeDevicesResponse{}, errors.New("can't allocate free devices: agent not started")
	}

	devices, err := a.agentState.allocateFreeDevices(msg.slots, msg.containerID)
	if err != nil {
		return allocateFreeDevicesResponse{}, err
	}
	return allocateFreeDevicesResponse{devices: devices}, nil
}

func (a *agent) DeallocateContainer(msg deallocateContainer) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		return errors.New("can't deallocate container: agent not started")
	}
	a.agentState.deallocateContainer(msg.containerID)
	return nil
}

func (a *agent) State() (*agentState, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		return nil, errors.New("agent state is not available: agent not started")
	}

	return a.agentState.deepCopy(), nil
}

func (a *agent) StartTaskContainer(msg sproto.StartTaskContainer) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.startTaskContainer(msg)
}

func (a *agent) startTaskContainer(msg sproto.StartTaskContainer) {
	if a.awaitingReconnect {
		a.bufferForRecovery(msg)
		return
	}
	log := a.syslog.
		WithFields(msg.LogContext.Fields()).
		WithField("container-id", msg.StartContainer.Container.ID).
		WithField("slots", len(msg.StartContainer.Container.Devices))
	log.Infof("starting container")

	a.socket.Outbox <- aproto.AgentMessage{StartContainer: &msg.StartContainer}

	if err := a.agentState.startContainer(msg); err != nil {
		log.WithError(err).Error("failed to update agent state")
	}
}

func (a *agent) KillTaskContainer(msg sproto.KillTaskContainer) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.killTaskContainer(msg)
}

func (a *agent) killTaskContainer(msg sproto.KillTaskContainer) {
	if a.awaitingReconnect {
		a.bufferForRecovery(msg)
		return
	}

	log := a.syslog.
		WithFields(msg.LogContext.Fields()).
		WithField("container-id", msg.ContainerID)
	log.Infof("killing container")

	killMsg := aproto.SignalContainer{
		ContainerID: msg.ContainerID, Signal: syscall.SIGKILL,
	}
	a.socket.Outbox <- aproto.AgentMessage{SignalContainer: &killMsg}
}

func (a *agent) stop(cause error) {
	defer a.unregister()

	if cause != nil {
		a.syslog.WithError(cause).WithFields(logrus.Fields{
			"address": a.address,
			"started": a.started,
		}).Error("agent crashed")
	}

	if a.started {
		// This normally will run on agent WebSocketRequest to populate
		// agentState.containerAllocation. There is technically still race here
		// (and also in calling this in WebSocketRequest). We have no synchronization
		// between allocation actors starting and registering themselves in the
		// allocationmap and the lookup of allocationmap in restoreContainersField().
		// Though this will likely run after agentReconnectWait which should
		// give enough time for this to be populated.
		// TODO: add explicit synchronization here.
		err := a.agentState.restoreContainersField()
		if err != nil {
			a.syslog.WithError(err).Error("failed restoreContainersField in shutdown of agent")
		}

		for cid := range a.agentState.containerAllocation {
			stopped := aproto.ContainerError(
				aproto.AgentFailed, errors.New("agent closed with allocated containers"))
			a.containerStateChanged(aproto.ContainerStateChanged{
				Container: cproto.Container{
					ID:    cid,
					State: cproto.Terminated,
				},
				ContainerStopped: &stopped,
			})
		}

		if err := a.agentState.delete(); err != nil {
			a.syslog.WithError(err).Warnf("failed to delete agent state")
		}
	} else {
		a.syslog.Info("agent disconnected but wasn't started")
	}

	if a.socket != nil {
		if err := a.socket.Close(); err != nil {
			a.syslog.WithError(err).Warnf("error while shutting down agent WebSocket")
		}
	}

	a.syslog.Infof("removing agent: %s", a.agentState.agentID())
	err := a.updateAgentEndStats(string(a.id))
	if err != nil {
		a.syslog.WithError(err).Error("failed to update agent end stats")
	}
	a.notifyListeners()
}

func (a *agent) HandleWebsocketConnection(msg webSocketRequest) {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.handleWebsocketConnection(msg)
	if err != nil {
		a.stop(err)
		return
	}
}

func (a *agent) handleWebsocketConnection(msg webSocketRequest) error {
	if a.socket != nil {
		err := errors.New("websocket already connected")
		a.syslog.WithError(err).Error("socket not nil when WebSocketRequest received")
		return err
	}

	conn, err := ws.UpgradeEchoConnection(msg.echoCtx)
	if err != nil {
		msg := "error upgrading connection to WebSocket"
		a.syslog.WithError(err).Error(msg)
		return errors.Wrap(err, msg)
	}

	wsName := "master-agent-ws-" + string(a.id)
	socket, err := ws.Wrap[*aproto.MasterMessage, aproto.AgentMessage](wsName, conn)
	if err != nil {
		msg := "failed to accept websocket connection"
		a.syslog.WithError(err).Error(msg)
		return errors.Wrap(err, msg)
	}

	// spin up goroutine that sends messages to self
	go func() {
		defer a.HandleWebsocketDisconnect()

		for {
			select {
			case msg := <-socket.Inbox:
				// If the Inbox has closed, we get a zero value
				if msg == nil {
					return
				}
				a.HandleIncomingWebsocketMessage(msg)
			case <-socket.Done:
				return
			}
		}
	}()

	a.socket = socket
	a.version = msg.echoCtx.QueryParam("version")

	lastColonIndex := strings.LastIndex(msg.echoCtx.Request().RemoteAddr, ":")
	if lastColonIndex == -1 {
		a.address = msg.echoCtx.Request().RemoteAddr
	} else {
		a.address = msg.echoCtx.Request().RemoteAddr[0:lastColonIndex]
	}

	a.adjustAgentIPAddrIfRunningDevClusterOnHpcUsingAnSSHTunnel(msg)

	var masterSetAgentOptions aproto.AgentMessage
	if a.awaitingReconnect {
		optsCopy := *a.opts
		optsCopy.ContainersToReattach = a.gatherContainersToReattach()
		masterSetAgentOptions = aproto.AgentMessage{MasterSetAgentOptions: &optsCopy}
	} else {
		masterSetAgentOptions = aproto.AgentMessage{MasterSetAgentOptions: a.opts}
	}

	if a.awaitingRestore {
		a.awaitingRestore = false
	}

	a.socket.Outbox <- masterSetAgentOptions

	if a.awaitingReconnect {
		a.syslog.Info("agent reconnected")
		a.awaitingReconnect = false

		// Cancel reconnect timers.
		for _, timerActor := range a.reconnectTimers {
			timerActor.Stop()
		}
		a.reconnectTimers = nil

		// Re-propagate our old state back on successful recovery.
		if a.preDisconnectEnabled {
			a.agentState.enable()
		} else {
			a.agentState.disable(a.preDisconnectDraining)
		}
		a.agentState.patchAllSlotsState(patchAllSlotsState{
			enabled: &a.agentState.enabled,
			drain:   &a.agentState.draining,
		})

		if len(a.reconnectBacklog) > 0 {
			for i, msg := range a.reconnectBacklog {
				a.syslog.Debugf("will replay reconnectBacklog %d %s", i, reflect.TypeOf(msg))
			}
		}

		for _, msg := range a.reconnectBacklog {
			switch msg := msg.(type) {
			case sproto.KillTaskContainer:
				a.killTaskContainer(msg)
			case sproto.StartTaskContainer:
				a.startTaskContainer(msg)
			default:
				panic(fmt.Sprintf("incorrect type for message buffered for recovery: %T", msg))
			}
		}
		a.reconnectBacklog = nil
		a.notifyListeners()
	}
	return nil
}

func (a *agent) HandleWebsocketDisconnect() {
	a.mu.Lock()
	defer a.mu.Unlock()

	defer a.notifyListeners()
	defer a.socketDisconnected()

	err := a.socket.Close()
	if err != nil {
		if !a.started {
			// If we happen to fail before the agent has started and been registered with
			// the resource manager, then nothing can be running on it. In this case we
			// just fail outright and make it restart.
			a.stop(errors.Wrapf(err, "child failed: %s", a.socket.Name()))
			return
		}

		a.syslog.
			WithError(err).
			Errorf("WebSocket failed, awaiting reconnect: %s", a.socket.Name())
		return
	}

	// If the socket has closed gracefully, there are really two cases:
	//  * the agent is being brought down temporarily (software or config update)
	//  * the agent is being brought down permanently
	// Since the former is more frequent and it doesn't really hurt the latter for the agent to
	// hang around for a bit on our side, we always treat gracefully socket closures as
	// temporary disconnects.
	if !a.started {
		a.stop(nil)
		return
	}

	a.syslog.Infof("websocket closed gracefully, awaiting reconnect: %s", a.socket.Name())
}

func (a *agent) GetAgent(msg *apiv1.GetAgentRequest) *apiv1.GetAgentResponse {
	a.mu.Lock()
	defer a.mu.Unlock()

	return &apiv1.GetAgentResponse{Agent: a.summarize().ToProto()}
}

func (a *agent) GetSlots(msg *apiv1.GetSlotsRequest) *apiv1.GetSlotsResponse {
	a.mu.Lock()
	defer a.mu.Unlock()

	var slots []*agentv1.Slot
	for _, s := range a.summarize().Slots {
		slots = append(slots, s.ToProto())
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].Id < slots[j].Id })
	return &apiv1.GetSlotsResponse{Slots: slots}
}

func (a *agent) EnableAgent(msg *apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.awaitingReconnect {
		return nil, errRecovering
	}

	if !a.started {
		return nil, errors.New("can't enable agent: agent not started")
	}

	a.agentState.enable()
	a.agentState.patchAllSlotsState(patchAllSlotsState{
		enabled: &a.agentState.enabled,
		drain:   &a.agentState.draining,
	})
	a.notifyListeners()
	return &apiv1.EnableAgentResponse{Agent: a.summarize().ToProto()}, nil
}

func (a *agent) DisableAgent(msg *apiv1.DisableAgentRequest) (*apiv1.DisableAgentResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.awaitingReconnect {
		return nil, errRecovering
	}

	if !a.started {
		return nil, errors.New("can't disable agent: agent not started")
	}

	// Mark current agent as disabled with RP.
	a.agentState.disable(msg.Drain)
	// Update individual slot state.
	a.agentState.patchAllSlotsState(patchAllSlotsState{
		enabled: &a.agentState.enabled,
		drain:   &a.agentState.draining,
	})
	// Kill both slotted and zero-slot tasks, unless draining.
	if !msg.Drain {
		for _, aID := range a.agentState.containerAllocation {
			rmevents.Publish(aID, &sproto.ReleaseResources{
				Reason:    "agent disabled",
				ForceKill: true,
			})
		}
	}
	a.notifyListeners()
	return &apiv1.DisableAgentResponse{Agent: a.summarize().ToProto()}, nil
}

func (a *agent) PatchSlotState(msg patchSlotState) (*model.SlotSummary, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		return nil, errors.New("can't patch slot state: agent not started")
	}

	result, err := a.agentState.patchSlotState(msg)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// On the Determined Enterprise Edition, when the dev cluster is started with
// "tools/slurmcluster.sh", an SSH tunnel is created between the local host and
// the HPC cluster. Due to the way the reverse tunnel is created, all the nodes
// will report a loopback address of "[::1]". This will cause distributed
// experiments to fail. To work around this, use the agent's "hostname"
// parameter, which will be the node name that the agent is running on, as the
// IP address. As long as the cluster has been configured to resolve the node
// names to their respective IP addresses via "/etc/hosts", DNS, or some other
// mechanism, this will work.
func (a *agent) adjustAgentIPAddrIfRunningDevClusterOnHpcUsingAnSSHTunnel(
	msg webSocketRequest,
) {
	// Check if the address is a loopback address.
	if addr := net.ParseIP(strings.Trim(a.address, "[]")); addr != nil && addr.IsLoopback() {
		agentHostname := strings.TrimSpace(msg.echoCtx.QueryParam("hostname"))

		masterHostname, err := os.Hostname()
		if err != nil {
			a.syslog.Warnf("Unable to get hostname : %v", err)
		}

		// We're not running on a local cluster. In other words, the agent and
		// master are on not the same host. Therefore, the assumption is that
		// we received a loopback address (i.e., "[::1]") from the agent due
		// to the reverse tunnel that was set up by "tools/slurmcluster.sh".
		// Use the "hostname" parameter that the agent sent us as the address.
		if agentHostname != masterHostname {
			a.syslog.Infof(
				"remote address for agent is loopback ('%s'), using provided hostname '%s' instead",
				a.address, agentHostname)
			a.address = agentHostname
		}
	}
}

func (a *agent) bufferForRecovery(msg any) {
	// This explodes the debug logs when msg is big
	// a.syslog.WithField("msg", msg).Debugf("buffering message until agent reconnects")
	a.syslog.WithField("msg-type", reflect.TypeOf(msg)).
		Debugf("buffering message until agent reconnects")
	a.reconnectBacklog = append(a.reconnectBacklog, msg)
}

func (a *agent) HandleIncomingWebsocketMessage(msg *aproto.MasterMessage) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch {
	case msg.AgentStarted != nil:
		a.syslog.Infof("agent connected ip: %v resource pool: %s slots: %d",
			a.address, a.resourcePoolName, len(msg.AgentStarted.Devices))

		if a.started {
			err := a.agentState.checkAgentStartedDevicesMatch(msg.AgentStarted)
			if err != nil {
				a.syslog.WithError(err).
					Error("change in agent devices was detected")
				a.socket.Outbox <- aproto.AgentMessage{
					AgentShutdown: &aproto.AgentShutdown{
						ErrMsg: aproto.ErrAgentMustReconnect.Error(),
					},
				}

				a.stop(err)
				return
			}
		} else {
			a.agentStarted(msg.AgentStarted)
		}

		a.started = true

		if err := a.handleContainersReattached(msg.AgentStarted); err != nil {
			a.syslog.WithError(err).
				Error("failure in handleContainersReattached")
		}
	case msg.ContainerStateChanged != nil:
		a.containerStateChanged(*msg.ContainerStateChanged)
	case msg.ContainerLog != nil:
		aID, ok := a.agentState.containerAllocation[msg.ContainerLog.ContainerID]
		if !ok {
			containerID := msg.ContainerLog.ContainerID
			a.syslog.WithField("container-id", containerID).Warnf(
				"received ContainerLog from container not allocated to agent: "+
					"container %s, message: %v", containerID, msg.ContainerLog)
			return
		}
		rmevents.Publish(aID, &sproto.ContainerLog{
			ContainerID: msg.ContainerLog.ContainerID,
			Timestamp:   msg.ContainerLog.Timestamp,
			PullMessage: msg.ContainerLog.PullMessage,
			RunMessage:  msg.ContainerLog.RunMessage,
			AuxMessage:  msg.ContainerLog.AuxMessage,
			Level:       msg.ContainerLog.Level,
			Source:      msg.ContainerLog.Source,
			AgentID:     msg.ContainerLog.AgentID,
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
				a.syslog.Errorf("error recording task stats %s", err)
			}
		}

	default:
		check.Panic(errors.Errorf("error parsing incoming message"))
	}
}

func (a *agent) taskNeedsRecording(record *aproto.ContainerStatsRecord) bool {
	return record.TaskType == model.TaskTypeTrial
}

func (a *agent) agentStarted(agentStarted *aproto.AgentStarted) {
	a.agentState = newAgentState(a.id, a.maxZeroSlotContainers)
	a.agentState.handler = a
	a.agentState.resourcePoolName = a.resourcePoolName
	a.agentState.agentStarted(agentStarted)

	a.syslog.Infof("adding agent: %s", a.agentState.agentID())
	err := a.updateAgentStartStats(a.resourcePoolName, string(a.id), a.agentState.numSlots())
	if err != nil {
		a.syslog.WithError(err).Error("failed to update agent start stats")
	}
	a.notifyListeners()
}

func (a *agent) containerStateChanged(sc aproto.ContainerStateChanged) {
	aID, ok := a.agentState.containerAllocation[sc.Container.ID]
	if !ok {
		// We may receieve late terminations when reconnected agent is cleaning up
		// terminated containers.
		if sc.Container.State != cproto.Terminated {
			containerID := sc.Container.ID
			a.syslog.WithField("container-id", containerID).Warnf(
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
		a.syslog.
			WithError(sc.ContainerStopped.Failure).
			Infof("container %s terminated", sc.Container.ID)
		delete(a.agentState.containerAllocation, sc.Container.ID)
	}

	rmevents.Publish(aID, sproto.FromContainerStateChanged(sc))
	a.agentState.containerStateChanged(sc)
}

func (a *agent) Summarize() model.AgentSummary {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.summarize()
}

func (a *agent) summarize() model.AgentSummary {
	result := model.AgentSummary{
		ID:             string(a.id),
		RegisteredTime: a.registeredTime,
		ResourcePool:   []string{a.resourcePoolName},
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
		result.Slots = a.agentState.getSlotsSummary(fmt.Sprintf("/agents/%s", a.id))
		result.Enabled = a.agentState.enabled
		result.Draining = a.agentState.draining
		result.NumContainers = len(a.agentState.containerAllocation)
	}

	return result
}

func (a *agent) gatherContainersToReattach() []aproto.ContainerReattach {
	err := a.agentState.restoreContainersField()
	if err != nil {
		a.syslog.WithError(err).Warn("failed restoreContainersField in gatherContainersToReattach")
	}
	result := make([]aproto.ContainerReattach, 0, len(a.agentState.containerAllocation))

	for _, container := range a.agentState.containerState {
		result = append(result, aproto.ContainerReattach{Container: *container})
	}
	a.syslog.Infof("going to try to reattach containers (%v)", result)
	return result
}

func (a *agent) handleContainersReattached(agentStarted *aproto.AgentStarted) error {
	a.syslog.WithField("ip", a.address).
		Debugf(
			"reattached containers: actual: %v, expected: %v",
			agentStarted.ContainersReattached, maps.Keys(a.agentState.containerState),
		)

	recovered := map[cproto.ID]aproto.ContainerReattachAck{}
	doomed := map[cproto.ID]aproto.ContainerReattachAck{}

	for _, containerRestored := range agentStarted.ContainersReattached {
		cid := containerRestored.Container.ID
		if containerRestored.Failure != nil &&
			containerRestored.Failure.FailureType == aproto.RestoreError {
			a.syslog.Infof(
				"agent failed to restore container: %s: %s",
				cid, containerRestored.Failure.ErrMsg)
			doomed[cid] = containerRestored
			continue
		}

		if containerRestored.Failure != nil {
			a.syslog.Infof(
				"reattached container %s terminated while away: %s",
				cid, containerRestored.Failure.ErrMsg)
			doomed[cid] = containerRestored
			continue
		}

		if containerRestored.Container.State == cproto.Terminated {
			a.syslog.Warnf(
				"reattached container %s terminated while away", cid)
			doomed[cid] = containerRestored
			continue
		}

		_, ok := a.agentState.containerAllocation[cid]
		if !ok {
			a.syslog.Warnf(
				"agent state is missing container %s on reattach", cid)
			doomed[cid] = containerRestored
			continue
		}

		if a.agentState.containerState[cid].State != containerRestored.Container.State {
			a.syslog.Warnf(
				"reattached container %s has changed state: %s to %s",
				cid, a.agentState.containerState[cid].State,
				containerRestored.Container.State)
			doomed[cid] = containerRestored
			continue
		}

		recovered[cid] = containerRestored
	}

	// Mark the rest as dead.
	return a.clearNonReattachedContainers(recovered, doomed)
}

func (a *agent) clearNonReattachedContainers(
	recovered map[cproto.ID]aproto.ContainerReattachAck,
	explicitlyDoomed map[cproto.ID]aproto.ContainerReattachAck,
) error {
	for cID := range a.agentState.containerAllocation {
		if _, ok := recovered[cID]; ok {
			continue
		}

		containerState := a.agentState.containerState[cID]
		if containerState == nil {
			containerState = &cproto.Container{ID: cID}
		}
		containerState.State = cproto.Terminated

		var stopped aproto.ContainerStopped
		ack, ok := explicitlyDoomed[cID]
		if ok {
			stopped = aproto.ContainerStopped{Failure: ack.Failure}
		} else {
			stopped = a.defaultReattachFailureMessage()
		}

		a.containerStateChanged(aproto.ContainerStateChanged{
			Container:        *containerState,
			ContainerStopped: &stopped,
		})

		// One of the reasons we can fail to recover a container is if there is no allocation awaiting it (e.g.,
		// if the allocation has already run "purgeRestoreableResources" and crashed). In this case, sending
		// aproto.ContainerStateChanged does nothing since the allocation isn't there to handle it, so we send an
		// extra SIGKILL to make sure we don't oversubscribe the agent since we are about to clear it from the agent
		// state and those slots will become reschedulable.
		//
		// To me, this is a hack to make up for an architectural deficiency. I think this problem and others would go
		// away if we merged task.AllocationService and ResourceManagers into a single entity. There is too much shared
		// responsibility between them.
		a.socket.Outbox <- aproto.AgentMessage{
			SignalContainer: &aproto.SignalContainer{
				ContainerID: cID,
				Signal:      syscall.SIGKILL,
			},
		}
	}

	return a.agentState.clearUnlessRecovered(recovered)
}

func (a *agent) defaultReattachFailureMessage() aproto.ContainerStopped {
	return aproto.ContainerError(
		aproto.AgentFailed,
		errors.New("failed to reattach container on reconnect"),
	)
}

func (a *agent) socketDisconnected() {
	a.socket = nil
	a.awaitingReconnect = true

	timer := time.AfterFunc(a.agentReconnectWait, a.HandleReconnectTimeout)
	a.reconnectTimers = append(a.reconnectTimers, timer)

	a.preDisconnectEnabled = a.agentState.enabled
	a.preDisconnectDraining = a.agentState.draining
	// Mark ourselves as draining to avoid action on ourselves while we recover. While the
	// system is technically correct without this, it's better because we avoid any waste
	// effort scheduling things only to have them suffer AgentErrors later.
	a.agentState.disable(true)
	a.agentState.patchAllSlotsState(patchAllSlotsState{
		enabled: &a.agentState.enabled,
		drain:   &a.agentState.draining,
	})
	a.notifyListeners()
}

func (a *agent) HandleReconnectTimeout() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.awaitingReconnect {
		a.stop(errors.New("agent failed to reconnect by deadline"))
		return
	}
}

func (a *agent) notifyListeners() {
	a.agentUpdates.Put(agentUpdatedEvent{resourcePool: a.resourcePoolName})
}

func (a *agent) updateAgentStartStats(
	poolName string, agentID string, slots int,
) error {
	return db.SingleDB().RecordAgentStats(&model.AgentStats{
		ResourcePool: poolName,
		AgentID:      agentID,
		Slots:        slots,
	})
}

func (a *agent) updateAgentEndStats(agentID string) error {
	return db.EndAgentStats(&model.AgentStats{
		AgentID: agentID,
	})
}
