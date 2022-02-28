package resourcemanagers

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/resourcemanagers/agent"
	"github.com/determined-ai/determined/master/internal/resourcemanagers/provisioner"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// ResourcePool manages the agent and task lifecycles.
type ResourcePool struct {
	config *ResourcePoolConfig
	cert   *tls.Certificate

	scheduler        Scheduler
	fittingMethod    SoftConstraint
	provisioner      *actor.Ref
	slotsPerInstance int

	agents           map[*actor.Ref]bool
	agentStatesCache map[*actor.Ref]*agent.AgentState
	taskList         *taskList
	groups           map[*actor.Ref]*group
	queuePositions   jobSortState // secondary sort key initialized based on job submission time
	groupActorToID   map[*actor.Ref]model.JobID
	IDToGroupActor   map[model.JobID]*actor.Ref
	scalingInfo      *sproto.ScalingInfo

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

		agents:         make(map[*actor.Ref]bool),
		taskList:       newTaskList(),
		groups:         make(map[*actor.Ref]*group),
		queuePositions: initalizeJobSortState(),
		groupActorToID: make(map[*actor.Ref]model.JobID),
		IDToGroupActor: make(map[model.JobID]*actor.Ref),
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
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed Task"
	}

	ctx.Log().Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.TaskActor.Address(), msg.AllocationID,
	)
	if msg.IsUserVisible {
		if _, ok := rp.queuePositions[msg.JobID]; !ok {
			rp.queuePositions[msg.JobID] = initalizeQueuePosition(msg.JobSubmissionTime)
		}
		rp.groupActorToID[msg.Group] = msg.JobID
		rp.IDToGroupActor[msg.JobID] = msg.Group
	}
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
	fits := findFits(req, rp.agentStatesCache, rp.fittingMethod)

	if len(fits) == 0 {
		return false
	}

	allocations := make([]sproto.Reservation, 0, len(fits))
	for _, fit := range fits {
		container := newContainer(req, fit.Slots)
		resp := ctx.Ask(fit.Agent.Handler, agent.AllocateFreeDevices{
			Slots:       fit.Slots,
			ContainerID: container.id,
		}).Get()
		switch resp := resp.(type) {
		case agent.AllocateFreeDevicesResponse:
			devices := resp.Devices
			allocations = append(allocations, &containerReservation{
				req:       req,
				agent:     fit.Agent,
				container: container,
				devices:   devices,
			})
		case error:
			// Rollback previous allocations.
			ctx.Log().WithError(resp).Warnf("failed to allocate request %s", req.AllocationID)
			for _, allocation := range allocations {
				allocation := allocation.(*containerReservation)
				ctx.Tell(allocation.agent.Handler,
					agent.DeallocateContainer{ContainerID: allocation.container.id})
			}

			return false
		default:
			panic(fmt.Sprintf("bad AllocateFreeDevices response: %s", resp))
		}
	}

	allocated := sproto.ResourcesAllocated{
		ID: req.AllocationID, ResourcePool: rp.config.PoolName, Reservations: allocations,
	}
	rp.taskList.SetAllocations(req.TaskActor, &allocated)
	req.TaskActor.System().Tell(req.TaskActor, allocated)

	// Refresh state for the updated agents.
	allocatedAgents := make([]*actor.Ref, 0, len(allocations))
	for _, allocation := range allocations {
		allocation := allocation.(*containerReservation)
		allocatedAgents = append(allocatedAgents, allocation.agent.Handler)
	}

	rp.refreshAgentStateCacheFor(ctx, allocatedAgents)

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
			ctx.Tell(typed.agent.Handler, agent.DeallocateContainer{ContainerID: typed.container.id})
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
	for _, agentState := range rp.agentStatesCache {
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
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})
		return err

	case
		sproto.AddAgent,
		sproto.RemoveAgent:
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
		job.SetGroupPriority,
		job.RecoverJobPosition:
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
		rp.agentStatesCache = rp.fetchAgentStates(ctx)
		defer func() {
			rp.agentStatesCache = nil
		}()
		ctx.Respond(getResourceSummary(rp.agentStatesCache))

	case aproto.GetRPConfig:
		reschedule = false
		ctx.Respond(aproto.GetRPResponse{
			AgentReconnectWait:    rp.config.AgentReconnectWait,
			AgentReattachEnabled:  rp.config.AgentReattachEnabled,
			MaxZeroSlotContainers: rp.config.MaxAuxContainersPerAgent,
		})

	case schedulerTick:
		if rp.reschedule {
			rp.agentStatesCache = rp.fetchAgentStates(ctx)
			defer func() {
				rp.agentStatesCache = nil
			}()

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
		rp.agents[msg.Agent] = true

	case sproto.RemoveAgent:
		ctx.Log().Infof("removing agent: %s", msg.Agent.Address().Local())
		delete(rp.agents, msg.Agent)
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) moveJob(msg job.MoveJob) (job.RegisterJobPosition, error) {
	rp.queuePositions[job.TailAnchor] = initalizeQueuePosition(time.Now())
	newPos, err := computeNewJobPos(msg, rp.queuePositions)
	if err != nil {
		return job.RegisterJobPosition{}, err
	}
	rp.queuePositions[msg.ID] = newPos

	// actors.NotifyAfter(ctx, 0, schedulerTick{}) ?
	return job.RegisterJobPosition{
		JobID:       msg.ID,
		JobPosition: newPos.String(),
	}, nil
}

func (rp *ResourcePool) recoverJobPosition(msg job.RecoverJobPosition) error {
	if msg.JobPosition == "" {
		return nil
	}
	position, err := decimal.NewFromString(msg.JobPosition)
	if err != nil {
		return err
	}
	rp.queuePositions[msg.JobID] = position
	return nil
}

func (rp *ResourcePool) receiveJobQueueMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case job.GetJobQStats:
		ctx.Respond(*jobStats(rp.taskList))

	case job.GetJobQ:
		ctx.Respond(rp.scheduler.JobQInfo(rp))
	case job.MoveJob:
		response, err := rp.moveJob(msg)
		ctx.Respond(err)
		if err == nil {
			addr, ok := rp.IDToGroupActor[response.JobID]
			if ok {
				ctx.Tell(addr, response)
			}
		}

	case job.SetGroupWeight:
		rp.getOrCreateGroup(ctx, msg.Handler).weight = msg.Weight

	case job.SetGroupPriority:
		g := rp.getOrCreateGroup(ctx, msg.Handler)
		if (g.priority != nil && *g.priority == msg.Priority) ||
			rp.config.Scheduler.Priority == nil {
			return nil
		}
		ctx.Log().Infof("setting priority for group of %s to %d",
			msg.Handler.Address().String(), msg.Priority)
		g.priority = &msg.Priority
		jobID, ok := rp.groupActorToID[msg.Handler]
		if ok {
			time, err := getJobSubmissionTime(rp.taskList, jobID)
			if err != nil {
				ctx.Log().Errorf("failed to get job submission time: %s", err)
				return nil
			}
			rp.queuePositions[jobID] = initalizeQueuePosition(time)
		}
		// if !ok: we haven't seen the job yet or this group is not IsUserVisible
		// thus no need to reinitialize its queue position.

	case job.RecoverJobPosition:
		err := rp.recoverJobPosition(msg)
		if err != nil {
			ctx.Log().Errorf("failed to recover job position for job: %s", msg.JobID)
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case groupActorStopped:
		if jobID, ok := rp.groupActorToID[msg.Ref]; ok {
			delete(rp.queuePositions, jobID)
			delete(rp.groupActorToID, msg.Ref)
			delete(rp.IDToGroupActor, jobID)
		} else {
			ctx.Log().Errorf("group actor stopped but no job id found for group: %s", msg.Ref)
		}
		delete(rp.groups, msg.Ref)

	case sproto.SetGroupMaxSlots:
		rp.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

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

func (rp *ResourcePool) fetchAgentStates(ctx *actor.Context) map[*actor.Ref]*agent.AgentState {
	agents := make([]*actor.Ref, 0, len(rp.agents))

	for k := range rp.agents {
		agents = append(agents, k)
	}

	responses := ctx.AskAll(agent.GetAgentState{}, agents...).GetAll()

	result := make(map[*actor.Ref]*agent.AgentState, len(rp.agents))
	for ref, msg := range responses {
		switch msg := msg.(type) {
		case *agent.AgentState:
			result[ref] = msg
		case error:
			ctx.Log().WithError(msg).Warnf("failed to get agent state for agent %s", ref.Address().Local())
		default:
			ctx.Log().Warnf("bad agent state response for agent %s", ref.Address().Local())
		}
	}

	return result
}

func (rp *ResourcePool) refreshAgentStateCacheFor(ctx *actor.Context, agents []*actor.Ref) {
	responses := ctx.AskAll(agent.GetAgentState{}, agents...).GetAll()

	for ref, msg := range responses {
		switch msg := msg.(type) {
		case *agent.AgentState:
			rp.agentStatesCache[ref] = msg
		case error:
			ctx.Log().WithError(msg).Warnf("failed to get agent state for agent %s", ref.Address().Local())
			delete(rp.agentStatesCache, ref)
		default:
			ctx.Log().Warnf("bad agent state response for agent %s", ref.Address().Local())
			delete(rp.agentStatesCache, ref)
		}
	}
}

// containerReservation contains information for tasks have been allocated but not yet started.
type containerReservation struct {
	req       *sproto.AllocateRequest
	container *container
	agent     *agent.AgentState
	devices   []device.Device
}

// Summary summarizes a container allocation.
func (c containerReservation) Summary() sproto.ContainerSummary {
	return sproto.ContainerSummary{
		AllocationID: c.req.AllocationID,
		ID:           c.container.id,
		Agent:        c.agent.Handler.Address().Local(),
		Devices:      c.devices,
	}
}

// StartContainer notifies the agent to start a container.
func (c containerReservation) Start(
	ctx *actor.Context, spec tasks.TaskSpec, rri sproto.ReservationRuntimeInfo,
) {
	handler := c.agent.Handler
	spec.ContainerID = string(c.container.id)
	spec.AllocationID = string(c.req.AllocationID)
	spec.AllocationSessionToken = rri.Token
	spec.TaskID = string(c.req.TaskID)
	if spec.LoggingFields == nil {
		spec.LoggingFields = map[string]string{}
	}
	spec.LoggingFields["allocation_id"] = spec.AllocationID
	spec.LoggingFields["task_id"] = spec.TaskID
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
	ctx.Tell(c.agent.Handler, sproto.KillTaskContainer{
		ContainerID: c.container.id,
	})
}
