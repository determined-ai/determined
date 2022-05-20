package resourcemanagers

import (
	"context"
	"crypto/tls"
	"fmt"

	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/resourcemanagers/agent"
	"github.com/determined-ai/determined/master/internal/resourcemanagers/provisioner"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
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
	rp.notifyOnStop(ctx, msg.TaskActor, sproto.ResourcesReleased{TaskActor: msg.TaskActor})
	log := ctx.Log().WithField("allocation-id", msg.AllocationID)

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

	log.Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.TaskActor.Address(), msg.AllocationID,
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
			if err != nil {
				log.WithError(err).Error("error restoring resources")
			} else {
				log.Info("failed to restore resources")
			}

			// Clear out the state / close and terminate the allocation.
			errMsg := "failed to restore"
			if err != nil {
				errMsg = err.Error()
			}

			rf := sproto.RestoreResourcesFailure{
				FailureType: sproto.RestoreError,
				ErrMsg:      errMsg,
				ExitCode:    nil,
			}
			ctx.Tell(msg.TaskActor, rf)

			return
		}
	}

	rp.taskList.AddTask(&msg)
}

func (rp *ResourcePool) restoreResources(
	ctx *actor.Context, req *sproto.AllocateRequest) error {
	rp.agentStatesCache = rp.fetchAgentStates(ctx)
	defer func() {
		rp.agentStatesCache = nil
	}()

	allocationID := req.AllocationID

	subq := db.Bun().NewSelect().Model((*task.ResourcesWithState)(nil)).
		Where("allocation_id = ?", allocationID).
		Column("resource_id")

	containerSnapshots := []agent.ContainerSnapshot{}
	err := db.Bun().NewSelect().Model(&containerSnapshots).
		Where("resource_id in (?)", subq).
		Scan(context.TODO())
	if err != nil {
		return err
	}

	if len(containerSnapshots) == 0 {
		return errors.New("0 container snapshots")
	}

	resources := make([]sproto.Resources, 0, len(containerSnapshots))

	agentStateMap := map[aproto.ID]*agent.AgentState{}

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
		}
		resources = append(resources, &cr)
	}

	allocated := sproto.ResourcesAllocated{
		ID:           req.AllocationID,
		ResourcePool: rp.config.PoolName,
		Resources:    resources,
		Recovered:    true,
	}

	rp.taskList.AddTask(req)
	rp.taskList.SetAllocations(req.TaskActor, &allocated)
	ctx.Tell(req.TaskActor, allocated)

	return nil
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

	resources := make([]*containerResources, 0, len(fits))
	rollback := false

	defer func() {
		if rollback {
			// Rollback previous allocations.
			for _, resource := range resources {
				ctx.Tell(resource.agent.Handler,
					agent.DeallocateContainer{ContainerID: resource.containerID})
			}
		}
	}()

	for _, fit := range fits {
		containerID := cproto.NewID()
		rr := ctx.Ask(fit.Agent.Handler, agent.AllocateFreeDevices{
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
		case agent.AllocateFreeDevicesResponse:
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
		rs := task.NewResourcesState(cr, -1)
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

	sprotoResources := make([]sproto.Resources, len(resources))
	for i, v := range resources {
		sprotoResources[i] = v
	}

	allocated := sproto.ResourcesAllocated{
		ID:                req.AllocationID,
		ResourcePool:      rp.config.PoolName,
		Resources:         sprotoResources,
		JobSubmissionTime: req.JobSubmissionTime,
	}
	rp.taskList.SetAllocations(req.TaskActor, &allocated)
	ctx.Tell(req.TaskActor, allocated)

	// Refresh state for the updated agents.
	allocatedAgents := make([]*actor.Ref, 0, len(resources))
	for _, allocation := range resources {
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
		for _, allocation := range allocated.Resources {
			typed := allocation.(*containerResources)
			ctx.Tell(typed.agent.Handler, agent.DeallocateContainer{ContainerID: typed.containerID})
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
		job.RecoverJobPosition,
		job.DeleteJob:
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
	if rp.config.Scheduler.GetType() != config.PriorityScheduling {
		return fmt.Errorf("unable to perform operation on resource pool with %s",
			rp.config.Scheduler.GetType())
	}
	if anchorID == "" || jobID == "" || anchorID == jobID {
		return nil
	}

	if _, ok := rp.queuePositions[jobID]; !ok {
		return nil
	}

	groupAddr, ok := rp.IDToGroupActor[jobID]
	if !ok {
		return job.ErrJobNotFound(jobID)
	}
	if _, ok := rp.queuePositions[anchorID]; !ok {
		return job.ErrJobNotFound(anchorID)
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
		err := rp.setGroupPriority(ctx, job.SetGroupPriority{
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
			_ = rp.setGroupPriority(ctx, job.SetGroupPriority{
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
	case job.GetJobQStats:
		ctx.Respond(jobStats(rp.taskList))

	case job.GetJobQ:
		ctx.Respond(rp.scheduler.JobQInfo(rp))

	case job.MoveJob:
		if rp.config.Scheduler.GetType() != config.PriorityScheduling {
			return fmt.Errorf("unable to perform operation on resource pool with %s",
				rp.config.Scheduler.GetType())
		}
		err := rp.moveJob(ctx, msg.ID, msg.Anchor, msg.Ahead)
		ctx.Respond(err)

	case job.SetGroupWeight:
		rp.getOrCreateGroup(ctx, msg.Handler).weight = msg.Weight

	case job.SetGroupPriority:
		err := rp.setGroupPriority(ctx, msg)
		ctx.Respond(err)

	case job.RecoverJobPosition:
		rp.queuePositions.RecoverJobPosition(msg.JobID, msg.JobPosition)

	case job.DeleteJob:
		// For now, there is nothing to cleanup in determined-agents world.
		ctx.Respond(job.EmptyDeleteJobResponse())

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) setGroupPriority(ctx *actor.Context, msg job.SetGroupPriority) error {
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

	case sproto.SetTaskName:
		rp.receiveSetTaskName(ctx, msg)

	case sproto.AllocateRequest:
		rp.allocateRequest(ctx, msg)

	case sproto.ResourcesReleased:
		rp.resourcesReleased(ctx, msg.TaskActor)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (rp *ResourcePool) updateAgentStartStats(
	poolName string, agentID string, slots int) error {
	return rp.db.RecordAgentStats(&model.AgentStats{
		ResourcePool: poolName,
		AgentID:      agentID,
		Slots:        slots,
	})
}

func (rp *ResourcePool) updateAgentEndStats(agentID string) error {
	return rp.db.EndAgentStats(&model.AgentStats{
		AgentID: agentID,
	})
}

func (rp *ResourcePool) fetchAgentStates(ctx *actor.Context) map[*actor.Ref]*agent.AgentState {
	agents := maps.Keys(rp.agents)

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

// containerResources contains information for tasks have been allocated but not yet started.
type containerResources struct {
	req         *sproto.AllocateRequest
	agent       *agent.AgentState
	devices     []device.Device
	containerID cproto.ID
}

// Summary summarizes a container allocation.
func (c containerResources) Summary() sproto.ResourcesSummary {
	return sproto.ResourcesSummary{
		ResourcesID:   sproto.ResourcesID(c.containerID),
		ResourcesType: sproto.ResourcesTypeDockerContainer,
		AllocationID:  c.req.AllocationID,
		AgentDevices: map[aproto.ID][]device.Device{
			aproto.ID(c.agent.Handler.Address().Local()): c.devices},

		ContainerID: &c.containerID,
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
	return ctx.Ask(handler, sproto.StartTaskContainer{
		TaskActor: c.req.TaskActor,
		StartContainer: aproto.StartContainer{
			Container: cproto.Container{
				Parent:  c.req.TaskActor.Address(),
				ID:      c.containerID,
				State:   cproto.Assigned,
				Devices: c.devices,
			},
			Spec: spec.ToDockerSpec(),
		},
		LogContext: logCtx,
	}).Error()
}

// KillContainer notifies the agent to kill the container.
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

	snapshot := agent.ContainerSnapshot{
		ResourceID: summary.ResourcesID,
		ID:         c.containerID,
		AgentID:    agentID,
	}
	_, err := db.Bun().NewInsert().Model(&snapshot).Exec(context.TODO())
	return err
}
