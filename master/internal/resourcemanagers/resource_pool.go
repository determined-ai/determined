package resourcemanagers

import (
	"crypto/tls"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// ResourcePool manages the agent and task lifecycles.
type ResourcePool struct {
	config *ResourcePoolConfig
	cert   *tls.Certificate

	scheduler        Scheduler
	fittingMethod    SoftConstraint
	provisioner      *actor.Ref
	slotsPerInstance int

	agents      map[*actor.Ref]*agentState
	taskList    *taskList
	groups      map[*actor.Ref]*group
	scalingInfo *sproto.ScalingInfo

	reschedule bool

	// Track notifyOnStop for testing purposes.
	saveNotifications bool
	notifications     []<-chan struct{}
}

// GetResourceSummary is a message to request a summary of the resources used by the
// resource pool (agents, slots, cpu containers).
type GetResourceSummary struct{}

// NewResourcePool initializes a new empty default resource provider.
func NewResourcePool(
	config *ResourcePoolConfig,
	cert *tls.Certificate,
	scheduler Scheduler,
	fittingMethod SoftConstraint,
) *ResourcePool {
	d := &ResourcePool{
		config: config,
		cert:   cert,

		scheduler:     scheduler,
		fittingMethod: fittingMethod,

		agents:      make(map[*actor.Ref]*agentState),
		taskList:    newTaskList(),
		groups:      make(map[*actor.Ref]*group),
		scalingInfo: &sproto.ScalingInfo{},

		reschedule: false,
	}
	return d
}

func (rp *ResourcePool) setupProvisioner(ctx *actor.Context) error {
	if rp.config.Provider == nil {
		ctx.Log().Infof("not enabling provisioner for resource pool: %s", rp.config.PoolName)
		return nil
	}
	p, pRef, err := provisioner.Setup(ctx, rp.config.Provider, rp.config.PoolName, rp.cert)
	if err != nil {
		return errors.Wrapf(err, "cannot create resource pool: %s", rp.config.PoolName)
	}
	rp.slotsPerInstance = p.SlotsPerInstance()
	rp.provisioner = pRef
	return nil
}

func (rp *ResourcePool) addTask(ctx *actor.Context, msg sproto.AllocateRequest) {
	rp.notifyOnStop(ctx, msg.TaskActor, sproto.ResourcesReleased{TaskActor: msg.TaskActor})

	if len(msg.ID) == 0 {
		msg.ID = sproto.TaskID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.TaskActor
	}
	rp.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed Task"
	}

	ctx.Log().Infof(
		"resources are requested by %s (Task ID: %s)",
		msg.TaskActor.Address(), msg.ID,
	)
	rp.taskList.AddTask(&msg)
}

func (rp *ResourcePool) receiveSetTaskName(ctx *actor.Context, msg sproto.SetTaskName) {
	if task, found := rp.taskList.GetTaskByHandler(msg.TaskHandler); found {
		task.Name = msg.Name
	}
}

// allocateResources assigns resources based on a request and notifies the request
// handler of the assignment. It returns true if it is successfully allocated.
func (rp *ResourcePool) allocateResources(ctx *actor.Context, req *sproto.AllocateRequest) bool {
	fits := findFits(req, rp.agents, rp.fittingMethod)

	if len(fits) == 0 {
		return false
	}

	allocations := make([]sproto.Allocation, 0, len(fits))
	for _, fit := range fits {
		container := newContainer(req, fit.Agent, fit.Slots)
		allocations = append(allocations, &containerAllocation{
			req:       req,
			agent:     fit.Agent,
			container: container,
			devices:   fit.Agent.allocateFreeDevices(fit.Slots, container.id),
		})
	}

	allocated := sproto.ResourcesAllocated{
		ID: req.ID, ResourcePool: rp.config.PoolName, Allocations: allocations,
	}
	rp.taskList.SetAllocations(req.TaskActor, &allocated)
	req.TaskActor.System().Tell(req.TaskActor, allocated)
	ctx.Log().Infof("allocated resources to %s", req.TaskActor.Address())

	return true
}

func (rp *ResourcePool) releaseResource(ctx *actor.Context, handler *actor.Ref) {
	ctx.Log().Infof("releasing resources taken by %s", handler.Address())
	handler.System().Tell(handler, sproto.ReleaseResources{ResourcePool: rp.config.PoolName})
}

func (rp *ResourcePool) resourcesReleased(ctx *actor.Context, handler *actor.Ref) {
	ctx.Log().Infof("resources are released for %s", handler.Address())
	rp.taskList.RemoveTaskByHandler(handler)
}

func (rp *ResourcePool) getOrCreateGroup(
	ctx *actor.Context, handler *actor.Ref,
) *group {
	if g, ok := rp.groups[handler]; ok {
		return g
	}
	g := &group{handler: handler, weight: 1}

	if rp.config.Scheduler.Priority != nil {
		if rp.config.Scheduler.Priority.DefaultPriority == nil {
			panic("default priority is not configured")
		}
		g.priority = rp.config.Scheduler.Priority.DefaultPriority
	}

	rp.groups[handler] = g
	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, groupActorStopped{})
	}
	return g
}

func (rp *ResourcePool) notifyOnStop(
	ctx *actor.Context, ref *actor.Ref, msg actor.Message,
) {
	done := actors.NotifyOnStop(ctx, ref, msg)
	if rp.saveNotifications {
		rp.notifications = append(rp.notifications, done)
	}
}

func (rp *ResourcePool) updateScalingInfo() bool {
	desiredInstanceNum := calculateDesiredNewAgentNum(
		rp.taskList, rp.slotsPerInstance, rp.config.MaxCPUContainersPerAgent,
	)
	agents := make(map[string]sproto.AgentSummary)
	for _, agentState := range rp.agents {
		summary := newAgentSummary(agentState)
		agents[summary.Name] = summary
	}
	return rp.scalingInfo.Update(desiredInstanceNum, agents)
}

func (rp *ResourcePool) sendScalingInfo(ctx *actor.Context) {
	if rp.provisioner != nil && rp.updateScalingInfo() {
		ctx.Tell(rp.provisioner, *rp.scalingInfo)
	}
}

// Receive implements the actor.Actor interface.
func (rp *ResourcePool) Receive(ctx *actor.Context) error {
	ctx.AddLabel("resource-pool", rp.config.PoolName)

	reschedule := true
	defer func() {
		// Default to scheduling every 500ms if a message was received, but allow messages
		// that don't affect the cluster to be skipped.
		rp.reschedule = rp.reschedule || reschedule
	}()

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		reportResourcePoolCreated(ctx.Self().System(), rp.config)
		err := rp.setupProvisioner(ctx)
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})
		return err

	case
		sproto.AddAgent,
		sproto.AddDevice,
		sproto.FreeDevice,
		sproto.RemoveDevice,
		sproto.RemoveAgent:
		return rp.receiveAgentMsg(ctx)

	case
		groupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetGroupWeight,
		sproto.SetGroupPriority,
		sproto.SetTaskName,
		sproto.AllocateRequest,
		sproto.ResourcesReleased:
		return rp.receiveRequestMsg(ctx)

	case sproto.GetTaskSummary:
		reschedule = false
		if resp := getTaskSummary(
			rp.taskList, *msg.ID, rp.groups, rp.config.Scheduler.GetType()); resp != nil {
			ctx.Respond(*resp)
		}

	case sproto.GetTaskSummaries:
		reschedule = false
		ctx.Respond(getTaskSummaries(rp.taskList, rp.groups, rp.config.Scheduler.GetType()))

	case GetResourceSummary:
		reschedule = false
		ctx.Respond(getResourceSummary(rp.agents))

	case schedulerTick:
		if rp.reschedule {
			toAllocate, toRelease := rp.scheduler.Schedule(rp)
			for _, req := range toAllocate {
				rp.allocateResources(ctx, req)
			}
			for _, taskActor := range toRelease {
				rp.releaseResource(ctx, taskActor)
			}
			rp.sendScalingInfo(ctx)
		}
		rp.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	default:
		reschedule = false
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) receiveAgentMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.AddAgent:
		ctx.Log().Infof("adding agent: %s", msg.Agent.Address().Local())
		rp.agents[msg.Agent] = newAgentState(msg, rp.config.MaxCPUContainersPerAgent)

	case sproto.AddDevice:
		ctx.Log().Infof("adding device: %s on %s", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := rp.agents[msg.Agent]
		check.Panic(check.True(ok, "error adding device, agent not found: %s", msg.Agent.Address()))
		state.devices[msg.Device] = msg.ContainerID

	case sproto.FreeDevice:
		ctx.Log().Infof("freeing device from container %s: %s on %s",
			msg.ContainerID, msg.Device.String(), msg.Agent.Address().Local())
		state, ok := rp.agents[msg.Agent]
		check.Panic(check.True(ok, "error freeing device, agent not found: %s", msg.Agent.Address()))
		state.deallocateDevice(msg.Device.Type, *msg.ContainerID, msg.Device)

	case sproto.RemoveDevice:
		ctx.Log().Infof("removing device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := rp.agents[msg.Agent]
		check.Panic(check.True(ok, "error removing device, agent not found: %s", msg.Agent.Address()))
		delete(state.devices, msg.Device)

	case sproto.RemoveAgent:
		ctx.Log().Infof("removing agent: %s", msg.Agent.Address().Local())
		delete(rp.agents, msg.Agent)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case groupActorStopped:
		delete(rp.groups, msg.Ref)

	case sproto.SetGroupMaxSlots:
		rp.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

	case sproto.SetGroupWeight:
		rp.getOrCreateGroup(ctx, msg.Handler).weight = msg.Weight

	case sproto.SetGroupPriority:
		group := rp.getOrCreateGroup(ctx, msg.Handler)
		if msg.Priority != nil {
			group.priority = msg.Priority
		}

		if rp.config.Scheduler.Priority != nil {
			ctx.Log().Infof("setting priority for group of %s to %d",
				msg.Handler.Address().String(), *group.priority)
		}

	case sproto.SetTaskName:
		rp.receiveSetTaskName(ctx, msg)

	case sproto.AllocateRequest:
		rp.addTask(ctx, msg)

	case sproto.ResourcesReleased:
		rp.resourcesReleased(ctx, msg.TaskActor)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// containerAllocation contains information for tasks have been allocated but not yet started.
type containerAllocation struct {
	req       *sproto.AllocateRequest
	container *container
	agent     *agentState
	devices   []device.Device
}

// Summary summarizes a container allocation.
func (c containerAllocation) Summary() sproto.ContainerSummary {
	return sproto.ContainerSummary{
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

func reportResourcePoolCreated(system *actor.System, config *ResourcePoolConfig) {
	if config.Scheduler == nil {
		panic("scheduler not configured in resource pool")
	}

	var preemptionEnabled bool
	switch {
	case config.Scheduler.FairShare != nil:
		preemptionEnabled = true
	case config.Scheduler.Priority != nil:
		preemptionEnabled = config.Scheduler.Priority.Preemption
	case config.Scheduler.RoundRobin != nil:
		preemptionEnabled = false
	}

	telemetry.ReportResourcePoolCreated(
		system, config.PoolName, config.Scheduler.GetType(),
		config.Scheduler.FittingPolicy, preemptionEnabled,
	)
}
