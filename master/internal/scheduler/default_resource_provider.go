package scheduler

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/device"

	"github.com/determined-ai/determined/master/internal/agent"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// DefaultRP manages the agent and task lifecycles.
type DefaultRP struct {
	scheduler     Scheduler
	fittingMethod SoftConstraint
	agents        map[*actor.Ref]*agentState

	reqList *taskList
	groups  map[*actor.Ref]*group

	provisioner     *actor.Ref
	provisionerView *FilterableView

	reschedule bool

	// Track notifyOnStop for testing purposes.
	saveNotifications bool
	notifications     []<-chan struct{}
}

// NewDefaultRP initializes a new empty default resource provider.
func NewDefaultRP(
	scheduler Scheduler,
	fittingMethod SoftConstraint,
	provisioner *actor.Ref,
	provisionerSlotsPerInstance int,
) actor.Actor {
	d := &DefaultRP{
		scheduler:     scheduler,
		fittingMethod: fittingMethod,
		agents:        make(map[*actor.Ref]*agentState),
		groups:        make(map[*actor.Ref]*group),

		reqList: newTaskList(),

		provisioner:     provisioner,
		provisionerView: newProvisionerView(provisionerSlotsPerInstance),

		reschedule: false,
	}
	return d
}

func (d *DefaultRP) addTask(ctx *actor.Context, msg AddTask) {
	d.notifyOnStop(ctx, msg.Handler, RemoveTask{Handler: msg.Handler})

	if len(msg.ID) == 0 {
		msg.ID = TaskID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.Handler
	}
	d.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed Task"
	}

	ctx.Log().Infof(
		"resources are requested by %s (request ID: %s)",
		msg.Handler.Address(), msg.ID,
	)
	d.reqList.AddTask(&msg)
}

// assignResources assigns resources based on a request and notifies the request
// handler of the assignment. It returns true if it is successfully assigned.
func (d *DefaultRP) assignResources(req *AddTask) bool {
	fits := findFits(req, d.agents, d.fittingMethod)

	if len(fits) == 0 {
		return false
	}

	assignments := make([]Assignment, 0, len(fits))
	for _, fit := range fits {
		container := newContainer(req, fit.Agent, fit.Slots, len(assignments))
		assignments = append(assignments, &containerAssignment{
			req:       req,
			agent:     fit.Agent,
			container: container,
			devices:   fit.Agent.assignFreeDevices(fit.Slots, cproto.ID(container.id)),
		})
	}

	assigned := ResourceAssigned{Assignments: assignments}
	d.reqList.SetAssignments(req.Handler, &assigned)
	req.Handler.System().Tell(req.Handler, assigned)

	return true
}

func (d *DefaultRP) releaseResource(handler *actor.Ref) {
	// The request handler is removed so that it would not take up resources.
	// In practice, the request handler might gracefully wait for the containers
	// to exit themselves, which might take very long time or might not happen at all.
	// In the mean time, resources are re-assigned to other requests, which might let
	// the old container exiting and the new container run slower than normal.
	// The task handler should kill the containers to physically release the resources
	// after a timeout.
	d.reqList.RemoveTask(handler)
	handler.System().Tell(handler, ReleaseResource{})
}

func (d *DefaultRP) resourcesReleased(ctx *actor.Context, handler *actor.Ref) {
	ctx.Log().Infof("resources are released for %s", handler.Address())
	d.reqList.RemoveTask(handler)
}

func (d *DefaultRP) getOrCreateGroup(ctx *actor.Context, handler *actor.Ref) *group {
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

func (d *DefaultRP) notifyOnStop(ctx *actor.Context, ref *actor.Ref, msg actor.Message) {
	done := actors.NotifyOnStop(ctx, ref, msg)
	if d.saveNotifications {
		d.notifications = append(d.notifications, done)
	}
}

func (d *DefaultRP) sendProvisionerView(ctx *actor.Context) {
	if d.provisioner != nil {
		if snapshot, updateMade := d.provisionerView.Update(d); updateMade {
			ctx.Tell(d.provisioner, snapshot)
		}
	}
}

// Receive implements the actor.Actor interface.
func (d *DefaultRP) Receive(ctx *actor.Context) error {
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
		SetGroupMaxSlots,
		SetGroupWeight,
		AddTask,
		RemoveTask:
		return d.receiveRequestMsg(ctx)

	case GetTaskSummary:
		reschedule = false
		if resp := getTaskSummary(d.reqList, *msg.ID); resp != nil {
			ctx.Respond(*resp)
		}

	case GetTaskSummaries:
		reschedule = false
		ctx.Respond(getTaskSummaries(d.reqList))

	case sproto.GetEndpointActorAddress:
		reschedule = false
		ctx.Respond("/agents")

	case schedulerTick:
		if d.reschedule {
			d.scheduler.Schedule(d)
			d.sendProvisionerView(ctx)
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

func (d *DefaultRP) receiveAgentMsg(ctx *actor.Context) error {
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

		if msg.Device.Type == device.Unspecified {
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

func (d *DefaultRP) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case groupActorStopped:
		delete(d.groups, msg.Ref)

	case SetGroupMaxSlots:
		d.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

	case SetGroupWeight:
		d.getOrCreateGroup(ctx, msg.Handler).weight = msg.Weight

	case AddTask:
		d.addTask(ctx, msg)

	case RemoveTask:
		d.resourcesReleased(ctx, msg.Handler)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// containerAssignment contains information for tasks have been assigned but not yet started.
type containerAssignment struct {
	req       *AddTask
	container *container
	agent     *agentState
	devices   []device.Device
}

// Summary summerizes a container assignment.
func (c containerAssignment) Summary() ContainerSummary {
	return ContainerSummary{
		TaskID: c.req.ID,
		ID:     c.container.id,
		Agent:  c.agent.handler.Address().Local(),
	}
}

// StartContainer notifies the agent to start a container.
func (c containerAssignment) StartContainer(ctx *actor.Context, spec image.TaskSpec) {
	handler := c.agent.handler
	spec.ContainerID = string(c.container.id)
	spec.TaskID = string(c.req.ID)
	spec.Devices = c.devices
	ctx.Tell(handler, sproto.StartTaskOnAgent{
		Task: c.req.Handler,
		StartContainer: aproto.StartContainer{
			Container: cproto.Container{
				Parent:  c.req.Handler.Address(),
				ID:      cproto.ID(c.container.id),
				State:   cproto.Assigned,
				Devices: c.devices,
			},
			Spec: image.ToContainerSpec(spec),
		},
	})
}

// KillContainer notifies the agent to kill the container.
func (c containerAssignment) KillContainer(ctx *actor.Context) {
	ctx.Tell(c.agent.handler, sproto.KillContainer{
		ContainerID: cproto.ID(c.container.id),
	})
}
