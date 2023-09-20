package kubernetesrm

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const resourcePoolEnvVar = "DET_K8S_RESOURCE_POOL"

// getResourceSummary is a message to request a summary of the resources used by the
// resource pool (agents, slots, cpu containers).
type getResourceSummary struct{}

type kubernetesResourcePool struct {
	config     *config.KubernetesResourceManagerConfig
	poolConfig *config.ResourcePoolConfig

	reqList                   *tasklist.TaskList
	groups                    map[*actor.Ref]*tasklist.Group
	allocationIDToContainerID map[model.AllocationID]cproto.ID
	containerIDtoAllocationID map[string]model.AllocationID
	// TODO(DET-9613): Jobs have many allocs.
	jobIDToAllocationID       map[model.JobID]model.AllocationID
	allocationIDToJobID       map[model.AllocationID]model.JobID
	groupActorToID            map[*actor.Ref]model.JobID
	IDToGroupActor            map[model.JobID]*actor.Ref
	slotsUsedPerGroup         map[*tasklist.Group]int
	allocationIDToRunningPods map[model.AllocationID]int

	podsActor *actor.Ref

	queuePositions tasklist.JobSortState
	reschedule     bool
}

func newResourcePool(
	rmConfig *config.KubernetesResourceManagerConfig,
	poolConfig *config.ResourcePoolConfig,
	podsActor *actor.Ref,
) *kubernetesResourcePool {
	return &kubernetesResourcePool{
		config:                    rmConfig,
		poolConfig:                poolConfig,
		reqList:                   tasklist.New(),
		groups:                    map[*actor.Ref]*tasklist.Group{},
		allocationIDToContainerID: map[model.AllocationID]cproto.ID{},
		containerIDtoAllocationID: map[string]model.AllocationID{},
		jobIDToAllocationID:       map[model.JobID]model.AllocationID{},
		allocationIDToJobID:       map[model.AllocationID]model.JobID{},
		groupActorToID:            map[*actor.Ref]model.JobID{},
		IDToGroupActor:            map[model.JobID]*actor.Ref{},
		slotsUsedPerGroup:         map[*tasklist.Group]int{},
		allocationIDToRunningPods: map[model.AllocationID]int{},
		podsActor:                 podsActor,
		queuePositions:            tasklist.InitializeJobSortState(true),
	}
}

func (k *kubernetesResourcePool) Receive(ctx *actor.Context) error {
	reschedule := true
	defer func() {
		// Default to scheduling every 500ms if a message was received, but allow messages
		// that don't affect the cluster to be skipped.
		k.reschedule = k.reschedule || reschedule
	}()

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, ActionCoolDown, SchedulerTick{})

	case
		tasklist.GroupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetAllocationName,
		sproto.AllocateRequest,
		sproto.ResourcesReleased,
		sproto.UpdatePodStatus,
		sproto.PendingPreemption:
		return k.receiveRequestMsg(ctx)

	case
		sproto.GetJobQ,
		sproto.GetJobQStats,
		sproto.SetGroupWeight,
		sproto.SetGroupPriority,
		sproto.MoveJob,
		sproto.DeleteJob,
		sproto.RecoverJobPosition,
		*apiv1.GetJobQueueStatsRequest:
		return k.receiveJobQueueMsg(ctx)

	case sproto.GetAllocationSummary:
		if resp := k.reqList.TaskSummary(
			msg.ID, k.groups, kubernetesScheduler); resp != nil {
			ctx.Respond(*resp)
		}
		reschedule = false

	case sproto.GetAllocationSummaries:
		reschedule = false
		ctx.Respond(k.reqList.TaskSummaries(k.groups, kubernetesScheduler))

	case getResourceSummary:
		slotsUsed := 0
		for _, slotsUsedByGroup := range k.slotsUsedPerGroup {
			slotsUsed += slotsUsedByGroup
		}
		pods, err := k.summarizePods(ctx)
		if err != nil {
			return err
		}

		ctx.Respond(resourceSummary{
			numAgents:              pods.NumAgents,
			numTotalSlots:          pods.SlotsAvailable,
			numActiveSlots:         slotsUsed,
			maxNumAuxContainers:    1,
			numActiveAuxContainers: 0,
			slotType:               "",
		})

	case SchedulerTick:
		if k.reschedule {
			k.schedulePendingTasks(ctx)
		}
		k.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, ActionCoolDown, SchedulerTick{})

	case sproto.ValidateCommandResourcesRequest:
		fulfillable := k.config.MaxSlotsPerPod >= msg.Slots
		ctx.Respond(sproto.ValidateCommandResourcesResponse{Fulfillable: fulfillable})

	default:
		reschedule = false
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourcePool) summarizePods(
	ctx *actor.Context,
) (*PodsInfo, error) {
	resp := ctx.Ask(k.podsActor, SummarizeResources{PoolName: k.poolConfig.PoolName})
	if err := resp.Error(); err != nil {
		return nil, err
	}
	pods, ok := resp.Get().(*PodsInfo)
	if !ok {
		return nil, actor.ErrUnexpectedMessage(ctx)
	}
	return pods, nil
}

func (k *kubernetesResourcePool) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case tasklist.GroupActorStopped:
		delete(k.slotsUsedPerGroup, k.groups[msg.Ref])
		delete(k.groups, msg.Ref)
		if jobID, ok := k.groupActorToID[msg.Ref]; ok {
			delete(k.queuePositions, jobID)
			delete(k.allocationIDToJobID, k.jobIDToAllocationID[jobID])
			delete(k.jobIDToAllocationID, jobID)
			delete(k.groupActorToID, msg.Ref)
			delete(k.IDToGroupActor, jobID)
		}

	case sproto.SetGroupMaxSlots:
		k.getOrCreateGroup(ctx, msg.Handler).MaxSlots = msg.MaxSlots

	case sproto.SetAllocationName:
		k.receiveSetAllocationName(ctx, msg)

	case sproto.AllocateRequest:
		k.addTask(ctx, msg)

	case sproto.ResourcesReleased:
		k.resourcesReleased(ctx, msg)

	case sproto.UpdatePodStatus:
		id, ok := k.containerIDtoAllocationID[msg.ContainerID]
		if !ok {
			return nil
		}

		for it := k.reqList.Iterator(); it.Next(); {
			req := it.Value()
			if req.AllocationID == id {
				req.State = msg.State
				if sproto.ScheduledStates[req.State] {
					k.allocationIDToRunningPods[id]++
				}
			}
		}

	case sproto.PendingPreemption:
		ctx.Respond(actor.ErrUnexpectedMessage(ctx))
		return nil

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (k *kubernetesResourcePool) addTask(ctx *actor.Context, msg sproto.AllocateRequest) {
	if len(msg.AllocationID) == 0 {
		msg.AllocationID = model.AllocationID(uuid.New().String())
	}
	k.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed-k8-Task"
	}

	ctx.Log().WithField("restore", msg.Restore).Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.Name, msg.AllocationID,
	)
	if msg.IsUserVisible {
		if _, ok := k.queuePositions[msg.JobID]; !ok {
			k.queuePositions[msg.JobID] = tasklist.InitializeQueuePosition(
				msg.JobSubmissionTime,
				true,
			)
		}
		k.jobIDToAllocationID[msg.JobID] = msg.AllocationID
		k.allocationIDToJobID[msg.AllocationID] = msg.JobID
		k.groupActorToID[msg.Group] = msg.JobID
		k.IDToGroupActor[msg.JobID] = msg.Group
		k.allocationIDToRunningPods[msg.AllocationID] = 0
	}
	k.reqList.AddTask(&msg)
}

func (k *kubernetesResourcePool) receiveJobQueueMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.GetJobQ:
		ctx.Respond(k.jobQInfo())

	case *apiv1.GetJobQueueStatsRequest:
		resp := &apiv1.GetJobQueueStatsResponse{
			Results: make([]*apiv1.RPQueueStat, 0),
		}
		resp.Results = append(resp.Results, &apiv1.RPQueueStat{
			Stats:        tasklist.JobStats(k.reqList),
			ResourcePool: k.poolConfig.PoolName,
		},
		)
		ctx.Respond(resp)

	case sproto.GetJobQStats:
		ctx.Respond(tasklist.JobStats(k.reqList))

	case sproto.MoveJob:
		err := k.moveJob(ctx, msg.ID, msg.Anchor, msg.Ahead)
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}

	case sproto.SetGroupWeight:
		// setting weights in kubernetes is not supported
		if ctx.ExpectingResponse() {
			ctx.Respond(rmerrors.ErrUnsupported("set group weight is unsupported in k8s"))
		}

	case sproto.SetGroupPriority:
		group := k.getOrCreateGroup(ctx, msg.Handler)
		// Check if there is already a submitted task in this group for which
		// priority is immutable. If so, respond with an error.
		for it := k.reqList.Iterator(); it.Next(); {
			if it.Value().Group == msg.Handler {
				if req := it.Value(); !req.Preemptible {
					if ctx.ExpectingResponse() {
						ctx.Respond(rmerrors.ErrUnsupported(fmt.Sprintf(
							"priority is immutable for %s in k8s because it may be destructive",
							req.Name,
						)))
					}
					return nil
				}
			}
		}

		group.Priority = &msg.Priority
		// Do the destructive thing if the group has a submitted task, since it is only allowed
		// for trials and trials take checkpoints.
		for it := k.reqList.Iterator(); it.Next(); {
			if it.Value().Group == msg.Handler {
				req := it.Value()
				if id, ok := k.allocationIDToContainerID[req.AllocationID]; ok {
					ctx.Tell(k.podsActor, ChangePriority{PodID: id})
					delete(k.allocationIDToContainerID, req.AllocationID)
				}
			}
		}

	case sproto.RecoverJobPosition:
		k.queuePositions.RecoverJobPosition(msg.JobID, msg.JobPosition)

	case sproto.DeleteJob:
		// For now, there is nothing to cleanup in k8s.
		ctx.Respond(sproto.EmptyDeleteJobResponse())

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (k *kubernetesResourcePool) moveJob(
	ctx *actor.Context,
	jobID model.JobID,
	anchorID model.JobID,
	aheadOf bool,
) error {
	for it := k.reqList.Iterator(); it.Next(); {
		if it.Value().JobID == jobID {
			if req := it.Value(); !req.Preemptible {
				ctx.Respond(fmt.Errorf(
					"move job for %s unsupported in k8s because it may be destructive",
					req.Name,
				))
				return nil
			}
		}
	}

	if anchorID == "" || jobID == "" || anchorID == jobID {
		return nil
	}

	if _, ok := k.queuePositions[jobID]; !ok {
		return nil
	}

	groupAddr, ok := k.IDToGroupActor[jobID]
	if !ok {
		return sproto.ErrJobNotFound(jobID)
	}

	if _, ok = k.queuePositions[anchorID]; !ok {
		return sproto.ErrJobNotFound(anchorID)
	}

	prioChange, secondAnchor, anchorPriority := tasklist.FindAnchor(
		jobID,
		anchorID,
		aheadOf,
		k.reqList,
		k.groups,
		k.queuePositions,
		true,
	)

	if secondAnchor == "" {
		return fmt.Errorf("unable to move job with ID %s", jobID)
	}

	if secondAnchor == jobID {
		return nil
	}

	if prioChange {
		g := k.getOrCreateGroup(ctx, k.IDToGroupActor[jobID])
		oldPriority := g.Priority
		g.Priority = &anchorPriority
		resp := ctx.Ask(k.IDToGroupActor[jobID], sproto.NotifyRMPriorityChange{
			Priority: anchorPriority,
		})
		if resp.Error() != nil {
			g.Priority = oldPriority
			return resp.Error()
		}
	}

	msg, err := k.queuePositions.SetJobPosition(jobID, anchorID, secondAnchor, aheadOf, true)
	if err != nil {
		return err
	}
	ctx.Tell(groupAddr, msg)

	allocationID, ok := k.jobIDToAllocationID[jobID]
	if !ok {
		return fmt.Errorf("job with ID %s has no valid task address", jobID)
	}
	containerID, ok := k.allocationIDToContainerID[allocationID]
	if !ok {
		return fmt.Errorf("job with ID %s has no valid containerID", jobID)
	}

	ctx.Tell(k.podsActor, ChangePosition{PodID: containerID})

	return nil
}

func (k *kubernetesResourcePool) correctJobQInfo(
	reqs []*sproto.AllocateRequest,
	q map[model.JobID]*sproto.RMJobInfo,
) map[model.JobID]*sproto.RMJobInfo {
	jobIDToAllocatedSlots := map[model.JobID]int{}
	for _, req := range reqs {
		runningPods := k.allocationIDToRunningPods[req.AllocationID]
		if req.SlotsNeeded <= k.config.MaxSlotsPerPod {
			jobIDToAllocatedSlots[req.JobID] += runningPods * req.SlotsNeeded
		} else {
			jobIDToAllocatedSlots[req.JobID] += runningPods * k.config.MaxSlotsPerPod
		}
	}

	for id := range q {
		q[id].AllocatedSlots = jobIDToAllocatedSlots[id]
	}

	return q
}

func (k *kubernetesResourcePool) jobQInfo() map[model.JobID]*sproto.RMJobInfo {
	reqs := tasklist.SortTasksWithPosition(k.reqList, k.groups, k.queuePositions, true)
	jobQInfo := tasklist.ReduceToJobQInfo(reqs)
	correctedJobQInfo := k.correctJobQInfo(reqs, jobQInfo)
	return correctedJobQInfo
}

func (k *kubernetesResourcePool) receiveSetAllocationName(
	ctx *actor.Context,
	msg sproto.SetAllocationName,
) {
	if task, found := k.reqList.TaskByID(msg.AllocationID); found {
		task.Name = msg.Name
	}
}

func (k *kubernetesResourcePool) assignResources(
	ctx *actor.Context, req *sproto.AllocateRequest,
) {
	numPods := 1
	slotsPerPod := req.SlotsNeeded
	if req.SlotsNeeded > 1 {
		if k.config.MaxSlotsPerPod == 0 {
			ctx.Log().WithField("allocation-id", req.AllocationID).Error(
				"set max_slots_per_pod > 0 to schedule tasks with slots")
			return
		}

		if req.SlotsNeeded <= k.config.MaxSlotsPerPod {
			numPods = 1
			slotsPerPod = req.SlotsNeeded
		} else {
			if req.SlotsNeeded%k.config.MaxSlotsPerPod != 0 {
				ctx.Log().WithField("allocation-id", req.AllocationID).Errorf(
					"task number of slots (%d) is not schedulable on the configured "+
						"max_slots_per_pod (%d)", req.SlotsNeeded, k.config.MaxSlotsPerPod)
				return
			}

			numPods = req.SlotsNeeded / k.config.MaxSlotsPerPod
			slotsPerPod = k.config.MaxSlotsPerPod
		}
	}

	k.slotsUsedPerGroup[k.groups[req.Group]] += req.SlotsNeeded

	var resources []*k8sPodResources
	if req.Restore {
		var err error
		resources, err = k.restoreResources(ctx, req, slotsPerPod, numPods)
		if err != nil {
			ctx.Log().
				WithField("allocation-id", req.AllocationID).
				WithError(err).Error("unable to restore allocation")
			unknownExit := sproto.ExitCode(-1)
			rmevents.Publish(req.AllocationID, &sproto.ResourcesFailure{
				FailureType: sproto.ResourcesMissing,
				ErrMsg:      errors.Wrap(err, "unable to restore allocation").Error(),
				ExitCode:    &unknownExit,
			})
			return
		}
	} else {
		resources = k.createResources(ctx, req, slotsPerPod, numPods)
	}

	allocations := sproto.ResourceList{}
	for _, rs := range resources {
		allocations[rs.Summary().ResourcesID] = rs
		k.allocationIDToContainerID[req.AllocationID] = rs.containerID
		k.containerIDtoAllocationID[rs.containerID.String()] = req.AllocationID
	}

	assigned := sproto.ResourcesAllocated{ID: req.AllocationID, Resources: allocations}
	k.reqList.AddAllocationRaw(req.AllocationID, &assigned)
	rmevents.Publish(req.AllocationID, assigned.Clone())

	if req.Restore {
		ctx.Log().
			WithField("allocation-id", req.AllocationID).
			WithField("task-handler", req.Name).
			Infof("resources restored with %d pods", numPods)
	} else {
		ctx.Log().
			WithField("allocation-id", req.AllocationID).
			WithField("task-handler", req.Name).
			Infof("resources assigned with %d pods", numPods)
	}

	if req.Restore {
		// This call must happen after we publish ResourcesAllocated, otherwise the allocation will
		// receive an update for resources it does not know about, ignore it, then hang if it missed
		// the termination.
		resp := ctx.Ask(k.podsActor, refreshPodStates{allocationID: req.AllocationID})
		if err := resp.Error(); err != nil {
			ctx.Log().WithError(err).Error("failed to refresh pod states after reattach")
		}
	}
}

func (k *kubernetesResourcePool) createResources(
	ctx *actor.Context, req *sproto.AllocateRequest, slotsPerPod, numPods int,
) []*k8sPodResources {
	var resources []*k8sPodResources
	for pod := 0; pod < numPods; pod++ {
		resources = append(resources, &k8sPodResources{
			req:             req,
			podsActor:       k.podsActor,
			containerID:     cproto.NewID(),
			slots:           slotsPerPod,
			group:           k.groups[req.Group],
			initialPosition: k.queuePositions[k.allocationIDToJobID[req.AllocationID]],
			namespace:       k.poolConfig.KubernetesNamespace,
		})
	}
	return resources
}

func (k *kubernetesResourcePool) restoreResources(
	ctx *actor.Context, req *sproto.AllocateRequest, slotsPerPod, numPods int,
) ([]*k8sPodResources, error) {
	resp := ctx.Ask(k.podsActor, reattachAllocationPods{
		allocationID: req.AllocationID,
		numPods:      numPods,
		slots:        slotsPerPod,
		logContext:   req.LogContext,
	})
	if err := resp.Error(); err != nil {
		return nil, err
	}
	restoreResponses := resp.Get().([]reattachPodResponse)

	var resources []*k8sPodResources
	for _, restoreResponse := range restoreResponses {
		resources = append(resources, &k8sPodResources{
			req:             req,
			podsActor:       k.podsActor,
			containerID:     cproto.ID(restoreResponse.containerID),
			slots:           slotsPerPod,
			group:           k.groups[req.Group],
			initialPosition: k.queuePositions[k.allocationIDToJobID[req.AllocationID]],
			namespace:       k.poolConfig.KubernetesNamespace,

			started: restoreResponse.started,
		})
	}

	return resources, nil
}

func (k *kubernetesResourcePool) resourcesReleased(
	ctx *actor.Context,
	msg sproto.ResourcesReleased,
) {
	req, ok := k.reqList.TaskByID(msg.AllocationID)
	if !ok {
		ctx.Log().Debugf("ignoring release for task not allocated to pool %s", msg.AllocationID)
		return
	}

	if msg.ResourcesID != nil {
		// Just ignore this minor optimization in Kubernetes.
		return
	}

	ctx.Log().Infof("resources are released for %s", msg.AllocationID)

	group := k.groups[req.Group]
	if group != nil {
		k.slotsUsedPerGroup[group] -= req.SlotsNeeded
	}

	k.reqList.RemoveTaskByID(msg.AllocationID)
	delete(k.allocationIDToContainerID, msg.AllocationID)
	delete(k.allocationIDToRunningPods, msg.AllocationID)

	for id, addr := range k.containerIDtoAllocationID {
		if addr == msg.AllocationID {
			delete(k.containerIDtoAllocationID, id)
			break
		}
	}
	rmevents.Publish(msg.AllocationID, sproto.ResourcesReleasedEvent{})
}

func (k *kubernetesResourcePool) getOrCreateGroup(
	ctx *actor.Context,
	handler *actor.Ref,
) *tasklist.Group {
	if g, ok := k.groups[handler]; ok {
		return g
	}
	priority := config.KubernetesDefaultPriority
	g := &tasklist.Group{Handler: handler, Weight: 1, Priority: &priority}

	k.groups[handler] = g
	k.slotsUsedPerGroup[g] = 0

	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, tasklist.GroupActorStopped{Ref: handler})
	}
	return g
}

func (k *kubernetesResourcePool) schedulePendingTasks(ctx *actor.Context) {
	for it := k.reqList.Iterator(); it.Next(); {
		req := it.Value()
		group := k.groups[req.Group]
		if !k.reqList.IsScheduled(req.AllocationID) {
			if maxSlots := group.MaxSlots; maxSlots != nil {
				if k.slotsUsedPerGroup[group]+req.SlotsNeeded > *maxSlots {
					continue
				}
			}

			k.assignResources(ctx, req)
		}
	}
}

type k8sPodResources struct {
	req             *sproto.AllocateRequest
	podsActor       *actor.Ref
	group           *tasklist.Group
	containerID     cproto.ID
	slots           int
	initialPosition decimal.Decimal
	namespace       string

	started *sproto.ResourcesStarted
}

// Summary summarizes a container allocation.
func (p k8sPodResources) Summary() sproto.ResourcesSummary {
	return sproto.ResourcesSummary{
		AllocationID:  p.req.AllocationID,
		ResourcesID:   sproto.ResourcesID(p.containerID),
		ResourcesType: sproto.ResourcesTypeK8sPod,
		AgentDevices: map[aproto.ID][]device.Device{
			// TODO: Make it more obvious k8s can't be trusted.
			aproto.ID(p.podsActor.Address().Local()): make([]device.Device, p.slots),
		},

		ContainerID: &p.containerID,
		Started:     p.started,
	}
}

// Start notifies the pods actor that it should launch a pod for the provided task spec.
func (p k8sPodResources) Start(
	ctx *actor.System, logCtx logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
) error {
	p.setPosition(&spec)
	spec.ContainerID = string(p.containerID)
	spec.ResourcesID = string(p.containerID)
	spec.AllocationID = string(p.req.AllocationID)
	spec.AllocationSessionToken = rri.Token
	spec.TaskID = string(p.req.TaskID)
	spec.UseHostMode = rri.IsMultiAgent
	spec.ResourcesConfig.SetPriority(p.group.Priority)
	if spec.LoggingFields == nil {
		spec.LoggingFields = map[string]string{}
	}
	spec.LoggingFields["allocation_id"] = spec.AllocationID
	spec.LoggingFields["task_id"] = spec.TaskID
	spec.ExtraEnvVars[sproto.ResourcesTypeEnvVar] = string(sproto.ResourcesTypeK8sPod)
	spec.ExtraEnvVars[resourcePoolEnvVar] = p.req.ResourcePool
	return ctx.Ask(p.podsActor, StartTaskPod{
		AllocationID: p.req.AllocationID,
		Spec:         spec,
		Slots:        p.slots,
		Rank:         rri.AgentRank,
		Namespace:    p.namespace,
		LogContext:   logCtx,
	}).Error()
}

func (p k8sPodResources) setPosition(spec *tasks.TaskSpec) {
	newSpec := spec.Environment.PodSpec()
	if newSpec == nil {
		newSpec = &expconf.PodSpec{}
	}
	if newSpec.Labels == nil {
		newSpec.Labels = make(map[string]string)
	}
	newSpec.Labels["determined-queue-position"] = p.initialPosition.String()
	spec.Environment.SetPodSpec(newSpec)
}

// Kill notifies the pods actor that it should stop the pod.
func (p k8sPodResources) Kill(ctx *actor.System, _ logger.Context) {
	ctx.Tell(p.podsActor, KillTaskPod{
		PodID: p.containerID,
	})
}

func (p k8sPodResources) Persist() error {
	return nil
}

// resourceSummary is a summary of the resource available/used by a resource pool.
type resourceSummary struct {
	numAgents              int
	numTotalSlots          int
	numActiveSlots         int
	maxNumAuxContainers    int
	numActiveAuxContainers int
	slotType               device.Type
}
