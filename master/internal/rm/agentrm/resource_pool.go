package agentrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/rm/agentrm/provisioner"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// resourcePool manages the agent and task lifecycles.
type resourcePool struct {
	syslog *logrus.Entry
	mu     sync.Mutex

	config *config.ResourcePoolConfig
	cert   *tls.Certificate

	scheduler        Scheduler
	fittingMethod    SoftConstraint
	slotsPerInstance int

	provisioner      *provisioner.Provisioner
	provisionerError error

	agentService     *agents
	agentStatesCache map[agentID]*agentState
	taskList         *tasklist.TaskList
	groups           map[model.JobID]*tasklist.Group
	queuePositions   tasklist.JobSortState // secondary sort key based on job submission time
	scalingInfo      *sproto.ScalingInfo

	reschedule      bool
	rescheduleTimer *time.Timer

	// Track notifyOnStop for testing purposes.
	saveNotifications bool
	notifications     []<-chan struct{}

	db db.DB
}

// actionCoolDown is the rate limit for scheduler action.
const actionCoolDown = 500 * time.Millisecond

// newResourcePool initializes a new empty default resource provider.
func newResourcePool(
	config *config.ResourcePoolConfig,
	db db.DB,
	cert *tls.Certificate,
	scheduler Scheduler,
	fittingMethod SoftConstraint,
	agentService *agents,
) (*resourcePool, error) {
	rp := &resourcePool{
		syslog: logrus.WithField("component", "resource-pool").WithField("name", config.PoolName),

		config: config,
		cert:   cert,

		scheduler:     scheduler,
		fittingMethod: fittingMethod,

		agentService:   agentService,
		taskList:       tasklist.New(),
		groups:         make(map[model.JobID]*tasklist.Group),
		queuePositions: tasklist.InitializeJobSortState(false),
		scalingInfo:    &sproto.ScalingInfo{},

		reschedule: false,
		db:         db,
	}

	rp.mu.Lock()
	defer rp.mu.Unlock()

	err := rp.setupProvisioner()
	if err != nil {
		return nil, err
	}

	rp.rescheduleTimer = time.AfterFunc(actionCoolDown, rp.schedulerTick)
	return rp, nil
}

func (rp *resourcePool) setupProvisioner() error {
	if rp.config.Provider == nil {
		rp.syslog.Infof("not enabling provisioner for resource pool: %s", rp.config.PoolName)
		return nil
	}
	p, err := provisioner.Setup(rp.config.Provider, rp.config.PoolName, rp.cert, rp.db)
	if err != nil {
		return errors.Wrapf(err, "cannot create resource pool: %s", rp.config.PoolName)
	}
	rp.slotsPerInstance = p.SlotsPerInstance()
	rp.provisioner = p
	return nil
}

func (rp *resourcePool) Allocate(msg sproto.AllocateRequest) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true

	rp.allocateRequest(msg)
}

func (rp *resourcePool) allocateRequest(msg sproto.AllocateRequest) {
	log := rp.syslog.
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
		err := rp.restoreResources(&msg)
		if err != nil {
			log.WithError(err).Error("error restoring resources")

			// Clear out the state / close and terminate the allocation.
			rmevents.Publish(msg.AllocationID, &sproto.ResourcesRestoreError{
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
	req *sproto.AllocateRequest,
) error {
	rp.agentStatesCache = rp.agentService.list(rp.config.PoolName)
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

	for agentRef, state := range rp.agentStatesCache {
		agentStateMap[aproto.ID(state.agentID())] = rp.agentStatesCache[agentRef]
	}

	for _, cs := range containerSnapshots {
		agentState, ok := agentStateMap[cs.AgentID]
		if !ok {
			return fmt.Errorf("can't find restorable agent %s", cs.AgentID)
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
	rp.taskList.AddAllocation(req.AllocationID, &allocated)
	rmevents.Publish(req.AllocationID, allocated.Clone())

	return nil
}

func (rp *resourcePool) ResourcesReleased(msg sproto.ResourcesReleased) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true

	rp.resourcesReleased(msg)
}

func (rp *resourcePool) resourcesReleased(msg sproto.ResourcesReleased) {
	_, ok := rp.taskList.TaskByID(msg.AllocationID)
	if !ok {
		rp.syslog.Debugf("ignoring release for task not allocated to pool %s", msg.AllocationID)
		return
	}

	switch allocated := rp.taskList.Allocation(msg.AllocationID); {
	case allocated == nil:
		rp.syslog.Infof("released before allocated for %s", msg.AllocationID)
		rp.taskList.RemoveTaskByID(msg.AllocationID)
		rmevents.Publish(msg.AllocationID, sproto.ResourcesReleasedEvent{})
	case msg.ResourcesID != nil:
		rp.syslog.Infof("incrementally released resources %v for %s", *msg.ResourcesID, msg.AllocationID)
		for rID, r := range allocated.Resources {
			if r.Summary().ResourcesID != *msg.ResourcesID {
				continue
			}

			typed := r.(*containerResources)
			err := typed.agent.handler.DeallocateContainer(deallocateContainer{containerID: typed.containerID})
			if err != nil {
				rp.syslog.WithError(err).Errorf(
					"failed to deallocate container %s on agent %s",
					typed.containerID, typed.agent.id,
				)
			}
			delete(allocated.Resources, rID)
			break
		}
	default:
		rp.syslog.Infof("all resources are released for %s", msg.AllocationID)
		for _, r := range allocated.Resources {
			typed := r.(*containerResources)
			err := typed.agent.handler.DeallocateContainer(deallocateContainer{containerID: typed.containerID})
			if err != nil {
				rp.syslog.WithError(err).Errorf(
					"failed to deallocate container %s on agent %s",
					typed.containerID, typed.agent.id,
				)
			}
		}
		rp.taskList.RemoveTaskByID(msg.AllocationID)
		rmevents.Publish(msg.AllocationID, sproto.ResourcesReleasedEvent{})
	}
}

func (rp *resourcePool) SetGroupWeight(msg sproto.SetGroupWeight) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true

	rp.getOrCreateGroup(msg.JobID).Weight = msg.Weight
}

func (rp *resourcePool) SetGroupMaxSlots(msg sproto.SetGroupMaxSlots) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true

	rp.getOrCreateGroup(msg.JobID).MaxSlots = msg.MaxSlots
}

func (rp *resourcePool) SetGroupPriority(msg sproto.SetGroupPriority) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true

	return rp.setGroupPriority(msg)
}

func (rp *resourcePool) setGroupPriority(msg sproto.SetGroupPriority) error {
	g := rp.getOrCreateGroup(msg.JobID)
	if (g.Priority != nil && *g.Priority == msg.Priority) ||
		rp.config.Scheduler.Priority == nil {
		return nil
	}
	rp.syslog.Infof("setting priority for group of %s to %d", msg.JobID, msg.Priority)
	g.Priority = &msg.Priority
	time, err := tasklist.GetJobSubmissionTime(rp.taskList, msg.JobID)
	if err != nil {
		rp.syslog.Errorf("failed to get job submission time: %s", err)
		return nil
	}
	rp.queuePositions[msg.JobID] = tasklist.InitializeQueuePosition(time, false)
	return nil
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

func (rp *resourcePool) schedulerTick() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.provisioner != nil {
		if err := rp.provisioner.LaunchError(); err != rp.provisionerError {
			rp.provisionerError = err
			if err != nil {
				rp.reschedule = true
			}
		}
	}
	if rp.reschedule {
		rp.syslog.Trace("scheduling")
		rp.agentStatesCache = rp.agentService.list(rp.config.PoolName)
		defer func() {
			rp.agentStatesCache = nil
		}()

		rp.pruneTaskList()
		toAllocate, toRelease := rp.scheduler.Schedule(rp)
		if len(toAllocate) > 0 || len(toRelease) > 0 {
			rp.syslog.
				WithField("toAllocate", len(toAllocate)).
				WithField("toRelease", len(toRelease)).
				Debugf("scheduled")
		}
		for _, req := range toAllocate {
			rp.allocateResources(req)
		}
		for _, aID := range toRelease {
			rp.releaseResource(aID)
		}
		rp.sendScalingInfo()
	}
	rp.reschedule = false
	rp.rescheduleTimer = time.AfterFunc(actionCoolDown, rp.schedulerTick)
}

// allocateResources assigns resources based on a request and notifies the request
// handler of the assignment. It returns true if it is successfully allocated.
func (rp *resourcePool) allocateResources(req *sproto.AllocateRequest) bool {
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
			for _, r := range resources {
				go func(resource *containerResources) {
					err := resource.agent.handler.DeallocateContainer(deallocateContainer{containerID: resource.containerID})
					if err != nil {
						rp.syslog.WithError(err).Errorf(
							"failed to deallocate container %s on agent %s when rolling back assignments",
							resource.containerID, resource.agent.id,
						)
					}
				}(r)
			}
		}
	}()

	for _, fit := range fits {
		containerID := cproto.NewID()
		resp, err := fit.Agent.handler.AllocateFreeDevices(allocateFreeDevices{
			slots:       fit.Slots,
			containerID: containerID,
		})
		if err != nil {
			// Rollback previous allocations.
			rp.syslog.WithError(err).Warnf("failed to allocate request %s", req.AllocationID)
			rollback = true
			return false
		}

		resources = append(resources, &containerResources{
			req:         req,
			agent:       fit.Agent,
			containerID: containerID,
			devices:     resp.devices,
		})
	}

	// persist allocation_resources and container_resources.
	for _, cr := range resources {
		rs := taskmodel.NewResourcesState(cr, -1)
		if err := rs.Persist(); err != nil {
			rp.syslog.WithError(err).Error("persistence failure")
			rollback = true
			return false
		}
		if err := cr.persist(); err != nil {
			rp.syslog.WithError(err).Error("persistence failure")
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
	allocatedAgents := make([]*agent, 0, len(resources))
	for _, allocation := range resources {
		allocatedAgents = append(allocatedAgents, allocation.agent.handler)
	}

	rp.refreshAgentStateCacheFor(allocatedAgents)

	rp.syslog.Infof("allocated resources to %s", req.Name)

	return true
}

func (rp *resourcePool) releaseResource(aID model.AllocationID) {
	rp.syslog.Infof("releasing resources taken by %s (preempted by the scheduler)", aID)
	rmevents.Publish(aID, &sproto.ReleaseResources{Reason: "preempted by the scheduler"})
}

func (rp *resourcePool) sendScalingInfo() {
	if rp.provisioner != nil && rp.updateScalingInfo() {
		rp.provisioner.UpdateScalingInfo(rp.scalingInfo)
	}
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

func (rp *resourcePool) refreshAgentStateCacheFor(agents []*agent) {
	for _, a := range agents {
		state, err := a.State()
		if err != nil {
			rp.syslog.WithError(err).Warnf("failed to get agent state for agent %s", a.id)
			delete(rp.agentStatesCache, state.id)
			continue
		}
		rp.agentStatesCache[a.id] = state
	}
}

func (rp *resourcePool) pruneTaskList() {
	if rp.provisioner == nil || rp.provisionerError == nil {
		return
	}

	before := rp.taskList.Len()
	slotCount, err := rp.provisioner.CurrentSlotCount()
	if err != nil {
		return
	}

	rp.syslog.
		WithError(rp.provisionerError).
		WithField("slotCount", slotCount).
		Error("provisioner in error state")

	var allocationsToRemove []model.AllocationID
	for it := rp.taskList.Iterator(); it.Next(); {
		task := it.Value()
		if rp.taskList.IsScheduled(task.AllocationID) {
			rp.syslog.Debugf("task %s already in progress", task.AllocationID)
			continue
		}
		if task.SlotsNeeded <= slotCount {
			rp.syslog.Debugf("task %s can be scheduled with number of available slots", task.AllocationID)
			continue
		}
		rp.syslog.WithError(rp.provisionerError).Warnf("removing task %s from list", task.AllocationID)
		allocationsToRemove = append(allocationsToRemove, task.AllocationID)
	}
	for _, aID := range allocationsToRemove {
		rmevents.Publish(aID, &sproto.InvalidResourcesRequestError{Cause: rp.provisionerError})
	}
	after := rp.taskList.Len()
	rp.syslog.WithField("before", before).WithField("after", after).Warn("pruned task list")
}

func (rp *resourcePool) NotifyAgentUpdated() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true
}

func (rp *resourcePool) GetAllocationSummaries(
	msg sproto.GetAllocationSummaries,
) map[model.AllocationID]sproto.AllocationSummary {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	return rp.taskList.TaskSummaries(rp.groups, rp.config.Scheduler.GetType())
}

func (rp *resourcePool) ValidateResources(
	msg sproto.ValidateResourcesRequest,
) sproto.ValidateResourcesResponse {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	var fulfillable bool

	if rp.slotsPerInstance > 0 {
		fulfillable = rp.slotsPerInstance >= msg.Slots
	} else {
		rp.agentStatesCache = rp.agentService.list(rp.config.PoolName)
		defer func() {
			rp.agentStatesCache = nil
		}()

		maxSlots := 0
		for _, a := range rp.agentStatesCache {
			maxSlots = max(maxSlots, len(a.slotStates))
		}

		fulfillable = maxSlots >= msg.Slots
	}

	return sproto.ValidateResourcesResponse{Fulfillable: fulfillable}
}

// GetResourceSummary requests a summary of the resources used by the resource pool (agents, slots, cpu containers).
func (rp *resourcePool) GetResourceSummary() resourceSummary {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.agentStatesCache = rp.agentService.list(rp.config.PoolName)
	defer func() {
		rp.agentStatesCache = nil
	}()
	return resourceSummaryFromAgentStates(rp.agentStatesCache)
}

func (rp *resourcePool) CapacityCheck(msg sproto.CapacityCheck) (sproto.CapacityCheckResponse, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	var totalSlots int
	blockedNodeSet := set.New[string]()
	if msg.TaskID != nil {
		blockedNodes, err := logpattern.GetBlockedNodes(context.TODO(), *msg.TaskID)
		if err != nil {
			return sproto.CapacityCheckResponse{}, err
		}
		blockedNodeSet = set.FromSlice(blockedNodes)
	}
	rp.agentStatesCache = rp.agentService.list(rp.config.PoolName)
	defer func() {
		rp.agentStatesCache = nil
	}()

	switch {
	case rp.config.Provider == nil:
		for id, a := range rp.agentStatesCache {
			if !blockedNodeSet.Contains(string(id)) {
				totalSlots += len(a.slotStates)
			}
		}
	case rp.config.Provider.AWS != nil:
		totalSlots = rp.config.Provider.MaxInstances * rp.config.Provider.AWS.SlotsPerInstance()

		for id, a := range rp.agentStatesCache {
			if blockedNodeSet.Contains(string(id)) {
				totalSlots -= len(a.slotStates)
			}
		}
	case rp.config.Provider.GCP != nil:
		totalSlots = rp.config.Provider.MaxInstances * rp.config.Provider.GCP.SlotsPerInstance()

		for id, a := range rp.agentStatesCache {
			if blockedNodeSet.Contains(string(id)) {
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

	return sproto.CapacityCheckResponse{
		CapacityExceeded: capacityExceeded,
		SlotsAvailable:   totalSlots,
	}, nil
}

func (rp *resourcePool) MoveJob(msg sproto.MoveJob) error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true

	return rp.moveJob(msg.ID, msg.Anchor, msg.Ahead)
}

func (rp *resourcePool) moveJob(
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
		err := rp.setGroupPriority(sproto.SetGroupPriority{
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
					_ = rp.setGroupPriority(sproto.SetGroupPriority{
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

func (rp *resourcePool) RecoverJobPosition(msg sproto.RecoverJobPosition) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true

	rp.queuePositions.RecoverJobPosition(msg.JobID, msg.JobPosition)
}

func (rp *resourcePool) GetJobQStats(msg sproto.GetJobQStats) *jobv1.QueueStats {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	return tasklist.JobStats(rp.taskList)
}

func (rp *resourcePool) GetJobQ(msg sproto.GetJobQ) map[model.JobID]*sproto.RMJobInfo {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	return rp.scheduler.JobQInfo(rp)
}

func (rp *resourcePool) JobStopped(jobID model.JobID) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.reschedule = true

	delete(rp.groups, jobID)
	delete(rp.queuePositions, jobID)
}

// only for tests, for now.
func (rp *resourcePool) stop() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.rescheduleTimer.Stop()
}
