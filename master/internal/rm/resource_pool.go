package rm

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/provisioner"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// ResourcePool manages the agent and task lifecycles.
type ResourcePool struct {
	config *config.ResourcePoolConfig
	cert   *tls.Certificate

	scheduler        Scheduler
	fittingMethod    SoftConstraint
	provisioner      *actor.Ref
	slotsPerInstance int

	agents           map[*actor.Ref]bool
	agentStatesCache map[*actor.Ref]*AgentState
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

	db db.DB
}

// GetResourceSummary is a message to request a summary of the resources used by the
// resource pool (agents, slots, cpu containers).
type GetResourceSummary struct{}

// NewResourcePool initializes a new empty default resource provider.
func NewResourcePool(
	config *config.ResourcePoolConfig,
	db db.DB,
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
		queuePositions: initalizeJobSortState(false),
		groupActorToID: make(map[*actor.Ref]model.JobID),
		IDToGroupActor: make(map[model.JobID]*actor.Ref),
		scalingInfo:    &sproto.ScalingInfo{},

		reschedule: false,
		db:         db,
	}
	return d
}

func (rp *ResourcePool) setupProvisioner(ctx *actor.Context) error {
	if rp.config.Provider == nil {
		ctx.Log().Infof("not enabling provisioner for resource pool: %s", rp.config.PoolName)
		return nil
	}
	p, pRef, err := provisioner.Setup(ctx, rp.config.Provider, rp.config.PoolName, rp.cert, rp.db)
	if err != nil {
		return errors.Wrapf(err, "cannot create resource pool: %s", rp.config.PoolName)
	}
	rp.slotsPerInstance = p.SlotsPerInstance()
	rp.provisioner = pRef
	return nil
}

func (rp *ResourcePool) allocateRequest(ctx *actor.Context, msg sproto.AllocateRequest) {
	rp.notifyOnStop(ctx, msg.AllocationRef, sproto.ResourcesReleased{
		AllocationRef: msg.AllocationRef,
	})
	log := ctx.Log().WithField("allocation-id", msg.AllocationID)

	if len(msg.AllocationID) == 0 {
		msg.AllocationID = model.AllocationID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.AllocationRef
	}
	rp.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed Task"
	}

	log.Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.AllocationRef.Address(), msg.AllocationID,
	)
	if msg.IsUserVisible {
		if _, ok := rp.queuePositions[msg.JobID]; !ok {
			rp.queuePositions[msg.JobID] = initalizeQueuePosition(msg.JobSubmissionTime, false)
		}
		rp.groupActorToID[msg.Group] = msg.JobID
		rp.IDToGroupActor[msg.JobID] = msg.Group
	}

	if msg.Restore {
		err := rp.restoreResources(ctx, &msg)
		if err != nil {
			log.WithError(err).Error("error restoring resources")

			// Clear out the state / close and terminate the allocation.
			rf := sproto.ResourcesFailure{
				FailureType: sproto.RestoreError,
				ErrMsg:      err.Error(),
				ExitCode:    nil,
			}
			ctx.Tell(msg.AllocationRef, rf)

			return
		}
	}

	rp.taskList.AddTask(&msg)
}

func (rp *ResourcePool) restoreResources(
	ctx *actor.Context, req *sproto.AllocateRequest,
) error {
	rp.agentStatesCache = rp.fetchAgentStates(ctx)
	defer func() {
		rp.agentStatesCache = nil
	}()

	allocationID := req.AllocationID

	containerSnapshots := []ContainerSnapshot{}
	err := db.Bun().NewSelect().Model(&containerSnapshots).
		Relation("ResourcesWithState").
		Where("resources_with_state.allocation_id = ?", allocationID).
		Scan(context.TODO())
	if err != nil {
		return err
	}

	if len(containerSnapshots) == 0 {
		return errors.New("0 container snapshots")
	}

	resources := sproto.ResourceList{}

	agentStateMap := map[aproto.ID]*AgentState{}

	for agentRef := range rp.agentStatesCache {
		agentStateMap[aproto.ID(agentRef.Address().Local())] = rp.agentStatesCache[agentRef]
	}

	for _, cs := range containerSnapshots {
		agentState, ok := agentStateMap[cs.AgentID]
		if !ok {
			return errors.New(fmt.Sprintf("can't find restorable agent %s", cs.AgentID))
		}

		cr := containerResources{
			req:         req,
			agent:       agentState,
			devices:     cs.Devices,
			containerID: cs.ID,
			started:     cs.ResourcesWithState.Started,
			exited:      cs.ResourcesWithState.Exited,
		}
		resources[cr.Summary().ResourcesID] = &cr
	}

	allocated := sproto.ResourcesAllocated{
		ID:           req.AllocationID,
		ResourcePool: rp.config.PoolName,
		Resources:    resources,
		Recovered:    true,
	}

	rp.taskList.AddTask(req)
	rp.taskList.SetAllocations(req.AllocationRef, &allocated)
	ctx.Tell(req.AllocationRef, allocated.Clone())

	return nil
}

func (rp *ResourcePool) receiveSetTaskName(ctx *actor.Context, msg sproto.SetAllocationName) {
	if task, found := rp.taskList.GetAllocationByHandler(msg.AllocationRef); found {
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

	resources := make([]*containerResources, 0, len(fits))
	rollback := false

	defer func() {
		if rollback {
			// Rollback previous allocations.
			for _, resource := range resources {
				ctx.Tell(resource.agent.Handler,
					DeallocateContainer{ContainerID: resource.containerID})
			}
		}
	}()

	for _, fit := range fits {
		containerID := cproto.NewID()
		rr := ctx.Ask(fit.Agent.Handler, AllocateFreeDevices{
			Slots:       fit.Slots,
			ContainerID: containerID,
		})
		var resp actor.Message
		if err := rr.Error(); err != nil {
			resp = errors.New("ask error in AllocateFreeDevices")
		} else {
			resp = rr.Get()
			if resp == nil {
				resp = errors.New("nil AllocateFreeDevices response")
			}
		}

		switch resp := resp.(type) {
		case AllocateFreeDevicesResponse:
			devices := resp.Devices
			resources = append(resources, &containerResources{
				req:         req,
				agent:       fit.Agent,
				containerID: containerID,
				devices:     devices,
			})
		case error:
			// Rollback previous allocations.
			ctx.Log().WithError(resp).Warnf("failed to allocate request %s", req.AllocationID)
			rollback = true
			return false
		default:
			panic(fmt.Sprintf("bad AllocateFreeDevices response: %+v", resp))
		}
	}

	// Persist allocation_resources and container_resources.
	for _, cr := range resources {
		rs := taskmodel.NewResourcesState(cr, -1)
		if err := rs.Persist(); err != nil {
			ctx.Log().WithError(err).Error("persistence failure")
			rollback = true
			return false
		}
		if err := cr.Persist(); err != nil {
			ctx.Log().WithError(err).Error("persistence failure")
			rollback = true
			return false
		}
	}

	sprotoResources := sproto.ResourceList{}
	for _, v := range resources {
		sprotoResources[v.Summary().ResourcesID] = v
	}

	allocated := sproto.ResourcesAllocated{
		ID:                req.AllocationID,
		ResourcePool:      rp.config.PoolName,
		Resources:         sprotoResources,
		JobSubmissionTime: req.JobSubmissionTime,
	}
	rp.taskList.SetAllocations(req.AllocationRef, &allocated)
	ctx.Tell(req.AllocationRef, allocated)

	// Refresh state for the updated agents.
	allocatedAgents := make([]*actor.Ref, 0, len(resources))
	for _, allocation := range resources {
		allocatedAgents = append(allocatedAgents, allocation.agent.Handler)
	}

	rp.refreshAgentStateCacheFor(ctx, allocatedAgents)

	ctx.Log().Infof("allocated resources to %s", req.AllocationRef.Address())

	return true
}

func (rp *ResourcePool) releaseResource(ctx *actor.Context, handler *actor.Ref) {
	ctx.Log().Infof("releasing resources taken by %s", handler.Address())
	handler.System().Tell(handler, sproto.ReleaseResources{ResourcePool: rp.config.PoolName})
}

func (rp *ResourcePool) resourcesReleased(
	ctx *actor.Context,
	msg sproto.ResourcesReleased,
) {
	switch a := rp.taskList.GetAllocations(msg.AllocationRef); {
	case a == nil:
		rp.taskList.RemoveTaskByHandler(msg.AllocationRef)
	case msg.ResourcesID != nil:
		ctx.Log().Infof(
			"resources %v are released for %s",
			*msg.ResourcesID, msg.AllocationRef.Address())
		for rID, r := range a.Resources {
			if r.Summary().ResourcesID != *msg.ResourcesID {
				continue
			}

			typed := r.(*containerResources)
			ctx.Tell(typed.agent.Handler, DeallocateContainer{ContainerID: typed.containerID})
			delete(a.Resources, rID)
			break
		}
	default:
		ctx.Log().Infof("all resources are released for %s", msg.AllocationRef.Address())
		for _, r := range a.Resources {
			typed := r.(*containerResources)
			ctx.Tell(typed.agent.Handler, DeallocateContainer{ContainerID: typed.containerID})
		}
		rp.taskList.RemoveTaskByHandler(msg.AllocationRef)
	}
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
		rp.taskList, rp.groups, rp.slotsPerInstance, rp.config.MaxAuxContainersPerAgent,
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
		sproto.RemoveAgent,
		sproto.UpdateAgent:
		return rp.receiveAgentMsg(ctx)

	case
		groupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetAllocationName,
		sproto.AllocateRequest,
		sproto.ResourcesReleased:
		return rp.receiveRequestMsg(ctx)

	case
		sproto.MoveJob,
		sproto.GetJobQ,
		sproto.GetJobQStats,
		sproto.SetGroupWeight,
		sproto.SetGroupPriority,
		sproto.RecoverJobPosition,
		sproto.DeleteJob:
		return rp.receiveJobQueueMsg(ctx)

	case sproto.GetAllocationHandler:
		reschedule = false
		ctx.Respond(getTaskHandler(rp.taskList, msg.ID))

	case sproto.GetAllocationSummary:
		reschedule = false
		if resp := getTaskSummary(
			rp.taskList, msg.ID, rp.groups, rp.config.Scheduler.GetType()); resp != nil {
			ctx.Respond(*resp)
		}

	case sproto.GetAllocationSummaries:
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
	var agentID string
	switch msg := ctx.Message().(type) {
	// TODO(ilia): I hope go will have a good way to do this one day.
	case sproto.AddAgent:
		agentID = msg.Agent.Address().Local()
	case sproto.RemoveAgent:
		agentID = msg.Agent.Address().Local()
	case sproto.UpdateAgent:
		agentID = msg.Agent.Address().Local()
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	logger := ctx.Log().WithField("agent-id", agentID)

	switch msg := ctx.Message().(type) {
	case sproto.AddAgent:
		// agent_id is logged in the unstructured message because this log line is used by
		// some scripts that parse the logs for GPU usage stats.
		logger.Infof("adding agent: %s", agentID)
		rp.agents[msg.Agent] = true
		err := rp.updateAgentStartStats(rp.config.PoolName, agentID, msg.Slots)
		if err != nil {
			logger.WithError(err).Error("failed to update agent start stats")
		}
	case sproto.RemoveAgent:
		logger.Infof("removing agent: %s", agentID)

		delete(rp.agents, msg.Agent)
		err := rp.updateAgentEndStats(agentID)
		if err != nil {
			logger.WithError(err).Error("failed to update agent end stats")
		}
	case sproto.UpdateAgent:
		_, ok := rp.agents[msg.Agent]
		if !ok {
			logger.Warn("received update on unknown agent")
		} else {
			logger.Debug("updating agent")
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (rp *ResourcePool) moveJob(
	ctx *actor.Context,
	jobID model.JobID,
	anchorID model.JobID,
	aheadOf bool,
) error {
	if anchorID == "" || jobID == "" || anchorID == jobID {
		return nil
	}

	// check whether the msg belongs to this resource pool or not.
	// job messages to agent rm are forwarded to all resource pools.
	if _, ok := rp.queuePositions[jobID]; !ok {
		return nil
	}

	if rp.config.Scheduler.GetType() != config.PriorityScheduling {
		return fmt.Errorf("unable to perform operation on resource pool with %s",
			rp.config.Scheduler.GetType())
	}

	groupAddr, ok := rp.IDToGroupActor[jobID]
	if !ok {
		return sproto.ErrJobNotFound(jobID)
	}
	if _, ok := rp.queuePositions[anchorID]; !ok {
		return sproto.ErrJobNotFound(anchorID)
	}

	prioChange, secondAnchor, anchorPriority := findAnchor(jobID, anchorID, aheadOf, rp.taskList,
		rp.groups, rp.queuePositions, false)

	if secondAnchor == "" {
		return fmt.Errorf("unable to move job with ID %s", jobID)
	}

	if secondAnchor == jobID {
		return nil
	}

	if prioChange {
		oldPriority := *rp.groups[groupAddr].priority
		err := rp.setGroupPriority(ctx, sproto.SetGroupPriority{
			Priority:     anchorPriority,
			ResourcePool: rp.config.PoolName,
			Handler:      rp.IDToGroupActor[jobID],
		})
		if err != nil {
			return err
		}

		resp := ctx.Ask(rp.IDToGroupActor[jobID], sproto.NotifyRMPriorityChange{
			Priority: anchorPriority,
		})
		if resp.Error() != nil {
			_ = rp.setGroupPriority(ctx, sproto.SetGroupPriority{
				Priority:     oldPriority,
				ResourcePool: rp.config.PoolName,
				Handler:      rp.IDToGroupActor[jobID],
			})
			return resp.Error()
		}
		if !needMove(
			rp.queuePositions[jobID],
			rp.queuePositions[anchorID],
			rp.queuePositions[secondAnchor],
			aheadOf,
		) {
			return nil
		}
	}

	msg, err := rp.queuePositions.SetJobPosition(jobID, anchorID, secondAnchor, aheadOf, false)
	if err != nil {
		return err
	}

	ctx.Tell(groupAddr, msg)

	return nil
}

func (rp *ResourcePool) receiveJobQueueMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.GetJobQStats:
		ctx.Respond(jobStats(rp.taskList))

	case sproto.GetJobQ:
		ctx.Respond(rp.scheduler.JobQInfo(rp))

	case sproto.MoveJob:
		err := rp.moveJob(ctx, msg.ID, msg.Anchor, msg.Ahead)
		ctx.Respond(err)

	case sproto.SetGroupWeight:
		rp.getOrCreateGroup(ctx, msg.Handler).weight = msg.Weight

	case sproto.SetGroupPriority:
		err := rp.setGroupPriority(ctx, msg)
		ctx.Respond(err)

	case sproto.RecoverJobPosition:
		rp.queuePositions.RecoverJobPosition(msg.JobID, msg.JobPosition)

	case sproto.DeleteJob:
		// For now, there is nothing to cleanup in determined-agents world.
		ctx.Respond(sproto.EmptyDeleteJobResponse())

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) setGroupPriority(ctx *actor.Context, msg sproto.SetGroupPriority) error {
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
		rp.queuePositions[jobID] = initalizeQueuePosition(time, false)
	}
	return nil
}

func (rp *ResourcePool) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case groupActorStopped:
		if jobID, ok := rp.groupActorToID[msg.Ref]; ok {
			delete(rp.queuePositions, jobID)
			delete(rp.IDToGroupActor, jobID)
		}
		delete(rp.groupActorToID, msg.Ref)
		delete(rp.groups, msg.Ref)

	case sproto.SetGroupMaxSlots:
		rp.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

	case sproto.SetAllocationName:
		rp.receiveSetTaskName(ctx, msg)

	case sproto.AllocateRequest:
		rp.allocateRequest(ctx, msg)

	case sproto.ResourcesReleased:
		rp.resourcesReleased(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) updateAgentStartStats(
	poolName string, agentID string, slots int,
) error {
	return rp.db.RecordAgentStats(&model.AgentStats{
		ResourcePool: poolName,
		AgentID:      agentID,
		Slots:        slots,
	})
}

func (rp *ResourcePool) updateAgentEndStats(agentID string) error {
	return db.EndAgentStats(&model.AgentStats{
		AgentID: agentID,
	})
}

func (rp *ResourcePool) fetchAgentStates(ctx *actor.Context) map[*actor.Ref]*AgentState {
	agents := maps.Keys(rp.agents)

	responses := ctx.AskAll(GetAgentState{}, agents...).GetAll()

	result := make(map[*actor.Ref]*AgentState, len(rp.agents))
	for ref, msg := range responses {
		switch msg := msg.(type) {
		case *AgentState:
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
	responses := ctx.AskAll(GetAgentState{}, agents...).GetAll()

	for ref, msg := range responses {
		switch msg := msg.(type) {
		case *AgentState:
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

// containerResources contains information for tasks have been allocated but not yet started.
type containerResources struct {
	req         *sproto.AllocateRequest
	agent       *AgentState
	devices     []device.Device
	containerID cproto.ID
	started     *sproto.ResourcesStarted
	exited      *sproto.ResourcesStopped
}

// Summary summarizes a container allocation.
func (c containerResources) Summary() sproto.ResourcesSummary {
	return sproto.ResourcesSummary{
		ResourcesID:   sproto.ResourcesID(c.containerID),
		ResourcesType: sproto.ResourcesTypeDockerContainer,
		AllocationID:  c.req.AllocationID,
		AgentDevices: map[aproto.ID][]device.Device{
			aproto.ID(c.agent.Handler.Address().Local()): c.devices,
		},

		ContainerID: &c.containerID,
		Started:     c.started,
		Exited:      c.exited,
	}
}

// StartContainer notifies the agent to start a container.
func (c containerResources) Start(
	ctx *actor.Context, logCtx logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
) error {
	handler := c.agent.Handler
	spec.ContainerID = string(c.containerID)
	spec.ResourcesID = string(c.containerID)
	spec.AllocationID = string(c.req.AllocationID)
	spec.AllocationSessionToken = rri.Token
	spec.TaskID = string(c.req.TaskID)
	if spec.LoggingFields == nil {
		spec.LoggingFields = map[string]string{}
	}
	spec.LoggingFields["allocation_id"] = spec.AllocationID
	spec.LoggingFields["task_id"] = spec.TaskID
	spec.ExtraEnvVars[sproto.ResourcesTypeEnvVar] = string(sproto.ResourcesTypeDockerContainer)
	spec.UseHostMode = rri.IsMultiAgent
	spec.Devices = c.devices
	// Write the real DET_UNIQUE_PORT_OFFSET value now that we know which devices to use.
	spec.ExtraEnvVars["DET_UNIQUE_PORT_OFFSET"] = strconv.Itoa(tasks.UniquePortOffset(spec.Devices))
	return ctx.Ask(handler, sproto.StartTaskContainer{
		TaskActor: c.req.AllocationRef,
		StartContainer: aproto.StartContainer{
			Container: cproto.Container{
				Parent:  c.req.AllocationRef.Address(),
				ID:      c.containerID,
				State:   cproto.Assigned,
				Devices: c.devices,
			},
			Spec: spec.ToDockerSpec(),
		},
		LogContext: logCtx,
	}).Error()
}

// Kill notifies the agent to kill the container.
func (c containerResources) Kill(ctx *actor.Context, logCtx logger.Context) {
	ctx.Tell(c.agent.Handler, sproto.KillTaskContainer{
		ContainerID: c.containerID,
		LogContext:  logCtx,
	})
}

// Single asserts there's a single element in the map and take it.
func Single[K comparable, V any](m map[K]V) (kr K, vr V, ok bool) {
	// TODO(ilia): move it into a shared utilities package when
	// it'll be used elsewhere.
	if len(m) != 1 {
		return kr, vr, false
	}
	for k, v := range m {
		kr = k
		vr = v
	}
	return kr, vr, true
}

func (c containerResources) Persist() error {
	summary := c.Summary()

	agentID, _, ok := Single(summary.AgentDevices)
	if !ok {
		return fmt.Errorf("%d agents in containerResources summary", len(summary.AgentDevices))
	}

	snapshot := ContainerSnapshot{
		ResourceID: summary.ResourcesID,
		ID:         c.containerID,
		AgentID:    agentID,
	}
	_, err := db.Bun().NewInsert().Model(&snapshot).Exec(context.TODO())
	return err
}
