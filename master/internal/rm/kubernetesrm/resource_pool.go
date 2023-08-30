package kubernetesrm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

const resourcePoolEnvVar = "DET_K8S_RESOURCE_POOL"

type kubernetesResourcePool struct {
	config     *config.KubernetesResourceManagerConfig
	poolConfig *config.ResourcePoolConfig

	syslog *logrus.Entry

	mu *sync.Mutex
	eg errgroupx.Group

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

	pods *pods

	queuePositions tasklist.JobSortState
	reschedule     bool

	system *actor.System
}

func newResourcePool(
	rmConfig *config.KubernetesResourceManagerConfig,
	poolConfig *config.ResourcePoolConfig,
	pods *pods,
) *kubernetesResourcePool {
	return &kubernetesResourcePool{
		config:     rmConfig,
		poolConfig: poolConfig,
		syslog: logrus.
			WithField("component", "k8s-resource-pool").
			WithField("name", poolConfig.PoolName),
		eg:                        errgroupx.WithContext(context.Background()),
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
		pods:                      pods,
		queuePositions:            tasklist.InitializeJobSortState(true),
	}
}

func (k *kubernetesResourcePool) periodicallySchedule() {
	t := time.NewTicker(ActionCoolDown)
	defer t.Stop()
	for range t.C {
		k.schedulePendingTasks()
	}
}

func (k *kubernetesResourcePool) AllocationSummary(
	req sproto.GetAllocationSummary,
) *sproto.AllocationSummary {
	k.mu.Lock()
	defer k.mu.Unlock()

	return k.reqList.TaskSummary(req.ID, k.groups, kubernetesScheduler)
}

func (k *kubernetesResourcePool) AllocationSummaries(
	req sproto.GetAllocationSummaries,
) map[model.AllocationID]sproto.AllocationSummary {
	k.mu.Lock()
	defer k.mu.Unlock()

	return k.reqList.TaskSummaries(k.groups, kubernetesScheduler)
}

func (k *kubernetesResourcePool) getResourceSummary() (resourceSummary, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	slotsUsed := 0
	for _, slotsUsedByGroup := range k.slotsUsedPerGroup {
		slotsUsed += slotsUsedByGroup
	}

	resp, err := k.pods.SummarizeResources(k.poolConfig.PoolName)
	if err != nil {
		return resourceSummary{}, fmt.Errorf("getting resource summary: %w", err)
	}

	return resourceSummary{
		numAgents:              resp.NumAgents,
		numTotalSlots:          resp.SlotsAvailable,
		numActiveSlots:         slotsUsed,
		maxNumAuxContainers:    1,
		numActiveAuxContainers: 0,
		slotType:               "",
	}, nil
}

func (k *kubernetesResourcePool) ValidateCommandResources(
	req sproto.ValidateCommandResourcesRequest,
) sproto.ValidateCommandResourcesResponse {
	fulfillable := k.config.MaxSlotsPerPod >= req.Slots
	return sproto.ValidateCommandResourcesResponse{Fulfillable: fulfillable}
}

// TODO(!!!): possibly inline. also, better name.
func (k *kubernetesResourcePool) GroupActorStopped(ref *actor.Ref) {
	k.mu.Lock()
	defer k.mu.Unlock()

	delete(k.slotsUsedPerGroup, k.groups[ref])
	delete(k.groups, ref)
	if jobID, ok := k.groupActorToID[ref]; ok {
		delete(k.queuePositions, jobID)
		delete(k.allocationIDToJobID, k.jobIDToAllocationID[jobID])
		delete(k.jobIDToAllocationID, jobID)
		delete(k.groupActorToID, ref)
		delete(k.IDToGroupActor, jobID)
	}
}

func (k *kubernetesResourcePool) SetGroupMaxSlots(req sproto.SetGroupMaxSlots) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.getOrCreateGroup(req.Handler).MaxSlots = req.MaxSlots
}

func (k *kubernetesResourcePool) SetAllocationName(req sproto.SetAllocationName) {
	k.mu.Lock()
	defer k.mu.Unlock()

	task, ok := k.reqList.TaskByID(req.AllocationID)
	if !ok {
		return
	}
	task.Name = req.Name
}

func (k *kubernetesResourcePool) AllocateRequest(req sproto.AllocateRequest) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if len(req.AllocationID) == 0 { // TODO(!!!): This is bogus validation.
		req.AllocationID = model.AllocationID(uuid.New().String())
	}

	k.getOrCreateGroup(req.Group)
	if len(req.Name) == 0 {
		req.Name = "Unnamed-k8-Task"
	}

	k.syslog.WithField("restore", req.Restore).Infof(
		"resources are requested by %s (Allocation ID: %s)",
		req.Name, req.AllocationID,
	)
	if req.IsUserVisible {
		if _, ok := k.queuePositions[req.JobID]; !ok {
			k.queuePositions[req.JobID] = tasklist.InitializeQueuePosition(
				req.JobSubmissionTime,
				true,
			)
		}
		k.jobIDToAllocationID[req.JobID] = req.AllocationID
		k.allocationIDToJobID[req.AllocationID] = req.JobID
		k.groupActorToID[req.Group] = req.JobID
		k.IDToGroupActor[req.JobID] = req.Group
		k.allocationIDToRunningPods[req.AllocationID] = 0
	}
	k.reqList.AddTask(&req)
}

func (k *kubernetesResourcePool) ResourcesReleased(req sproto.ResourcesReleased) {
	k.mu.Lock()
	defer k.mu.Unlock()

	ar, ok := k.reqList.TaskByID(req.AllocationID)
	if !ok {
		k.syslog.Debugf("ignoring release for task not allocated to pool %s", req.AllocationID)
		return
	}

	if req.ResourcesID != nil {
		// Just ignore this minor optimization in Kubernetes.
		return
	}

	k.syslog.Infof("resources are released for %s", ar.AllocationID)
	group := k.groups[ar.Group]
	if group != nil {
		k.slotsUsedPerGroup[group] -= ar.SlotsNeeded
	}

	k.reqList.RemoveTaskByID(ar.AllocationID)
	delete(k.allocationIDToContainerID, ar.AllocationID)
	delete(k.allocationIDToRunningPods, ar.AllocationID)

	for id, addr := range k.containerIDtoAllocationID {
		if addr == ar.AllocationID {
			delete(k.containerIDtoAllocationID, id)
			break
		}
	}
}

func (k *kubernetesResourcePool) UpdatePodStatus(req sproto.UpdatePodStatus) {
	k.mu.Lock()
	defer k.mu.Unlock()

	id, ok := k.containerIDtoAllocationID[req.ContainerID]
	if !ok {
		return
	}

	for it := k.reqList.Iterator(); it.Next(); {
		req := it.Value()
		if req.AllocationID == id {
			req.State = req.State
			if sproto.ScheduledStates[req.State] {
				k.allocationIDToRunningPods[id]++
			}
		}
	}
}

func (k *kubernetesResourcePool) JobQueue(req sproto.GetJobQ) map[model.JobID]*sproto.RMJobInfo {
	k.mu.Lock()
	defer k.mu.Unlock()

	reqs := tasklist.SortTasksWithPosition(k.reqList, k.groups, k.queuePositions, true)
	jobQInfo := tasklist.ReduceToJobQInfo(reqs)
	correctedJobQInfo := k.correctJobQInfo(reqs, jobQInfo)
	return correctedJobQInfo
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

func (k *kubernetesResourcePool) JobQueueStats(
	req *apiv1.GetJobQueueStatsRequest,
) *apiv1.GetJobQueueStatsResponse {
	k.mu.Lock()
	defer k.mu.Unlock()

	return &apiv1.GetJobQueueStatsResponse{
		Results: []*apiv1.RPQueueStat{
			{
				Stats:        tasklist.JobStats(k.reqList),
				ResourcePool: k.poolConfig.PoolName,
			},
		},
	}
}

func (k *kubernetesResourcePool) JobQStats() *jobv1.QueueStats {
	k.mu.Lock()
	defer k.mu.Unlock()
	return tasklist.JobStats(k.reqList)
}

func (k *kubernetesResourcePool) MoveJob(req sproto.MoveJob) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.moveJob(req.ID, req.Anchor, req.Ahead)
}

func (k *kubernetesResourcePool) moveJob(
	jobID model.JobID,
	anchorID model.JobID,
	aheadOf bool,
) error {
	for it := k.reqList.Iterator(); it.Next(); {
		if it.Value().JobID == jobID {
			if req := it.Value(); !req.Preemptible {
				return fmt.Errorf(
					"move job for %s unsupported in k8s because it may be destructive",
					req.Name,
				)
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
		g := k.getOrCreateGroup(k.IDToGroupActor[jobID])
		oldPriority := g.Priority
		g.Priority = &anchorPriority
		resp := k.system.Ask(k.IDToGroupActor[jobID], sproto.NotifyRMPriorityChange{
			Priority: anchorPriority,
		}) // TODO(actors): Jobs are webs of lies.
		if resp.Error() != nil {
			g.Priority = oldPriority
			return resp.Error()
		}
	}

	req, err := k.queuePositions.SetJobPosition(jobID, anchorID, secondAnchor, aheadOf, true)
	if err != nil {
		return err
	}
	k.system.Tell(groupAddr, req) // TODO(actors): Jobs are webs of lies.

	allocationID, ok := k.jobIDToAllocationID[jobID]
	if !ok {
		return fmt.Errorf("job with ID %s has no valid task address", jobID)
	}
	containerID, ok := k.allocationIDToContainerID[allocationID]
	if !ok {
		return fmt.Errorf("job with ID %s has no valid containerID", jobID)
	}

	k.pods.ChangePosition(containerID)
	return nil
}

func (k *kubernetesResourcePool) SetGroupPriority(req sproto.SetGroupPriority) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	group := k.getOrCreateGroup(req.Handler)
	// Check if there is already a submitted task in this group for which
	// priority is immutable. If so, respond with an error.
	for it := k.reqList.Iterator(); it.Next(); {
		if it.Value().Group == req.Handler {
			if req := it.Value(); !req.Preemptible {
				return rmerrors.ErrUnsupported(fmt.Sprintf(
					"priority is immutable for %s in k8s because it may be destructive",
					req.Name,
				))
			}
		}
	}

	group.Priority = &req.Priority
	// Do the destructive thing if the group has a submitted task, since it is only allowed
	// for trials and trials take checkpoints.
	for it := k.reqList.Iterator(); it.Next(); {
		if it.Value().Group == req.Handler {
			req := it.Value()
			if id, ok := k.allocationIDToContainerID[req.AllocationID]; ok {
				k.pods.ChangePriority(id)
				delete(k.allocationIDToContainerID, req.AllocationID)
			}
		}
	}
	return nil
}

func (k *kubernetesResourcePool) RecoverJobPosition(req sproto.RecoverJobPosition) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.queuePositions.RecoverJobPosition(req.JobID, req.JobPosition)
}

func (k *kubernetesResourcePool) DeleteJob() sproto.DeleteJobResponse {
	return sproto.EmptyDeleteJobResponse()
}

func (k *kubernetesResourcePool) assignResources(req *sproto.AllocateRequest) {
	numPods := 1
	slotsPerPod := req.SlotsNeeded
	if req.SlotsNeeded > 1 {
		if k.config.MaxSlotsPerPod == 0 {
			k.syslog.WithField("allocation-id", req.AllocationID).Error(
				"set max_slots_per_pod > 0 to schedule tasks with slots")
			return
		}

		if req.SlotsNeeded <= k.config.MaxSlotsPerPod {
			numPods = 1
			slotsPerPod = req.SlotsNeeded
		} else {
			if req.SlotsNeeded%k.config.MaxSlotsPerPod != 0 {
				k.syslog.WithField("allocation-id", req.AllocationID).Errorf(
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
		resources, err = k.restoreResources(req, slotsPerPod, numPods)
		if err != nil {
			k.syslog.
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
		resources = k.createResources(req, slotsPerPod, numPods)
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
		k.syslog.
			WithField("allocation-id", req.AllocationID).
			WithField("task-handler", req.Name).
			Infof("resources restored with %d pods", numPods)
	} else {
		k.syslog.
			WithField("allocation-id", req.AllocationID).
			WithField("task-handler", req.Name).
			Infof("resources assigned with %d pods", numPods)
	}

	if req.Restore {
		// This call must happen after we publish ResourcesAllocated, otherwise the allocation will
		// receive an update for resources it does not know about, ignore it, then hang if it missed
		// the termination.
		err := k.pods.refreshPodStates(req.AllocationID)
		if err != nil {
			k.syslog.WithError(err).Error("failed to refresh pod states after reattach")
		}
	}
}

func (k *kubernetesResourcePool) createResources(
	req *sproto.AllocateRequest, slotsPerPod, numPods int,
) []*k8sPodResources {
	var resources []*k8sPodResources
	for pod := 0; pod < numPods; pod++ {
		resources = append(resources, &k8sPodResources{
			req:             req,
			pods:            k.pods,
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
	req *sproto.AllocateRequest, slotsPerPod, numPods int,
) ([]*k8sPodResources, error) {
	restoreResponses, err := k.pods.reattachPods(reattachPodsRequest{
		allocationID: req.AllocationID,
		numPods:      numPods,
		slots:        slotsPerPod,
		logContext:   req.LogContext,
	})
	if err != nil {
		return nil, err
	}

	var resources []*k8sPodResources
	for _, restoreResponse := range restoreResponses {
		resources = append(resources, &k8sPodResources{
			req:             req,
			pods:            k.pods,
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

func (k *kubernetesResourcePool) getOrCreateGroup(handler *actor.Ref) *tasklist.Group {
	if g, ok := k.groups[handler]; ok {
		return g
	}
	priority := config.KubernetesDefaultPriority
	g := &tasklist.Group{Handler: handler, Weight: 1, Priority: &priority}

	k.groups[handler] = g
	k.slotsUsedPerGroup[g] = 0

	if handler != nil {
		k.eg.Go(func(ctx context.Context) error {
			_ = handler.AwaitTerminationCtx(ctx)

			k.GroupActorStopped(handler)
			return nil
		})
	}
	return g
}

func (k *kubernetesResourcePool) schedulePendingTasks() {
	for it := k.reqList.Iterator(); it.Next(); {
		req := it.Value()
		group := k.groups[req.Group]
		if !k.reqList.IsScheduled(req.AllocationID) {
			if maxSlots := group.MaxSlots; maxSlots != nil {
				if k.slotsUsedPerGroup[group]+req.SlotsNeeded > *maxSlots {
					continue
				}
			}

			k.assignResources(req)
		}
	}
}

type k8sPodResources struct {
	req             *sproto.AllocateRequest
	pods            *pods
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
			aproto.ID(p.containerID): make([]device.Device, p.slots),
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
	return p.pods.StartTaskPod(StartTaskPodRequest{
		AllocationID: p.req.AllocationID,
		Spec:         spec,
		Slots:        p.slots,
		Rank:         rri.AgentRank,
		Namespace:    p.namespace,
		LogContext:   logCtx,
	})
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
	p.pods.KillTaskPod(p.containerID)
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
