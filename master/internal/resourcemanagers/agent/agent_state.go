package agent

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

type slotEnabled struct {
	deviceAdded  bool
	agentEnabled bool
	userEnabled  bool
	draining     bool
}

func (s slotEnabled) Enabled() bool {
	return s.agentEnabled && s.userEnabled
}

type slot struct {
	device    device.Device
	enabled   slotEnabled
	container *cproto.Container
}

func (s *slot) summarize() model.SlotSummary {
	return model.SlotSummary{
		ID:        strconv.Itoa(int(s.device.ID)),
		Device:    s.device,
		Enabled:   s.enabled.Enabled(),
		Container: s.container,
		Draining:  s.enabled.draining,
	}
}

// AgentState holds the scheduler state for an agent. The implementation of agent-related operations
// (e.g., socket I/O) is deferred to the actor.
type AgentState struct {
	// Handler is agent actor reference.
	Handler  *actor.Ref
	Devices  map[device.Device]*cproto.ID
	Label    string
	enabled  bool
	draining bool

	// Since we only model GPUs as devices/slots and assume each slot can be allocated with
	// one container, we add one additional field to keep track of zero-slot containers.
	// We need this field to know if the agent is idle.
	ZeroSlotContainers    map[cproto.ID]bool
	maxZeroSlotContainers int

	slotStates map[device.ID]*slot
	containers map[cproto.ID]*actor.Ref
}

// NewAgentState returns a new agent empty agent state backed by the handler.
func NewAgentState(msg sproto.AddAgent, maxZeroSlotContainers int) *AgentState {
	return &AgentState{
		Handler:               msg.Agent,
		Label:                 msg.Label,
		Devices:               make(map[device.Device]*cproto.ID),
		ZeroSlotContainers:    make(map[cproto.ID]bool),
		maxZeroSlotContainers: maxZeroSlotContainers,
		enabled:               true,
		slotStates:            make(map[device.ID]*slot),
		containers:            make(map[cproto.ID]*actor.Ref),
	}
}

func (a *AgentState) string() string {
	return a.Handler.Address().Local()
}

// NumSlots returns the total number of slots available.
func (a *AgentState) NumSlots() int {
	switch {
	case a.draining:
		return a.NumUsedSlots()
	case !a.enabled:
		return 0
	default:
		return len(a.Devices)
	}
}

// NumEmptySlots returns the number of slots that have not been allocated to containers.
func (a *AgentState) NumEmptySlots() (slots int) {
	switch {
	case a.draining || !a.enabled:
		return 0
	default:
		return a.NumSlots() - a.NumUsedSlots()
	}
}

// NumUsedSlots returns the number of slots that have been allocated to containers.
func (a *AgentState) NumUsedSlots() (slots int) {
	for _, id := range a.Devices {
		if id != nil {
			slots++
		}
	}
	return slots
}

// NumUsedZeroSlots returns the number of allocated zero-slot units.
func (a *AgentState) NumUsedZeroSlots() int {
	return len(a.ZeroSlotContainers)
}

// NumZeroSlots returns the total number of zero-slot units.
func (a *AgentState) NumZeroSlots() int {
	switch {
	case a.draining:
		return a.NumUsedZeroSlots()
	case !a.enabled:
		return 0
	default:
		return a.maxZeroSlotContainers
	}
}

// NumEmptyZeroSlots returns the number of unallocated zero-slot units.
func (a *AgentState) NumEmptyZeroSlots() int {
	switch {
	case a.draining || !a.enabled:
		return 0
	default:
		return a.NumZeroSlots() - a.NumUsedZeroSlots()
	}
}

// Idle signals if the agent is idle.
func (a *AgentState) Idle() bool {
	return a.NumUsedZeroSlots() == 0 && a.NumUsedSlots() == 0
}

// AllocateFreeDevices allocates devices.
func (a *AgentState) AllocateFreeDevices(slots int, id cproto.ID) []device.Device {
	if slots == 0 {
		a.ZeroSlotContainers[id] = true
		return nil
	}
	cid := id
	devices := make([]device.Device, 0, slots)
	for d, dcid := range a.Devices {
		if dcid == nil {
			a.Devices[d] = &cid
			devices = append(devices, d)
		}
		if len(devices) == slots {
			break
		}
	}
	check.Panic(check.True(len(devices) == slots, "not enough devices"))
	return devices
}

// DeallocateContainer deallocates containers.
func (a *AgentState) DeallocateContainer(id cproto.ID) {
	delete(a.ZeroSlotContainers, id)
	for d, cid := range a.Devices {
		if cid != nil && *cid == id {
			a.Devices[d] = nil
		}
	}
}

// DeepCopy returns a copy of agentState for scheduler internals.
func (a *AgentState) DeepCopy() *AgentState {
	copiedAgent := &AgentState{
		Handler:               a.Handler,
		Label:                 a.Label,
		Devices:               make(map[device.Device]*cproto.ID),
		ZeroSlotContainers:    make(map[cproto.ID]bool),
		maxZeroSlotContainers: a.maxZeroSlotContainers,
		enabled:               a.enabled,
		draining:              a.draining,
		// TODO(ilia): Deepcopy of `slotStates` may be necessary one day.
		slotStates: a.slotStates,
	}

	for originalDevice, id := range a.Devices {
		copiedDevice := device.Device{
			ID:    originalDevice.ID,
			Brand: originalDevice.Brand,
			UUID:  originalDevice.UUID,
			Type:  originalDevice.Type,
		}
		copiedAgent.Devices[copiedDevice] = id
	}

	for originalKey, originalValue := range a.ZeroSlotContainers {
		copiedAgent.ZeroSlotContainers[originalKey] = originalValue
	}

	return copiedAgent
}

// Enable enables the agent.
func (a *AgentState) Enable(ctx *actor.Context) {
	ctx.Log().Infof("enabling agent: %s", a.string())
	a.enabled = true
	a.draining = false
}

// Disable disables or drains the agent.
func (a *AgentState) Disable(ctx *actor.Context, drain bool) {
	drainStr := "disabling"
	if drain {
		drainStr = "draining"
	}
	ctx.Log().Infof("%s agent: %s", drainStr, a.string())
	a.draining = drain
	a.enabled = false
}

func (a *AgentState) addDevice(ctx *actor.Context, device device.Device, containerID *cproto.ID) {
	ctx.Log().Infof("adding device: %s on %s", device.String(), a.string())
	a.Devices[device] = containerID
}

func (a *AgentState) removeDevice(ctx *actor.Context, device device.Device) {
	ctx.Log().Infof("removing device: %s (%s)", device.String(), a.string())
	delete(a.Devices, device)
}

// agentStarted initializes slots from AgentStarted.Devices.
func (a *AgentState) agentStarted(ctx *actor.Context, agentStarted *aproto.AgentStarted) {
	msg := agentStarted
	for _, d := range msg.Devices {
		enabled := slotEnabled{
			agentEnabled: true,
			userEnabled:  true,
		}
		a.slotStates[d.ID] = &slot{enabled: enabled, device: d}
		a.updateSlotDeviceView(ctx, d.ID)
	}
}

func (a *AgentState) containerStateChanged(ctx *actor.Context, msg aproto.ContainerStateChanged) {
	for _, d := range msg.Container.Devices {
		s, ok := a.slotStates[d.ID]
		if !ok {
			ctx.Log().Warnf("bad containerStateChanged on device: %d (%s)", d.ID, a.string())
			continue
		}

		s.container = &msg.Container
		if msg.Container.State == cproto.Terminated {
			s.container = nil
		}
	}
}

func (a *AgentState) startContainer(ctx *actor.Context, msg sproto.StartTaskContainer) error {
	inner := func(deviceId device.ID) error {
		s, ok := a.slotStates[deviceId]
		if !ok {
			return errors.New("can't find slot")
		}

		// TODO(ilia): Potential race condition if slot is disabled in-between scheduling?
		if !s.enabled.Enabled() {
			return errors.New("container allocated but slot is not enabled")
		}
		if s.container != nil {
			return errors.New("container already allocated to slot")
		}

		s.container = &msg.StartContainer.Container

		return nil
	}

	for _, d := range msg.StartContainer.Container.Devices {
		if err := inner(d.ID); err != nil {
			return errors.Wrapf(err, "bad startedContainer on device: %d (%s)", d.ID, a.string())
		}
	}

	a.containers[msg.Container.ID] = msg.TaskActor

	return nil
}

func (a *AgentState) getSlotsSummary(ctx *actor.Context) model.SlotsSummary {
	summary := make(model.SlotsSummary, len(a.slotStates))
	for deviceID, slotState := range a.slotStates {
		summary[fmt.Sprintf("%s/slots/%d", ctx.Self().Address(), deviceID)] = slotState.summarize()
	}

	return summary
}

func (a *AgentState) updateSlotDeviceView(ctx *actor.Context, deviceID device.ID) {
	s, ok := a.slotStates[deviceID]
	if !ok {
		ctx.Log().Warnf("bad updateSlotDeviceView on device: %d (%s): not found", deviceID, a.string())
		return
	}

	// TODO(ilia): Don't materialize `Devices` view on slots.
	if s.enabled.Enabled() && !s.enabled.deviceAdded {
		s.enabled.deviceAdded = true

		var containerID *cproto.ID
		if s.container != nil {
			containerID = &s.container.ID
		}

		a.addDevice(ctx, s.device, containerID)
	} else if !s.enabled.Enabled() {
		if !s.enabled.draining && s.enabled.deviceAdded {
			s.enabled.deviceAdded = false
			a.removeDevice(ctx, s.device)
		}

		// On `PostStop`, draining will be already set to false, and we'll kill the container
		// whether we have the device or not.
		if !s.enabled.draining && s.container != nil {
			ctx.Self().System().TellAt(s.container.Parent, task.Kill)
		}
	}
}

func (a *AgentState) patchSlotStateInner(
	ctx *actor.Context, msg PatchSlotState, slotState *slot) model.SlotSummary {
	if msg.Enabled != nil {
		slotState.enabled.userEnabled = *msg.Enabled
	}
	if msg.Drain != nil {
		slotState.enabled.draining = *msg.Drain
	}
	a.updateSlotDeviceView(ctx, slotState.device.ID)

	return slotState.summarize()
}

func (a *AgentState) patchAllSlotsState(
	ctx *actor.Context, msg PatchAllSlotsState) model.SlotsSummary {
	result := model.SlotsSummary{}
	for _, slotState := range a.slotStates {
		summary := a.patchSlotStateInner(
			ctx, PatchSlotState{ID: -1, Enabled: msg.Enabled, Drain: msg.Drain}, slotState)
		result[summary.ID] = summary
	}
	return result
}

func (a *AgentState) patchSlotState(
	ctx *actor.Context, msg PatchSlotState) (model.SlotSummary, error) {
	s, ok := a.slotStates[msg.ID]
	if !ok {
		return model.SlotSummary{}, errors.New(
			fmt.Sprintf("bad updateSlotDeviceView on device: %d (%s): not found", msg.ID, a.string()))
	}
	return a.patchSlotStateInner(ctx, msg, s), nil
}
