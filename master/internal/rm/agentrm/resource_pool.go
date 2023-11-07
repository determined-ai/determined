package agentrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/rm/agentrm/provisioner"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
)

// resourcePool manages the agent and task lifecycles.
type resourcePool struct {
	mu sync.Mutex

	config *config.ResourcePoolConfig
	cert   *tls.Certificate

	scheduler        Scheduler
	fittingMethod    SoftConstraint
	slotsPerInstance int

	provisioner      *provisioner.Provisioner
	provisionerError error

	agents           map[*actor.Ref]bool
	agentStatesCache map[*actor.Ref]*agentState
	taskList         *tasklist.TaskList
	groups           map[model.JobID]*tasklist.Group
	queuePositions   tasklist.JobSortState // secondary sort key based on job submission time
	scalingInfo      *sproto.ScalingInfo

	reschedule bool

	// Track notifyOnStop for testing purposes.
	saveNotifications bool
	notifications     []<-chan struct{}

	db db.DB
}

// getResourceSummary is a message to request a summary of the resources used by the
// resource pool (agents, slots, cpu containers).
type getResourceSummary struct{}

// schedulerTick periodically triggers the scheduler to act.
type schedulerTick struct{}

// actionCoolDown is the rate limit for scheduler action.
const actionCoolDown = 500 * time.Millisecond

// newResourcePool initializes a new empty default resource provider.
func newResourcePool(
	config *config.ResourcePoolConfig,
	db db.DB,
	cert *tls.Certificate,
	scheduler Scheduler,
	fittingMethod SoftConstraint,
) *resourcePool {
	d := &resourcePool{
		config: config,
		cert:   cert,

		scheduler:     scheduler,
		fittingMethod: fittingMethod,

		agents:         make(map[*actor.Ref]bool),
		taskList:       tasklist.New(),
		groups:         make(map[model.JobID]*tasklist.Group),
		queuePositions: tasklist.InitializeJobSortState(false),
		scalingInfo:    &sproto.ScalingInfo{},

		reschedule: false,
		db:         db,
	}
	return d
}

func (rp *resourcePool) setupProvisioner(ctx *actor.Context) error {
	if rp.config.Provider == nil {
		ctx.Log().Infof("not enabling provisioner for resource pool: %s", rp.config.PoolName)
		return nil
	}
	p, err := provisioner.Setup(ctx, rp.config.Provider, rp.config.PoolName, rp.cert, rp.db)
	if err != nil {
		return errors.Wrapf(err, "cannot create resource pool: %s", rp.config.PoolName)
	}
	rp.slotsPerInstance = p.SlotsPerInstance()
	rp.provisioner = p
	return nil
}

func (rp *resourcePool) allocateRequest(ctx *actor.Context, msg sproto.AllocateRequest) {
	log := ctx.Log().
		WithField("allocation-id", msg.AllocationID).
		WithField("restoring", msg.Restore)

	if len(msg.AllocationID) == 0 {
		msg.AllocationID = model.AllocationID(uuid.New().String())
	}
	rp.getOrCreateGroup(msg.JobID)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed Task"
	}

	log.WithField("restore", msg.Restore).Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.Name, msg.AllocationID,
	)
	if msg.IsUserVisible {
		if _, ok := rp.queuePositions[msg.JobID]; !ok {
			rp.queuePositions[msg.JobID] = tasklist.InitializeQueuePosition(
				msg.JobSubmissionTime,
				false,
			)
		}
	}

	if msg.Restore {
		err := rp.restoreResources(ctx, &msg)
		if err != nil {
			log.WithError(err).Error("error restoring resources")

			// Clear out the state / close and terminate the allocation.
			rmevents.Publish(msg.AllocationID, &sproto.ResourcesFailure{
				FailureType: sproto.RestoreError,
				ErrMsg:      err.Error(),
				ExitCode:    nil,
			})
			return
		}
	}

	rp.taskList.AddTask(&msg)
}

func (rp *resourcePool) restoreResources(
	ctx *actor.Context, req *sproto.AllocateRequest,
) error {
	rp.agentStatesCache = rp.fetchAgentStates(ctx)
	defer func() {
		rp.agentStatesCache = nil
	}()

	allocationID := req.AllocationID

	containerSnapshots := []containerSnapshot{}
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

	agentStateMap := map[aproto.ID]*agentState{}

	for agentRef := range rp.agentStatesCache {
		agentStateMap[aproto.ID(agentRef.Address().Local())] = rp.agentStatesCache[agentRef]
	}

	for _, cs := range containerSnapshots {
		agentState, ok := agentStateMap[cs.AgentID]
		if !ok {
			return fmt.Errorf("can't find restorable agent %s", cs.AgentID)
		}

		cr := containerResources{
			system:      ctx.Self().System(),
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
	rp.taskList.AddAllocation(req.AllocationID, &allocated)
	rmevents.Publish(req.AllocationID, allocated.Clone())

	return nil
}

func (rp *resourcePool) receiveSetTaskName(ctx *actor.Context, msg sproto.SetAllocationName) {
	if task, found := rp.taskList.TaskByID(msg.AllocationID); found {
		task.Name = msg.Name
	}
}

// allocateResources assigns resources based on a request and notifies the request
// handler of the assignment. It returns true if it is successfully allocated.
func (rp *resourcePool) allocateResources(ctx *actor.Context, req *sproto.AllocateRequest) bool {
	fits := findFits(
		req,
		rp.agentStatesCache,
		rp.fittingMethod,
		rp.config.Scheduler.AllowHeterogeneousFits,
	)

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
					deallocateContainer{containerID: resource.containerID})
			}
		}
	}()

	for _, fit := range fits {
		containerID := cproto.NewID()
		rr := ctx.Ask(fit.Agent.Handler, allocateFreeDevices{
			slots:       fit.Slots,
			containerID: containerID,
		})
		var resp actor.Message
		if err := rr.Error(); err != nil {
			resp = errors.New("ask error in allocateFreeDevices")
		} else {
			resp = rr.Get()
			if resp == nil {
				resp = errors.New("nil allocateFreeDevices response")
			}
		}

		switch resp := resp.(type) {
		case allocateFreeDevicesResponse:
			devices := resp.devices
			resources = append(resources, &containerResources{
				system:      ctx.Self().System(),
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
			panic(fmt.Sprintf("bad allocateFreeDevices response: %+v", resp))
		}
	}

	// persist allocation_resources and container_resources.
	for _, cr := range resources {
		rs := taskmodel.NewResourcesState(cr, -1)
		if err := rs.Persist(); err != nil {
			ctx.Log().WithError(err).Error("persistence failure")
			rollback = true
			return false
		}
		if err := cr.persist(); err != nil {
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
	rp.taskList.AddAllocation(req.AllocationID, &allocated)
	rmevents.Publish(req.AllocationID, allocated.Clone())

	// Refresh state for the updated agents.
	allocatedAgents := make([]*actor.Ref, 0, len(resources))
	for _, allocation := range resources {
		allocatedAgents = append(allocatedAgents, allocation.agent.Handler)
	}

	rp.refreshAgentStateCacheFor(ctx, allocatedAgents)

	ctx.Log().Infof("allocated resources to %s", req.Name)

	return true
}

func (rp *resourcePool) releaseResource(ctx *actor.Context, aID model.AllocationID) {
	ctx.Log().Infof("releasing resources taken by %s (preempted by the scheduler)", aID)
	rmevents.Publish(aID, &sproto.ReleaseResources{Reason: "preempted by the scheduler"})
}

func (rp *resourcePool) resourcesReleased(
	ctx *actor.Context,
	msg sproto.ResourcesReleased,
) {
	_, ok := rp.taskList.TaskByID(msg.AllocationID)
	if !ok {
		ctx.Log().Debugf("ignoring release for task not allocated to pool %s", msg.AllocationID)
		return
	}

	switch allocated := rp.taskList.Allocation(msg.AllocationID); {
	case allocated == nil:
		ctx.Log().Infof("released before allocated for %s", msg.AllocationID)
		rp.taskList.RemoveTaskByID(msg.AllocationID)
		rmevents.Publish(msg.AllocationID, sproto.ResourcesReleasedEvent{})
	case msg.ResourcesID != nil:
		ctx.Log().Infof("incrementally released resources %v for %s", *msg.ResourcesID, msg.AllocationID)
		for rID, r := range allocated.Resources {
			if r.Summary().ResourcesID != *msg.ResourcesID {
				continue
			}

			typed := r.(*containerResources)
			ctx.Tell(typed.agent.Handler, deallocateContainer{containerID: typed.containerID})
			delete(allocated.Resources, rID)
			break
		}
	default:
		ctx.Log().Infof("all resources are released for %s", msg.AllocationID)
		for _, r := range allocated.Resources {
			typed := r.(*containerResources)
			ctx.Tell(typed.agent.Handler, deallocateContainer{containerID: typed.containerID})
		}
		rp.taskList.RemoveTaskByID(msg.AllocationID)
		rmevents.Publish(msg.AllocationID, sproto.ResourcesReleasedEvent{})
	}
}

func (rp *resourcePool) getOrCreateGroup(jobID model.JobID) *tasklist.Group {
	if g, ok := rp.groups[jobID]; ok {
		return g
	}
	g := &tasklist.Group{JobID: jobID, Weight: 1}

	if rp.config.Scheduler.Priority != nil {
		if rp.config.Scheduler.Priority.DefaultPriority == nil {
			panic("default priority is not configured")
		}
		g.Priority = rp.config.Scheduler.Priority.DefaultPriority
	}

	rp.groups[jobID] = g
	tasklist.GroupPriorityChangeRegistry.OnDelete(jobID, func() {
		rp.JobStopped(jobID)
	})
	return g
}

func (rp *resourcePool) updateScalingInfo() bool {
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

func (rp *resourcePool) sendScalingInfo(ctx *actor.Context) {
	if rp.provisioner != nil && rp.updateScalingInfo() {
		rp.provisioner.UpdateScalingInfo(rp.scalingInfo)
	}
}

// Receive implements the actor.Actor interface.
func (rp *resourcePool) Receive(ctx *actor.Context) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

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

	case sproto.GetAllocationSummary:
		reschedule = false
		if resp := rp.taskList.TaskSummary(
			msg.ID, rp.groups, rp.config.Scheduler.GetType()); resp != nil {
			ctx.Respond(*resp)
		}

	case sproto.GetAllocationSummaries:
		reschedule = false
		ctx.Respond(rp.taskList.TaskSummaries(rp.groups, rp.config.Scheduler.GetType()))

	case getResourceSummary:
		reschedule = false
		rp.agentStatesCache = rp.fetchAgentStates(ctx)
		defer func() {
			rp.agentStatesCache = nil
		}()
		ctx.Respond(resourceSummaryFromAgentStates(rp.agentStatesCache))

	case sproto.CapacityCheck:
		reschedule = false
		var totalSlots int
		blockedNodeSet := set.New[string]()
		if msg.TaskID != nil {
			blockedNodes, err := logpattern.GetBlockedNodes(context.TODO(), *msg.TaskID)
			if err != nil {
				ctx.Respond(err)
				return nil
			}
			blockedNodeSet = set.FromSlice(blockedNodes)
		}
		rp.agentStatesCache = rp.fetchAgentStates(ctx)
		defer func() {
			rp.agentStatesCache = nil
		}()

		switch {
		case rp.config.Provider == nil:
			for _, a := range rp.agentStatesCache {
				if !blockedNodeSet.Contains(a.Handler.Address().Local()) {
					totalSlots += len(a.slotStates)
				}
			}
		case rp.config.Provider.AWS != nil:
			totalSlots = rp.config.Provider.MaxInstances * rp.config.Provider.AWS.SlotsPerInstance()

			for _, a := range rp.agentStatesCache {
				if blockedNodeSet.Contains(a.Handler.Address().Local()) {
					totalSlots -= len(a.slotStates)
				}
			}
		case rp.config.Provider.GCP != nil:
			totalSlots = rp.config.Provider.MaxInstances * rp.config.Provider.GCP.SlotsPerInstance()

			for _, a := range rp.agentStatesCache {
				if blockedNodeSet.Contains(a.Handler.Address().Local()) {
					totalSlots -= len(a.slotStates)
				}
			}
		default:
			panic("Invalid provider")
		}

		var capacityExceeded bool
		if totalSlots < msg.Slots {
			capacityExceeded = true
		}

		ctx.Respond(sproto.CapacityCheckResponse{
			CapacityExceeded: capacityExceeded,
			SlotsAvailable:   totalSlots,
		})

	case aproto.GetRPConfig:
		reschedule = false
		ctx.Respond(aproto.GetRPResponse{
			AgentReconnectWait:    rp.config.AgentReconnectWait,
			MaxZeroSlotContainers: rp.config.MaxAuxContainersPerAgent,
		})

	case schedulerTick:
		if rp.provisioner != nil {
			if err := rp.provisioner.LaunchError(); err != rp.provisionerError {
				rp.provisionerError = err
				if err != nil {
					rp.reschedule = true
				}
			}
		}
		if rp.reschedule {
			ctx.Log().Trace("scheduling")
			rp.agentStatesCache = rp.fetchAgentStates(ctx)
			defer func() {
				rp.agentStatesCache = nil
			}()

			rp.pruneTaskList(ctx)
			toAllocate, toRelease := rp.scheduler.Schedule(rp)
			if len(toAllocate) > 0 || len(toRelease) > 0 {
				ctx.Log().
					WithField("toAllocate", len(toAllocate)).
					WithField("toRelease", len(toRelease)).
					Debugf("scheduled")
			}
			for _, req := range toAllocate {
				rp.allocateResources(ctx, req)
			}
			for _, aID := range toRelease {
				rp.releaseResource(ctx, aID)
			}
			rp.sendScalingInfo(ctx)
		}
		rp.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	case sproto.ValidateCommandResourcesRequest:
		reschedule = false
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

func (rp *resourcePool) receiveAgentMsg(ctx *actor.Context) error {
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

func (rp *resourcePool) moveJob(
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

	if _, ok := rp.groups[jobID]; !ok {
		return sproto.ErrJobNotFound(jobID)
	}
	if _, ok := rp.queuePositions[anchorID]; !ok {
		return sproto.ErrJobNotFound(anchorID)
	}

	prioChange, secondAnchor, anchorPriority := tasklist.FindAnchor(
		jobID,
		anchorID,
		aheadOf,
		rp.taskList,
		rp.groups,
		rp.queuePositions,
		false,
	)

	if secondAnchor == "" {
		return fmt.Errorf("unable to move job with ID %s", jobID)
	}

	if secondAnchor == jobID {
		return nil
	}

	if prioChange {
		group := rp.groups[jobID]
		if group == nil {
			return fmt.Errorf("moveJob cannot find group for job %s", jobID)
		}
		oldPriority := *group.Priority
		err := rp.setGroupPriority(ctx, sproto.SetGroupPriority{
			Priority:     anchorPriority,
			ResourcePool: rp.config.PoolName,
			JobID:        jobID,
		})
		if err != nil {
			return err
		}

		if priorityChanger, ok := tasklist.GroupPriorityChangeRegistry.Load(jobID); ok {
			if priorityChanger != nil {
				if err := priorityChanger(anchorPriority); err != nil {
					_ = rp.setGroupPriority(ctx, sproto.SetGroupPriority{
						Priority:     oldPriority,
						ResourcePool: rp.config.PoolName,
						JobID:        jobID,
					})
					return err
				}
			}
		} else {
			return fmt.Errorf("unable to move job with ID %s", jobID)
		}

		if !tasklist.NeedMove(
			rp.queuePositions[jobID],
			rp.queuePositions[anchorID],
			rp.queuePositions[secondAnchor],
			aheadOf,
		) {
			return nil
		}
	}

	jobPosition, err := rp.queuePositions.SetJobPosition(jobID, anchorID, secondAnchor, aheadOf, false)
	if err != nil {
		return err
	}
	if err := rp.db.UpdateJobPosition(jobID, jobPosition); err != nil {
		return err
	}

	return nil
}

func (rp *resourcePool) receiveJobQueueMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.GetJobQStats:
		ctx.Respond(tasklist.JobStats(rp.taskList))

	case sproto.GetJobQ:
		ctx.Respond(rp.scheduler.JobQInfo(rp))

	case sproto.MoveJob:
		err := rp.moveJob(ctx, msg.ID, msg.Anchor, msg.Ahead)
		ctx.Respond(err)

	case sproto.SetGroupWeight:
		rp.getOrCreateGroup(msg.JobID).Weight = msg.Weight

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

func (rp *resourcePool) setGroupPriority(ctx *actor.Context, msg sproto.SetGroupPriority) error {
	g := rp.getOrCreateGroup(msg.JobID)
	if (g.Priority != nil && *g.Priority == msg.Priority) ||
		rp.config.Scheduler.Priority == nil {
		return nil
	}
	ctx.Log().Infof("setting priority for group of %s to %d", msg.JobID, msg.Priority)
	g.Priority = &msg.Priority
	time, err := tasklist.GetJobSubmissionTime(rp.taskList, msg.JobID)
	if err != nil {
		ctx.Log().Errorf("failed to get job submission time: %s", err)
		return nil
	}
	rp.queuePositions[msg.JobID] = tasklist.InitializeQueuePosition(time, false)
	return nil
}

func (rp *resourcePool) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.SetGroupMaxSlots:
		rp.getOrCreateGroup(msg.JobID).MaxSlots = msg.MaxSlots

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

func (rp *resourcePool) JobStopped(jobID model.JobID) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	delete(rp.groups, jobID)
	delete(rp.queuePositions, jobID)
}

func (rp *resourcePool) updateAgentStartStats(
	poolName string, agentID string, slots int,
) error {
	return rp.db.RecordAgentStats(&model.AgentStats{
		ResourcePool: poolName,
		AgentID:      agentID,
		Slots:        slots,
	})
}

func (rp *resourcePool) updateAgentEndStats(agentID string) error {
	return db.EndAgentStats(&model.AgentStats{
		AgentID: agentID,
	})
}

func (rp *resourcePool) fetchAgentStates(ctx *actor.Context) map[*actor.Ref]*agentState {
	agents := maps.Keys(rp.agents)

	responses := ctx.AskAll(getAgentState{}, agents...).GetAll()

	result := make(map[*actor.Ref]*agentState, len(rp.agents))
	for ref, msg := range responses {
		switch msg := msg.(type) {
		case *agentState:
			result[ref] = msg
		case error:
			ctx.Log().WithError(msg).Warnf("failed to get agent state for agent %s", ref.Address().Local())
		default:
			ctx.Log().Warnf("bad agent state response for agent %s", ref.Address().Local())
		}
	}

	return result
}

func (rp *resourcePool) refreshAgentStateCacheFor(ctx *actor.Context, agents []*actor.Ref) {
	responses := ctx.AskAll(getAgentState{}, agents...).GetAll()

	for ref, msg := range responses {
		switch msg := msg.(type) {
		case *agentState:
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

func (rp *resourcePool) pruneTaskList(ctx *actor.Context) {
	if rp.provisioner == nil || rp.provisionerError == nil {
		return
	}

	before := rp.taskList.Len()
	slotCount, err := rp.provisioner.CurrentSlotCount()
	if err != nil {
		return
	}

	ctx.Log().
		WithError(rp.provisionerError).
		WithField("slotCount", slotCount).
		Error("provisioner in error state")

	var allocationsToRemove []model.AllocationID
	for it := rp.taskList.Iterator(); it.Next(); {
		task := it.Value()
		if rp.taskList.IsScheduled(task.AllocationID) {
			ctx.Log().Debugf("task %s already in progress", task.AllocationID)
			continue
		}
		if task.SlotsNeeded <= slotCount {
			ctx.Log().Debugf("task %s can be scheduled with number of available slots", task.AllocationID)
			continue
		}
		ctx.Log().WithError(rp.provisionerError).Warnf("removing task %s from list", task.AllocationID)
		allocationsToRemove = append(allocationsToRemove, task.AllocationID)
	}
	for _, aID := range allocationsToRemove {
		rmevents.Publish(aID, &sproto.InvalidResourcesRequestError{Cause: rp.provisionerError})
	}
	after := rp.taskList.Len()
	ctx.Log().WithField("before", before).WithField("after", after).Warn("pruned task list")
}
