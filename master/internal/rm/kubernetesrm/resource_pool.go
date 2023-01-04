package kubernetesrm

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

type kubernetesResourcePool struct {
	config *config.KubernetesResourceManagerConfig

	reqList           *tasklist.TaskList
	groups            map[*actor.Ref]*tasklist.Group
	addrToContainerID map[*actor.Ref]cproto.ID
	containerIDtoAddr map[string]*actor.Ref
	jobIDtoAddr       map[model.JobID]*actor.Ref
	addrToJobID       map[*actor.Ref]model.JobID
	groupActorToID    map[*actor.Ref]model.JobID
	IDToGroupActor    map[model.JobID]*actor.Ref
	slotsUsedPerGroup map[*tasklist.Group]int

	podsActor *actor.Ref

	queuePositions tasklist.JobSortState
	reschedule     bool
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

	case sproto.GetAllocationHandler:
		reschedule = false
		ctx.Respond(k.reqList.TaskHandler(msg.ID))

	case sproto.GetAllocationSummary:
		if resp := k.reqList.TaskSummary(
			msg.ID, k.groups, kubernetesScheduler); resp != nil {
			ctx.Respond(*resp)
		}
		reschedule = false

	case sproto.GetAllocationSummaries:
		reschedule = false
		ctx.Respond(k.reqList.TaskSummaries(k.groups, kubernetesScheduler))

	case SchedulerTick:
		if k.reschedule {
			k.schedulePendingTasks(ctx)
		}
		k.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, ActionCoolDown, SchedulerTick{})

	case *apiv1.GetResourcePoolsRequest:
		if summary, err := k.summarizeResourcePool(ctx); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(summary)
		}

	default:
		reschedule = false
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourcePool) summarizeResourcePool(
	ctx *actor.Context,
) (*resourcepoolv1.ResourcePool, error) {
	slotsUsed := 0
	for _, slotsUsedByGroup := range k.slotsUsedPerGroup {
		slotsUsed += slotsUsedByGroup
	}

	pods, err := k.summarizePods(ctx)
	if err != nil {
		return nil, err
	}

	// Expose a fake number of zero slots here just to signal to the UI
	// that this RP does support the aux containers.

	return &resourcepoolv1.ResourcePool{
		Name:                         KubernetesDummyResourcePool,
		Description:                  "Kubernetes-managed pool of resources",
		Type:                         resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_K8S,
		NumAgents:                    int32(pods.NumAgents),
		SlotType:                     k.config.SlotType.Proto(),
		SlotsAvailable:               int32(pods.SlotsAvailable),
		SlotsUsed:                    int32(slotsUsed),
		AuxContainerCapacity:         int32(1),
		AuxContainersRunning:         int32(0),
		DefaultComputePool:           true,
		DefaultAuxPool:               true,
		Preemptible:                  k.config.GetPreemption(),
		MinAgents:                    0,
		MaxAgents:                    0,
		SlotsPerAgent:                int32(k.config.MaxSlotsPerPod),
		AuxContainerCapacityPerAgent: int32(1),
		SchedulerType:                resourcepoolv1.SchedulerType_SCHEDULER_TYPE_KUBERNETES,
		SchedulerFittingPolicy:       resourcepoolv1.FittingPolicy_FITTING_POLICY_KUBERNETES,
		Location:                     "kubernetes",
		ImageId:                      "",
		InstanceType:                 "kubernetes",
		Details:                      &resourcepoolv1.ResourcePoolDetail{},
	}, nil
}

func (k *kubernetesResourcePool) summarizePods(
	ctx *actor.Context,
) (*PodsInfo, error) {
	resp := ctx.Ask(k.podsActor, SummarizeResources{})
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
			delete(k.addrToJobID, k.jobIDtoAddr[jobID])
			delete(k.jobIDtoAddr, jobID)
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
		var ref *actor.Ref
		if addr, ok := k.containerIDtoAddr[msg.ContainerID]; ok {
			ref = addr
		}

		for it := k.reqList.Iterator(); it.Next(); {
			req := it.Value()
			if req.AllocationRef == ref {
				req.State = msg.State
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
	actors.NotifyOnStop(ctx, msg.AllocationRef, sproto.ResourcesReleased{
		AllocationRef: msg.AllocationRef,
	})

	if len(msg.AllocationID) == 0 {
		msg.AllocationID = model.AllocationID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.AllocationRef
	}
	k.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed-k8-Task"
	}

	ctx.Log().Infof(
		"resources are requested by %s (Allocation ID: %s)",
		msg.AllocationRef.Address(), msg.AllocationID,
	)
	if msg.IsUserVisible {
		if _, ok := k.queuePositions[msg.JobID]; !ok {
			k.queuePositions[msg.JobID] = tasklist.InitializeQueuePosition(
				msg.JobSubmissionTime,
				true,
			)
		}
		k.jobIDtoAddr[msg.JobID] = msg.AllocationRef
		k.addrToJobID[msg.AllocationRef] = msg.JobID
		k.groupActorToID[msg.Group] = msg.JobID
		k.IDToGroupActor[msg.JobID] = msg.Group
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
			ResourcePool: KubernetesDummyResourcePool,
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
				taskActor := it.Value().AllocationRef
				if id, ok := k.addrToContainerID[taskActor]; ok {
					ctx.Tell(k.podsActor, ChangePriority{PodID: id})
					delete(k.addrToContainerID, taskActor)
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

	addr, ok := k.jobIDtoAddr[jobID]
	if !ok {
		return fmt.Errorf("job with ID %s has no valid task address", jobID)
	}
	containerID, ok := k.addrToContainerID[addr]
	if !ok {
		return fmt.Errorf("job with ID %s has no valid containerID", jobID)
	}

	ctx.Tell(k.podsActor, ChangePosition{PodID: containerID})

	return nil
}

func (k *kubernetesResourcePool) jobQInfo() map[model.JobID]*sproto.RMJobInfo {
	reqs := tasklist.SortTasksWithPosition(k.reqList, k.groups, k.queuePositions, true)
	jobQinfo := tasklist.ReduceToJobQInfo(reqs)

	return jobQinfo
}

func (k *kubernetesResourcePool) receiveSetAllocationName(
	ctx *actor.Context,
	msg sproto.SetAllocationName,
) {
	if task, found := k.reqList.TaskByHandler(msg.AllocationRef); found {
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

	allocations := sproto.ResourceList{}
	for pod := 0; pod < numPods; pod++ {
		containerID := cproto.NewID()
		rs := &k8sPodResources{
			req:             req,
			podsActor:       k.podsActor,
			containerID:     containerID,
			slots:           slotsPerPod,
			group:           k.groups[req.Group],
			initialPosition: k.queuePositions[k.addrToJobID[req.AllocationRef]],
		}
		allocations[rs.Summary().ResourcesID] = rs
		k.addrToContainerID[req.AllocationRef] = containerID
		k.containerIDtoAddr[containerID.String()] = req.AllocationRef
	}

	assigned := sproto.ResourcesAllocated{ID: req.AllocationID, Resources: allocations}
	k.reqList.AddAllocationRaw(req.AllocationRef, &assigned)
	req.AllocationRef.System().Tell(req.AllocationRef, assigned.Clone())

	ctx.Log().
		WithField("allocation-id", req.AllocationID).
		WithField("task-handler", req.AllocationRef.Address()).
		Infof("resources assigned with %d pods", numPods)
}

func (k *kubernetesResourcePool) resourcesReleased(
	ctx *actor.Context,
	msg sproto.ResourcesReleased,
) {
	if msg.ResourcesID != nil {
		// Just ignore this minor optimization in Kubernetes.
		return
	}

	ctx.Log().Infof("resources are released for %s", msg.AllocationRef.Address())
	k.reqList.RemoveTaskByHandler(msg.AllocationRef)
	delete(k.addrToContainerID, msg.AllocationRef)

	deleteID := ""
	for id, addr := range k.containerIDtoAddr {
		if addr == msg.AllocationRef {
			deleteID = id
			delete(k.containerIDtoAddr, deleteID)
			break
		}
	}

	if req, ok := k.reqList.TaskByHandler(msg.AllocationRef); ok {
		group := k.groups[msg.AllocationRef]

		if group != nil {
			k.slotsUsedPerGroup[group] -= req.SlotsNeeded
		}
	}
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
		actors.NotifyOnStop(ctx, handler, tasklist.GroupActorStopped{})
	}
	return g
}

func (k *kubernetesResourcePool) schedulePendingTasks(ctx *actor.Context) {
	for it := k.reqList.Iterator(); it.Next(); {
		req := it.Value()
		group := k.groups[req.Group]
		assigned := k.reqList.Allocation(req.AllocationRef)
		if !tasklist.AssignmentIsScheduled(assigned) {
			if maxSlots := group.MaxSlots; maxSlots != nil {
				if k.slotsUsedPerGroup[group]+req.SlotsNeeded > *maxSlots {
					continue
				}
			}

			k.assignResources(ctx, req)
		}
	}
}
