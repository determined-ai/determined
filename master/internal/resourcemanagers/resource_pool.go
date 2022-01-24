package resourcemanagers

import (
	"crypto/tls"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/resourcemanagers/provisioner"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type jobSortState = map[model.JobID]float64

// ResourcePool manages the agent and task lifecycles.
type ResourcePool struct {
	config *ResourcePoolConfig
	cert   *tls.Certificate

	scheduler        Scheduler
	fittingMethod    SoftConstraint
	provisioner      *actor.Ref
	slotsPerInstance int

	agents         map[*actor.Ref]*agentState
	taskList       *taskList
	groups         map[*actor.Ref]*group
	queuePositions jobSortState // secondary sort key initialized based on job submission time
	groupActorToID map[*actor.Ref]model.JobID
	scalingInfo    *sproto.ScalingInfo

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

		agents:         make(map[*actor.Ref]*agentState),
		taskList:       newTaskList(),
		groups:         make(map[*actor.Ref]*group),
		queuePositions: make(jobSortState),
		groupActorToID: make(map[*actor.Ref]model.JobID),
		scalingInfo:    &sproto.ScalingInfo{},

		reschedule: false,
	}
	return d
}

// func (rp *ResourcePool) groupByJobID(jobID model.JobID) *group {
// 	return nil
// }

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

	if len(msg.AllocationID) == 0 {
		msg.AllocationID = model.AllocationID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.TaskActor
	}
	rp.getOrCreateGroup(ctx, msg.Group)
	rp.groupActorToID[msg.Group] = msg.JobID
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed Task"
	}

	ctx.Log().Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.TaskActor.Address(), msg.AllocationID,
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

	allocations := make([]sproto.Reservation, 0, len(fits))
	for _, fit := range fits {
		container := newContainer(req, fit.Slots)
		allocations = append(allocations, &containerReservation{
			req:       req,
			agent:     fit.Agent,
			container: container,
			devices:   fit.Agent.allocateFreeDevices(fit.Slots, container.id),
		})
	}

	allocated := sproto.ResourcesAllocated{
		ID: req.AllocationID, ResourcePool: rp.config.PoolName, Reservations: allocations,
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
	if allocated := rp.taskList.GetAllocations(handler); allocated != nil {
		ctx.Log().Infof("resources are released for %s", handler.Address())
		for _, allocation := range allocated.Reservations {
			typed := allocation.(*containerReservation)
			typed.agent.deallocateContainer(typed.container.id)
		}
	}
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
		rp.taskList, rp.slotsPerInstance, rp.config.MaxAuxContainersPerAgent,
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
		err := rp.setupProvisioner(ctx)
		if err != nil {
			return err
		}
		if err = rp.restore(); err != nil {
			ctx.Log().Errorf("failed to restore resource pool: %s", err)
		}
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})
		return err

	case
		sproto.AddAgent,
		sproto.AddDevice,
		sproto.RemoveDevice,
		sproto.RemoveAgent,
		sproto.EnableAgent,
		sproto.DisableAgent:
		return rp.receiveAgentMsg(ctx)

	case
		groupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetTaskName,
		sproto.AllocateRequest,
		sproto.ResourcesReleased:
		return rp.receiveRequestMsg(ctx)

	case
		job.MoveJob,
		job.GetJobQ,
		job.GetJobQStats,
		job.SetGroupWeight,
		job.SetGroupPriority:
		return rp.receiveJobQueueMsg(ctx)

	case sproto.GetTaskHandler:
		reschedule = false
		ctx.Respond(getTaskHandler(rp.taskList, msg.ID))

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

	case aproto.GetRPConfig:
		reschedule = false
		ctx.Respond(aproto.GetRPResponse{
			AgentReconnectWait:   rp.config.AgentReconnectWait,
			AgentReattachEnabled: rp.config.AgentReattachEnabled,
		})

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

	case sproto.ValidateCommandResourcesRequest:
		fulfillable := true // Default to "true" when unknown.
		if rp.slotsPerInstance > 0 {
			fulfillable = rp.slotsPerInstance >= msg.Slots
		}
		ctx.Respond(sproto.ValidateCommandResourcesResponse{Fulfillable: fulfillable})

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
		rp.agents[msg.Agent] = newAgentState(msg, rp.config.MaxAuxContainersPerAgent)

	case sproto.AddDevice:
		ctx.Log().Infof("adding device: %s on %s", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := rp.agents[msg.Agent]
		check.Panic(check.True(ok, "error adding device, agent not found: %s", msg.Agent.Address()))
		state.devices[msg.Device] = msg.ContainerID

	case sproto.RemoveDevice:
		ctx.Log().Infof("removing device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := rp.agents[msg.Agent]
		check.Panic(check.True(ok, "error removing device, agent not found: %s", msg.Agent.Address()))
		delete(state.devices, msg.Device)

	case sproto.RemoveAgent:
		ctx.Log().Infof("removing agent: %s", msg.Agent.Address().Local())
		delete(rp.agents, msg.Agent)

	case sproto.EnableAgent:
		ctx.Log().Infof("enabling agent: %s", msg.Agent.Address().Local())
		state, ok := rp.agents[msg.Agent]
		check.Panic(check.True(ok, "error enabling agent, agent not found: %s", msg.Agent.Address()))
		state.enabled = true
		state.draining = false

	case sproto.DisableAgent:
		drain := msg.Drain
		drainStr := "disabling"
		if drain {
			drainStr = "draining"
		}
		ctx.Log().Infof("%s agent: %s", drainStr, msg.Agent.Address().Local())
		state, ok := rp.agents[msg.Agent]
		check.Panic(check.True(ok, "error %s agent, agent not found: %s", drainStr, msg.Agent.Address()))
		state.draining = drain
		state.enabled = false

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) moveJob(msg job.MoveJob) error {
	// get (groups and then) q positions for anchor, anchor.next or before

	// find out what is the job before or after the anchor
	// jobInfo := rp.scheduler.JobQInfo(rp)

	// find the mid point and update the id's qPosition.

	// condiontally check if rebalancing is needed.
	adjustPriorities(&rp.queuePositions)

	if err := rp.persist(); err != nil {
		return err
	}
	// actors.NotifyAfter(ctx, 0, schedulerTick{}) ?
	return nil
}

func (rp *ResourcePool) receiveJobQueueMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case job.GetJobQStats:
		ctx.Respond(*jobStats(rp.taskList))
	case job.GetJobQ:
		ctx.Respond(rp.scheduler.JobQInfo(rp))
	case job.MoveJob:
		ctx.Respond(rp.moveJob(msg))
	case job.SetGroupWeight:
		rp.getOrCreateGroup(ctx, msg.Handler).weight = msg.Weight

	case job.SetGroupPriority:
		g := rp.getOrCreateGroup(ctx, msg.Handler)
		if msg.Priority != nil {
			g.priority = msg.Priority
		}

		if rp.config.Scheduler.Priority != nil {
			ctx.Log().Infof("setting priority for group of %s to %d",
				msg.Handler.Address().String(), *g.priority)
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) persist() error {
	fmt.Println("TODO persist rp", rp.config.PoolName)
	return nil // TODO
}

func (rp *ResourcePool) restore() error {
	fmt.Println("TODO restore rp", rp.config.PoolName)
	return nil // TODO
}

func (rp *ResourcePool) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case groupActorStopped:
		if jobID, ok := rp.groupActorToID[msg.Ref]; ok {
			delete(rp.queuePositions, jobID)
			delete(rp.groupActorToID, msg.Ref)
			if err := rp.persist(); err != nil {
				ctx.Log().Errorf("error persisting queuePositions: %s", err)
				// CHECK fail?
			}
		} else {
			ctx.Log().Errorf("group actor stopped but no job id found for group: %s", msg.Ref)
		}
		delete(rp.groups, msg.Ref)

	case sproto.SetGroupMaxSlots:
		rp.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

	case sproto.SetTaskName:
		rp.receiveSetTaskName(ctx, msg)

	case sproto.AllocateRequest:
		if _, ok := rp.queuePositions[msg.JobID]; !ok {
			rp.queuePositions[msg.JobID] = msg.InitialQueuePosition()
			if err := rp.persist(); err != nil {
				ctx.Log().Errorf("error persisting queue positions: %s", err)
				// CHECK: fail?
			}
		}
		rp.addTask(ctx, msg)

	case sproto.ResourcesReleased:
		rp.resourcesReleased(ctx, msg.TaskActor)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// containerReservation contains information for tasks have been allocated but not yet started.
type containerReservation struct {
	req       *sproto.AllocateRequest
	container *container
	agent     *agentState
	devices   []device.Device
}

// Summary summarizes a container allocation.
func (c containerReservation) Summary() sproto.ContainerSummary {
	return sproto.ContainerSummary{
		AllocationID: c.req.AllocationID,
		ID:           c.container.id,
		Agent:        c.agent.handler.Address().Local(),
		Devices:      c.devices,
	}
}

// StartContainer notifies the agent to start a container.
func (c containerReservation) Start(
	ctx *actor.Context, spec tasks.TaskSpec, rri sproto.ReservationRuntimeInfo,
) {
	handler := c.agent.handler
	spec.ContainerID = string(c.container.id)
	spec.AllocationID = string(c.req.AllocationID)
	spec.AllocationSessionToken = rri.Token
	spec.TaskID = string(c.req.TaskID)
	spec.UseHostMode = rri.IsMultiAgent
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
			Spec: spec.ToDockerSpec(),
		},
	})
}

// KillContainer notifies the agent to kill the container.
func (c containerReservation) Kill(ctx *actor.Context) {
	ctx.Tell(c.agent.handler, sproto.KillTaskContainer{
		ContainerID: c.container.id,
	})
}
