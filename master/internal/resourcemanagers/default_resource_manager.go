package resourcemanagers

import (
	"github.com/google/uuid"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/agent"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// DeterminedResourceManager manages the agent and task lifecycles.
type DeterminedResourceManager struct {
	scheduler     Scheduler
	fittingMethod SoftConstraint
	agents        map[*actor.Ref]*agentState

	taskList *taskList
	groups   map[*actor.Ref]*group

	provisioner      *actor.Ref
	slotsPerInstance int
	scalingInfo      *sproto.ScalingInfo

	reschedule bool

	// Track notifyOnStop for testing purposes.
	saveNotifications bool
	notifications     []<-chan struct{}
}

// NewDeterminedResourceManager initializes a new empty default resource provider.
func NewDeterminedResourceManager(
	scheduler Scheduler,
	fittingMethod SoftConstraint,
	provisioner *actor.Ref,
	provisionerSlotsPerInstance int,
) actor.Actor {
	d := &DeterminedResourceManager{
		scheduler:     scheduler,
		fittingMethod: fittingMethod,
		agents:        make(map[*actor.Ref]*agentState),
		groups:        make(map[*actor.Ref]*group),

		taskList: newTaskList(),

		provisioner:      provisioner,
		slotsPerInstance: provisionerSlotsPerInstance,
		scalingInfo:      &sproto.ScalingInfo{},

		reschedule: false,
	}
	return d
}

func (d *DeterminedResourceManager) addTask(ctx *actor.Context, msg AllocateRequest) {
	d.notifyOnStop(ctx, msg.TaskActor, ResourcesReleased{TaskActor: msg.TaskActor})

	if len(msg.ID) == 0 {
		msg.ID = TaskID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.TaskActor
	}
	d.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed Task"
	}

	ctx.Log().Infof(
		"resources are requested by %s (Task ID: %s)",
		msg.TaskActor.Address(), msg.ID,
	)
	d.taskList.AddTask(&msg)
}

// allocateResources assigns resources based on a request and notifies the request
// handler of the assignment. It returns true if it is successfully allocated.
func (d *DeterminedResourceManager) allocateResources(req *AllocateRequest) bool {
	fits := findFits(req, d.agents, d.fittingMethod)

	if len(fits) == 0 {
		return false
	}

	allocations := make([]Allocation, 0, len(fits))
	for _, fit := range fits {
		container := newContainer(req, fit.Agent, fit.Slots)
		allocations = append(allocations, &containerAllocation{
			req:       req,
			agent:     fit.Agent,
			container: container,
			devices:   fit.Agent.allocateFreeDevices(fit.Slots, container.id),
		})
	}

	allocated := ResourcesAllocated{ID: req.ID, Allocations: allocations}
	d.taskList.SetAllocations(req.TaskActor, &allocated)
	req.TaskActor.System().Tell(req.TaskActor, allocated)
	log.Infof("allocated resources to %s", req.TaskActor.Address())

	return true
}

func (d *DeterminedResourceManager) releaseResource(handler *actor.Ref) {
	log.Infof("releasing resources taken by %s", handler.Address())
	handler.System().Tell(handler, ReleaseResources{})
}

func (d *DeterminedResourceManager) resourcesReleased(ctx *actor.Context, handler *actor.Ref) {
	ctx.Log().Infof("resources are released for %s", handler.Address())
	d.taskList.RemoveTaskByHandler(handler)
}

func (d *DeterminedResourceManager) getOrCreateGroup(
	ctx *actor.Context, handler *actor.Ref,
) *group {
	if g, ok := d.groups[handler]; ok {
		return g
	}
	g := &group{handler: handler, weight: 1}
	d.groups[handler] = g
	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, groupActorStopped{})
	}
	return g
}

func (d *DeterminedResourceManager) notifyOnStop(
	ctx *actor.Context, ref *actor.Ref, msg actor.Message,
) {
	done := actors.NotifyOnStop(ctx, ref, msg)
	if d.saveNotifications {
		d.notifications = append(d.notifications, done)
	}
}

func (d *DeterminedResourceManager) updateScalingInfo() bool {
	desiredInstanceNum := calculateDesiredNewInstanceNum(d.taskList, d.slotsPerInstance)
	agents := make(map[string]sproto.AgentSummary)
	for _, agentState := range d.agents {
		summary := newAgentSummary(agentState)
		agents[summary.Name] = summary
	}
	return d.scalingInfo.Update(desiredInstanceNum, agents)
}

func (d *DeterminedResourceManager) sendScalingInfo(ctx *actor.Context) {
	if d.provisioner != nil && d.updateScalingInfo() {
		ctx.Tell(d.provisioner, *d.scalingInfo)
	}
}

// Receive implements the actor.Actor interface.
func (d *DeterminedResourceManager) Receive(ctx *actor.Context) error {
	reschedule := true
	defer func() {
		// Default to scheduling every 500ms if a message was received, but allow messages
		// that don't affect the cluster to be skipped.
		d.reschedule = d.reschedule || reschedule
	}()

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	case
		sproto.ConfigureEndpoints,
		sproto.AddAgent,
		sproto.AddDevice,
		sproto.FreeDevice,
		sproto.RemoveDevice,
		sproto.RemoveAgent:
		return d.receiveAgentMsg(ctx)

	case
		groupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetGroupWeight,
		AllocateRequest,
		ResourcesReleased:
		return d.receiveRequestMsg(ctx)

	case GetTaskSummary:
		reschedule = false
		if resp := getTaskSummary(d.taskList, *msg.ID); resp != nil {
			ctx.Respond(*resp)
		}

	case GetTaskSummaries:
		reschedule = false
		ctx.Respond(getTaskSummaries(d.taskList))

	case sproto.GetEndpointActorAddress:
		reschedule = false
		ctx.Respond("/agents")

	case schedulerTick:
		if d.reschedule {
			toAllocate, toRelease := d.scheduler.Schedule(d)
			for _, req := range toAllocate {
				d.allocateResources(req)
			}
			for _, taskActor := range toRelease {
				d.releaseResource(taskActor)
			}
			d.sendScalingInfo(ctx)
		}
		d.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	default:
		reschedule = false
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (d *DeterminedResourceManager) receiveAgentMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ConfigureEndpoints:
		ctx.Log().Infof("initializing endpoints for agents")
		agent.Initialize(msg.System, msg.Echo, ctx.Self())

	case sproto.AddAgent:
		ctx.Log().Infof("adding agent: %s", msg.Agent.Address().Local())
		d.agents[msg.Agent] = newAgentState(msg)

	case sproto.AddDevice:
		ctx.Log().Infof("adding device: %s on %s", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := d.agents[msg.Agent]
		check.Panic(check.True(ok, "error adding device, agent not found: %s", msg.Agent.Address()))
		state.devices[msg.Device] = msg.ContainerID

	case sproto.FreeDevice:
		ctx.Log().Infof("freeing device from container %s: %s on %s",
			msg.ContainerID, msg.Device.String(), msg.Agent.Address().Local())
		state, ok := d.agents[msg.Agent]
		check.Panic(check.True(ok, "error freeing device, agent not found: %s", msg.Agent.Address()))

		if msg.Device.Type == device.ZeroSlot {
			delete(state.zeroSlotContainers, *msg.ContainerID)
		} else {
			id, ok := d.agents[msg.Agent].devices[msg.Device]
			check.Panic(check.True(ok, "error freeing device, device not found: %s", msg.Device))
			check.Panic(check.True(id != nil, "error freeing device, device not assigned: %s", msg.Device))
			state.devices[msg.Device] = nil
		}

	case sproto.RemoveDevice:
		ctx.Log().Infof("removing device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := d.agents[msg.Agent]
		check.Panic(check.True(ok, "error removing device, agent not found: %s", msg.Agent.Address()))
		delete(state.devices, msg.Device)

	case sproto.RemoveAgent:
		ctx.Log().Infof("removing agent: %s", msg.Agent.Address().Local())
		delete(d.agents, msg.Agent)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (d *DeterminedResourceManager) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case groupActorStopped:
		delete(d.groups, msg.Ref)

	case sproto.SetGroupMaxSlots:
		d.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

	case sproto.SetGroupWeight:
		d.getOrCreateGroup(ctx, msg.Handler).weight = msg.Weight

	case AllocateRequest:
		d.addTask(ctx, msg)

	case ResourcesReleased:
		d.resourcesReleased(ctx, msg.TaskActor)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// containerAllocation contains information for tasks have been allocated but not yet started.
type containerAllocation struct {
	req       *AllocateRequest
	container *container
	agent     *agentState
	devices   []device.Device
}

// Summary summarizes a container allocation.
func (c containerAllocation) Summary() ContainerSummary {
	return ContainerSummary{
		TaskID: c.req.ID,
		ID:     c.container.id,
		Agent:  c.agent.handler.Address().Local(),
	}
}

// StartContainer notifies the agent to start a container.
func (c containerAllocation) Start(ctx *actor.Context, spec image.TaskSpec) {
	handler := c.agent.handler
	spec.ContainerID = string(c.container.id)
	spec.TaskID = string(c.req.ID)
	spec.Devices = c.devices
	ctx.Tell(handler, sproto.StartTaskContainer{
		TaskActor: c.req.TaskActor,
		StartContainer: aproto.StartContainer{
			Container: cproto.Container{
				Parent:  c.req.TaskActor.Address(),
				ID:      c.container.id,
				State:   cproto.Assigned,
				Devices: c.devices,
			},
			Spec: image.ToContainerSpec(spec),
		},
	})
}

// KillContainer notifies the agent to kill the container.
func (c containerAllocation) Kill(ctx *actor.Context) {
	ctx.Tell(c.agent.handler, sproto.KillTaskContainer{
		ContainerID: c.container.id,
	})
}
