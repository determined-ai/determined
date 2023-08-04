package agentrm

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
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

func (s slotEnabled) enabled() bool {
	return s.agentEnabled && s.userEnabled
}

type slot struct {
	device      device.Device
	enabled     slotEnabled
	containerID *cproto.ID
}

// agentState holds the scheduler state for an agent. The implementation of agent-related operations
// (e.g., socket I/O) is deferred to the actor.
type agentState struct {
	// Handler is agent actor reference.
	Handler          *actor.Ref
	Devices          map[device.Device]*cproto.ID
	resourcePoolName string
	enabled          bool
	draining         bool
	uuid             uuid.UUID

	maxZeroSlotContainers int

	slotStates          map[device.ID]*slot
	containerAllocation map[cproto.ID]model.AllocationID
	containerState      map[cproto.ID]*cproto.Container
}

// newAgentState returns a new agent empty agent state backed by the handler.
func newAgentState(msg sproto.AddAgent, maxZeroSlotContainers int) *agentState {
	return &agentState{
		Handler:               msg.Agent,
		Devices:               make(map[device.Device]*cproto.ID),
		maxZeroSlotContainers: maxZeroSlotContainers,
		enabled:               true,
		slotStates:            make(map[device.ID]*slot),
		containerAllocation:   make(map[cproto.ID]model.AllocationID),
		containerState:        make(map[cproto.ID]*cproto.Container),
		uuid:                  uuid.New(),
	}
}

func (a *agentState) string() string {
	return a.Handler.Address().Local()
}

func (a *agentState) agentID() agentID {
	return agentID(a.string())
}

// numSlots returns the total number of slots available.
func (a *agentState) numSlots() int {
	switch {
	case a.draining:
		return a.numUsedSlots()
	case !a.enabled:
		return 0
	default:
		return len(a.Devices)
	}
}

// numEmptySlots returns the number of slots that have not been allocated to containers.
func (a *agentState) numEmptySlots() (slots int) {
	switch {
	case a.draining, !a.enabled:
		return 0
	default:
		return a.numSlots() - a.numUsedSlots()
	}
}

// numUsedSlots returns the number of slots that have been allocated to containers.
func (a *agentState) numUsedSlots() (slots int) {
	for _, id := range a.Devices {
		if id != nil {
			slots++
		}
	}
	return slots
}

// numUsedZeroSlots returns the number of allocated zero-slot units.
func (a *agentState) numUsedZeroSlots() int {
	result := 0
	for _, container := range a.containerState {
		if len(container.Devices) == 0 {
			result++
		}
	}

	return result
}

// numZeroSlots returns the total number of zero-slot units.
func (a *agentState) numZeroSlots() int {
	switch {
	case a.draining:
		return a.numUsedZeroSlots()
	case !a.enabled:
		return 0
	default:
		return a.maxZeroSlotContainers
	}
}

// numEmptyZeroSlots returns the number of unallocated zero-slot units.
func (a *agentState) numEmptyZeroSlots() int {
	switch {
	case a.draining || !a.enabled:
		return 0
	default:
		return a.numZeroSlots() - a.numUsedZeroSlots()
	}
}

// idle signals if the agent is idle.
func (a *agentState) idle() bool {
	return a.numUsedZeroSlots() == 0 && a.numUsedSlots() == 0
}

// allocateFreeDevices allocates container.
func (a *agentState) allocateFreeDevices(slots int, cid cproto.ID) ([]device.Device, error) {
	// TODO(ilia): Rename to AllocateContainer.
	a.containerState[cid] = &cproto.Container{ID: cid}
	if slots == 0 {
		return nil, nil
	}

	devices := make([]device.Device, 0, slots)
	for d, dcid := range a.Devices {
		if dcid == nil {
			devices = append(devices, d)
		}
		if len(devices) == slots {
			break
		}
	}

	if len(devices) != slots {
		return nil, errors.New("not enough devices")
	}

	for _, d := range devices {
		a.Devices[d] = &cid
	}

	a.containerState[cid].Devices = devices

	return devices, nil
}

// deallocateContainer deallocates containers.
func (a *agentState) deallocateContainer(id cproto.ID) {
	delete(a.containerState, id)
	for d, cid := range a.Devices {
		if cid != nil && *cid == id {
			a.Devices[d] = nil
		}
	}
}

// deepCopy returns a copy of agentState for scheduler internals.
func (a *agentState) deepCopy() *agentState {
	copiedAgent := &agentState{
		Handler:               a.Handler,
		Devices:               maps.Clone(a.Devices),
		maxZeroSlotContainers: a.maxZeroSlotContainers,
		enabled:               a.enabled,
		draining:              a.draining,
		containerState:        maps.Clone(a.containerState),
		// TODO(ilia): Deepcopy of `slotStates` may be necessary one day.
		slotStates: a.slotStates,
	}

	return copiedAgent
}

// enable enables the agent.
func (a *agentState) enable(ctx *actor.Context) {
	ctx.Log().Infof("enabling agent: %s", a.string())
	a.enabled = true
	a.draining = false
}

// disable disables or drains the agent.
func (a *agentState) disable(ctx *actor.Context, drain bool) {
	drainStr := "disabling"
	if drain {
		drainStr = "draining"
	}
	ctx.Log().Infof("%s agent: %s", drainStr, a.string())
	a.draining = drain
	a.enabled = false
}

func (a *agentState) addDevice(ctx *actor.Context, device device.Device, containerID *cproto.ID) {
	ctx.Log().Infof("adding device: %s on %s", device.String(), a.string())
	a.Devices[device] = containerID
}

func (a *agentState) removeDevice(ctx *actor.Context, device device.Device) {
	ctx.Log().Infof("removing device: %s (%s)", device.String(), a.string())
	delete(a.Devices, device)
}

// agentStarted initializes slots from AgentStarted.Devices.
func (a *agentState) agentStarted(ctx *actor.Context, agentStarted *aproto.AgentStarted) {
	msg := agentStarted
	for _, d := range msg.Devices {
		enabled := slotEnabled{
			agentEnabled: true,
			userEnabled:  true,
		}
		a.slotStates[d.ID] = &slot{enabled: enabled, device: d}
		a.updateSlotDeviceView(ctx, d.ID)
	}

	if err := a.persist(); err != nil {
		ctx.Log().Warnf("agentStarted persist failure")
	}
}

func (a *agentState) checkAgentStartedDevicesMatch(
	ctx *actor.Context, agentStarted *aproto.AgentStarted,
) error {
	ourDevices := map[device.ID]device.Device{}
	for did, slot := range a.slotStates {
		ourDevices[did] = slot.device
	}

	theirDevices := map[device.ID]device.Device{}
	for _, d := range agentStarted.Devices {
		theirDevices[d.ID] = d
	}

	if len(ourDevices) != len(theirDevices) {
		return fmt.Errorf("device count has changed: %d -> %d", len(ourDevices), len(theirDevices))
	}

	if !maps.Equal(ourDevices, theirDevices) {
		for k := range ourDevices {
			if ourDevices[k] != theirDevices[k] {
				return fmt.Errorf(
					"device properties have changed: %v -> %v",
					ourDevices[k],
					theirDevices[k],
				)
			}
		}
		return fmt.Errorf("devices has changed") // This should not happen!
	}

	return nil
}

func (a *agentState) containerStateChanged(ctx *actor.Context, msg aproto.ContainerStateChanged) {
	for _, d := range msg.Container.Devices {
		s, ok := a.slotStates[d.ID]
		if !ok {
			ctx.Log().Warnf("bad containerStateChanged on device: %d (%s)", d.ID, a.string())
			continue
		}

		s.containerID = &msg.Container.ID

		if msg.Container.State == cproto.Terminated {
			s.containerID = nil
		}
	}

	a.containerState[msg.Container.ID] = &msg.Container
	if msg.Container.State == cproto.Terminated {
		delete(a.containerState, msg.Container.ID)
	}

	if err := a.persist(); err != nil {
		ctx.Log().WithError(err).Warnf("containerStateChanged persist failure")
	}

	if err := updateContainerState(&msg.Container); err != nil {
		ctx.Log().WithError(err).Warnf("containerStateChanged failed to update container state")
	}
}

func (a *agentState) startContainer(ctx *actor.Context, msg sproto.StartTaskContainer) error {
	inner := func(deviceId device.ID) error {
		s, ok := a.slotStates[deviceId]
		if !ok {
			return errors.New("can't find slot")
		}

		// TODO(ilia): Potential race condition if slot is disabled in-between scheduling?
		if !s.enabled.enabled() {
			return errors.New("container allocated but slot is not enabled")
		}
		if s.containerID != nil {
			return errors.New("container already allocated to slot")
		}

		s.containerID = &msg.StartContainer.Container.ID
		a.containerState[msg.StartContainer.Container.ID] = &msg.StartContainer.Container

		return nil
	}

	for _, d := range msg.StartContainer.Container.Devices {
		if err := inner(d.ID); err != nil {
			return errors.Wrapf(err, "bad startContainer on device: %d (%s)", d.ID, a.string())
		}
	}

	a.containerAllocation[msg.Container.ID] = msg.AllocationID

	if err := a.persist(); err != nil {
		ctx.Log().WithError(err).Warnf("startContainer persist failure")
	}

	if err := updateContainerState(&msg.StartContainer.Container); err != nil {
		ctx.Log().WithError(err).Warnf("startContainer failed to update container state")
	}

	return nil
}

func (a *agentState) getSlotsSummary(ctx *actor.Context) model.SlotsSummary {
	summary := make(model.SlotsSummary, len(a.slotStates))
	for deviceID := range a.slotStates {
		summary[fmt.Sprintf("%s/slots/%d", ctx.Self().Address(), deviceID)] = a.getSlotSummary(
			deviceID,
		)
	}

	return summary
}

func (a *agentState) getSlotSummary(deviceID device.ID) model.SlotSummary {
	s := a.slotStates[deviceID]
	cid := s.containerID
	var container *cproto.Container
	if cid != nil {
		container = a.containerState[*cid]
	}

	return model.SlotSummary{
		ID:        strconv.Itoa(int(s.device.ID)),
		Device:    s.device,
		Enabled:   s.enabled.enabled(),
		Container: container,
		Draining:  s.enabled.draining,
	}
}

func (a *agentState) updateSlotDeviceView(ctx *actor.Context, deviceID device.ID) {
	s, ok := a.slotStates[deviceID]
	if !ok {
		ctx.Log().
			Warnf("bad updateSlotDeviceView on device: %d (%s): not found", deviceID, a.string())
		return
	}

	// TODO(ilia): Don't materialize `Devices` view on slots.
	if s.enabled.enabled() && !s.enabled.deviceAdded {
		s.enabled.deviceAdded = true

		a.addDevice(ctx, s.device, s.containerID)
	} else if !s.enabled.enabled() {
		if !s.enabled.draining && s.enabled.deviceAdded {
			s.enabled.deviceAdded = false
			a.removeDevice(ctx, s.device)
		}

		// On `PostStop`, draining will be already set to false, and we'll kill the container
		// whether we have the device or not.
		if !s.enabled.draining && s.containerID != nil {
			rmevents.Publish(a.containerAllocation[*s.containerID], &sproto.ReleaseResources{
				Reason:    "slot disabled",
				ForceKill: true,
			})
		}
	}
}

func (a *agentState) patchSlotStateInner(
	ctx *actor.Context, msg patchSlotState, slotState *slot,
) model.SlotSummary {
	if msg.enabled != nil {
		slotState.enabled.userEnabled = *msg.enabled
	}
	if msg.drain != nil {
		slotState.enabled.draining = *msg.drain
	}
	a.updateSlotDeviceView(ctx, slotState.device.ID)

	return a.getSlotSummary(slotState.device.ID)
}

func (a *agentState) patchAllSlotsState(
	ctx *actor.Context, msg patchAllSlotsState,
) model.SlotsSummary {
	result := model.SlotsSummary{}
	for _, slotState := range a.slotStates {
		summary := a.patchSlotStateInner(
			ctx, patchSlotState{
				id:      slotState.device.ID, // Note: this is effectively unused.
				enabled: msg.enabled,
				drain:   msg.drain,
			},
			slotState)
		result[summary.ID] = summary
	}
	return result
}

func (a *agentState) patchSlotState(
	ctx *actor.Context, msg patchSlotState,
) (model.SlotSummary, error) {
	s, ok := a.slotStates[msg.id]
	if !ok {
		return model.SlotSummary{}, fmt.Errorf(
			"bad updateSlotDeviceView on device: %d (%s): not found",
			msg.id,
			a.string(),
		)
	}
	return a.patchSlotStateInner(ctx, msg, s), nil
}

func (a *agentState) snapshot() *agentSnapshot {
	slots := make([]slotData, 0, len(a.slotStates))
	for _, slotState := range a.slotStates {
		slots = append(slots, slotData{
			Device:      slotState.device,
			UserEnabled: slotState.enabled.userEnabled,
			ContainerID: slotState.containerID,
		})
	}

	containerIds := maps.Keys(a.containerState)

	s := agentSnapshot{
		AgentID:          a.agentID(),
		UUID:             a.uuid.String(),
		ResourcePoolName: a.resourcePoolName,
		// TODO(ilia): we need to disambiguate user setting (which needs to be saved)
		// vs current state.
		UserEnabled:           a.enabled,
		UserDraining:          a.draining,
		MaxZeroSlotContainers: a.maxZeroSlotContainers,
		Slots:                 slots,
		Containers:            containerIds,
	}

	return &s
}

func (a *agentState) persist() error {
	snapshot := a.snapshot()
	_, err := db.Bun().NewInsert().Model(snapshot).
		On("CONFLICT (uuid) DO UPDATE").
		On("CONFLICT (agent_id) DO UPDATE").
		Exec(context.TODO())
	return err
}

func (a *agentState) delete() error {
	_, err := db.Bun().NewDelete().Model((*agentSnapshot)(nil)).
		Where("agent_id = ?", a.Handler.Address().Local()).
		Exec(context.TODO())
	return err
}

func (a *agentState) clearUnlessRecovered(
	recovered map[cproto.ID]aproto.ContainerReattachAck,
) error {
	updated := false
	for d := range a.Devices {
		if cID := a.Devices[d]; cID != nil {
			_, ok := recovered[*cID]
			if !ok {
				a.Devices[d] = nil
				a.slotStates[d.ID].containerID = nil
				updated = true
			}
		}
	}

	for _, slot := range a.slotStates {
		if slot.containerID != nil {
			_, ok := recovered[*slot.containerID]
			if !ok {
				slot.containerID = nil
				updated = true
			}
		}
	}

	for cid := range a.containerState {
		_, ok := recovered[cid]
		if !ok {
			delete(a.containerState, cid)
			updated = true
		}
	}

	for cid := range a.containerAllocation {
		_, ok := recovered[cid]
		if !ok {
			delete(a.containerAllocation, cid)
			updated = true
		}
	}

	if updated {
		return a.persist()
	}

	return nil
}

func listResourcePoolsWithReattachEnabled() []string {
	rpConfigList := config.GetMasterConfig().ResourcePools
	result := make([]string, 0, len(rpConfigList))
	for _, rpConfig := range rpConfigList {
		result = append(result, rpConfig.PoolName)
	}
	return result
}

// retrieveAgentStates reconstructs AgentStates from the database for all resource pools that
// have agent_container_reattachment enabled.
func retrieveAgentStates() (map[agentID]agentState, error) {
	rpNames := listResourcePoolsWithReattachEnabled()

	if len(rpNames) == 0 {
		return map[agentID]agentState{}, nil
	}

	snapshots := []agentSnapshot{}
	err := db.Bun().NewSelect().Model(&snapshots).
		Where("resource_pool_name IN (?)", bun.In(rpNames)).
		Scan(context.TODO())
	if err != nil {
		return nil, err
	}

	result := make(map[agentID]agentState, len(snapshots))

	for _, s := range snapshots {
		state, err := newAgentStateFromSnapshot(s)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate agent state %s: %w", s.AgentID, err)
		}

		result[s.AgentID] = *state
	}

	return result, nil
}

func newAgentStateFromSnapshot(as agentSnapshot) (*agentState, error) {
	parsedUUID, err := uuid.Parse(as.UUID)
	if err != nil {
		return nil, err
	}

	slotStates := make(map[device.ID]*slot)
	devices := make(map[device.Device]*cproto.ID)

	for _, sd := range as.Slots {
		slotStates[sd.Device.ID] = &slot{
			device:      sd.Device,
			containerID: sd.ContainerID,
			enabled: slotEnabled{
				deviceAdded:  true,
				agentEnabled: as.UserEnabled,
				userEnabled:  as.UserEnabled,
				draining:     as.UserDraining,
			},
		}
		if sd.ContainerID != nil {
			devices[sd.Device] = sd.ContainerID
		} else {
			devices[sd.Device] = nil
		}
	}

	containerState := make(map[cproto.ID]*cproto.Container)

	if len(as.Containers) > 0 {
		containerSnapshots := make([]containerSnapshot, 0, len(as.Containers))
		err := db.Bun().NewSelect().Model(&containerSnapshots).
			Where("container_id IN (?)", bun.In(as.Containers)).
			Scan(context.TODO())
		if err != nil {
			return nil, err
		}

		for _, containerSnapshot := range containerSnapshots {
			container := containerSnapshot.ToContainer()
			containerState[container.ID] = &container
		}
	}

	result := agentState{
		maxZeroSlotContainers: as.MaxZeroSlotContainers,
		resourcePoolName:      as.ResourcePoolName,
		uuid:                  parsedUUID,
		enabled:               as.UserEnabled,
		draining:              as.UserDraining,
		slotStates:            slotStates,
		Devices:               devices,
		containerAllocation:   make(map[cproto.ID]model.AllocationID),
		containerState:        containerState,
	}

	return &result, nil
}

func (a *agentState) restoreContainersField() error {
	containerIDs := maps.Keys(a.containerState)

	res, err := loadContainersToAllocationIds(containerIDs)
	if err != nil {
		return err
	}
	log.WithField("agent-id", a.string()).Debugf("restored containers: %d", len(res))
	a.containerAllocation = res

	return nil
}

func clearAgentStates(agentIds []agentID) error {
	_, err := db.Bun().NewDelete().Where("agent_id in (?)", agentIds).Exec(context.TODO())

	return err
}

func updateContainerState(c *cproto.Container) error {
	snapshot := newContainerSnapshot(c)
	_, err := db.Bun().NewUpdate().Model(&snapshot).
		Where("container_id = ?", snapshot.ID).
		Column("state", "devices").
		Exec(context.TODO())

	return err
}

func loadContainersToAllocationIds(
	containerIDs []cproto.ID,
) (map[cproto.ID]model.AllocationID, error) {
	cs := []containerSnapshot{}
	result := []map[string]interface{}{}
	rr := map[cproto.ID]model.AllocationID{}

	if len(containerIDs) == 0 {
		return rr, nil
	}

	err := db.Bun().NewSelect().Model(&cs).
		Join("JOIN allocation_resources al_res ON al_res.resource_id = rmac.resource_id").
		Where("container_id IN (?)", bun.In(containerIDs)).
		Column("container_id", "allocation_id").
		Scan(context.TODO(), &result)
	if err != nil {
		return nil, err
	}

	for _, row := range result {
		rr[cproto.ID(row["container_id"].(string))] = model.AllocationID(
			row["allocation_id"].(string),
		)
	}

	return rr, nil
}
