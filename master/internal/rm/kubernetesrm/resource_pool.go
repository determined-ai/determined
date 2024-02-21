package kubernetesrm

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

const resourcePoolEnvVar = "DET_K8S_RESOURCE_POOL"

// getResourceSummary is a message to request a summary of the resources used by the
// resource pool (agents, slots, cpu containers).
type getResourceSummary struct{}

type kubernetesResourcePool struct {
	mu sync.Mutex

	maxSlotsPerPod int
	poolConfig     *config.ResourcePoolConfig

	reqList                   *tasklist.TaskList
	groups                    map[model.JobID]*tasklist.Group
	allocationIDToContainerID map[model.AllocationID]cproto.ID
	containerIDtoAllocationID map[string]model.AllocationID
	// TODO(DET-9613): Jobs have many allocs.
	jobIDToAllocationID       map[model.JobID]model.AllocationID
	allocationIDToJobID       map[model.AllocationID]model.JobID
	slotsUsedPerGroup         map[*tasklist.Group]int
	allocationIDToRunningPods map[model.AllocationID]int

	podsService *pods

	queuePositions tasklist.JobSortState
	reschedule     bool

	db *db.PgDB

	syslog *logrus.Entry
}

func newResourcePool(
	maxSlotsPerPod int,
	poolConfig *config.ResourcePoolConfig,
	podsService *pods,
	db *db.PgDB,
) *kubernetesResourcePool {
	return &kubernetesResourcePool{
		maxSlotsPerPod:            maxSlotsPerPod,
		poolConfig:                poolConfig,
		reqList:                   tasklist.New(),
		groups:                    map[model.JobID]*tasklist.Group{},
		allocationIDToContainerID: map[model.AllocationID]cproto.ID{},
		containerIDtoAllocationID: map[string]model.AllocationID{},
		jobIDToAllocationID:       map[model.JobID]model.AllocationID{},
		allocationIDToJobID:       map[model.AllocationID]model.JobID{},
		slotsUsedPerGroup:         map[*tasklist.Group]int{},
		allocationIDToRunningPods: map[model.AllocationID]int{},
		podsService:               podsService,
		queuePositions:            tasklist.InitializeJobSortState(true),
		db:                        db,
		syslog:                    logrus.WithField("component", "k8s-rp"),
	}
}

func (k *kubernetesResourcePool) SetGroupMaxSlots(msg sproto.SetGroupMaxSlots) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	k.getOrCreateGroup(msg.JobID).MaxSlots = msg.MaxSlots
}

func (k *kubernetesResourcePool) AllocateRequest(msg sproto.AllocateRequest) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	k.addTask(msg)
}

func (k *kubernetesResourcePool) ResourcesReleased(msg sproto.ResourcesReleased) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	k.resourcesReleased(msg)
}

func (k *kubernetesResourcePool) UpdatePodStatus(msg sproto.UpdatePodStatus) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	id, ok := k.containerIDtoAllocationID[msg.ContainerID]
	if !ok {
		return
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
}

func (k *kubernetesResourcePool) PendingPreemption(msg sproto.PendingPreemption) error {
	return rmerrors.ErrNotSupported
}

func (k *kubernetesResourcePool) GetJobQ(msg sproto.GetJobQ) map[model.JobID]*sproto.RMJobInfo {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	return k.jobQInfo()
}

func (k *kubernetesResourcePool) GetJobQStats(msg sproto.GetJobQStats) *jobv1.QueueStats {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	return tasklist.JobStats(k.reqList)
}

func (k *kubernetesResourcePool) GetJobQStatsAPI(msg *apiv1.GetJobQueueStatsRequest) *apiv1.GetJobQueueStatsResponse {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	resp := &apiv1.GetJobQueueStatsResponse{
		Results: make([]*apiv1.RPQueueStat, 0),
	}
	resp.Results = append(resp.Results, &apiv1.RPQueueStat{
		Stats:        tasklist.JobStats(k.reqList),
		ResourcePool: k.poolConfig.PoolName,
	})
	return resp
}

func (k *kubernetesResourcePool) SetGroupWeight(msg sproto.SetGroupWeight) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	return rmerrors.UnsupportedError("set group weight is unsupported in k8s")
}

func (k *kubernetesResourcePool) SetGroupPriority(msg sproto.SetGroupPriority) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	group := k.getOrCreateGroup(msg.JobID)
	// Check if there is already a submitted task in this group for which
	// priority is immutable. If so, respond with an error.
	for it := k.reqList.Iterator(); it.Next(); {
		if it.Value().JobID == msg.JobID {
			if req := it.Value(); !req.Preemptible {
				return rmerrors.UnsupportedError(fmt.Sprintf(
					"priority is immutable for %s in k8s because it may be destructive",
					req.Name,
				))
			}
		}
	}

	group.Priority = &msg.Priority
	// Do the destructive thing if the group has a submitted task, since it is only allowed
	// for trials and trials take checkpoints.
	for it := k.reqList.Iterator(); it.Next(); {
		if it.Value().JobID == msg.JobID {
			req := it.Value()
			if id, ok := k.allocationIDToContainerID[req.AllocationID]; ok {
				k.podsService.ChangePriority(ChangePriority{PodID: id})
				delete(k.allocationIDToContainerID, req.AllocationID)
			}
		}
	}
	return nil
}

func (k *kubernetesResourcePool) MoveJob(msg sproto.MoveJob) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	return k.moveJob(msg.ID, msg.Anchor, msg.Ahead)
}

func (k *kubernetesResourcePool) DeleteJob(msg sproto.DeleteJob) sproto.DeleteJobResponse {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	// For now, there is nothing to cleanup in k8s.
	return sproto.EmptyDeleteJobResponse()
}

func (k *kubernetesResourcePool) RecoverJobPosition(msg sproto.RecoverJobPosition) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	k.queuePositions.RecoverJobPosition(msg.JobID, msg.JobPosition)
}

func (k *kubernetesResourcePool) GetAllocationSummaries(
	msg sproto.GetAllocationSummaries,
) map[model.AllocationID]sproto.AllocationSummary {
	k.mu.Lock()
	defer k.mu.Unlock()

	return k.reqList.TaskSummaries(k.groups, kubernetesScheduler)
}

func (k *kubernetesResourcePool) getResourceSummary(msg getResourceSummary) (*resourceSummary, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	slotsUsed := 0
	for _, slotsUsedByGroup := range k.slotsUsedPerGroup {
		slotsUsed += slotsUsedByGroup
	}
	pods, err := k.summarizePods()
	if err != nil {
		return nil, err
	}

	return &resourceSummary{
		numAgents:              pods.NumAgents,
		numTotalSlots:          pods.SlotsAvailable,
		numActiveSlots:         slotsUsed,
		maxNumAuxContainers:    1,
		numActiveAuxContainers: 0,
		slotType:               "",
	}, nil
}

func (k *kubernetesResourcePool) ValidateResources(
	msg sproto.ValidateResourcesRequest,
) sproto.ValidateResourcesResponse {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.reschedule = true

	fulfillable := k.maxSlotsPerPod >= msg.Slots
	return sproto.ValidateResourcesResponse{Fulfillable: fulfillable}
}

func (k *kubernetesResourcePool) Schedule() {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.reschedule {
		k.schedulePendingTasks()
	}
	k.reschedule = false
}

func (k *kubernetesResourcePool) summarizePods() (*PodsInfo, error) {
	resp, err := k.podsService.SummarizeResources(SummarizeResources{PoolName: k.poolConfig.PoolName})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (k *kubernetesResourcePool) JobStopped(jobID model.JobID) {
	k.mu.Lock()
	defer k.mu.Unlock()

	delete(k.slotsUsedPerGroup, k.groups[jobID])
	delete(k.groups, jobID)
	delete(k.queuePositions, jobID)
	delete(k.allocationIDToJobID, k.jobIDToAllocationID[jobID])
	delete(k.jobIDToAllocationID, jobID)
}

func (k *kubernetesResourcePool) addTask(msg sproto.AllocateRequest) {
	if len(msg.AllocationID) == 0 {
		msg.AllocationID = model.AllocationID(uuid.New().String())
	}
	k.getOrCreateGroup(msg.JobID)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed-k8-Task"
	}

	k.syslog.WithField("restore", msg.Restore).Infof(
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
		k.allocationIDToRunningPods[msg.AllocationID] = 0
	}
	k.reqList.AddTask(&msg)
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

	if _, ok := k.groups[jobID]; !ok {
		return sproto.ErrJobNotFound(jobID)
	}
	if _, ok := k.queuePositions[anchorID]; !ok {
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
		g := k.getOrCreateGroup(jobID)
		oldPriority := g.Priority
		g.Priority = &anchorPriority

		if priorityChanger, ok := tasklist.GroupPriorityChangeRegistry.Load(jobID); ok {
			if priorityChanger != nil {
				if err := priorityChanger(anchorPriority); err != nil {
					g.Priority = oldPriority
					return err
				}
			}
		} else {
			return fmt.Errorf("unable to move job with ID %s", jobID)
		}
	}

	jobPosition, err := k.queuePositions.SetJobPosition(jobID, anchorID, secondAnchor, aheadOf, true)
	if err != nil {
		return err
	}
	if err := k.db.UpdateJobPosition(jobID, jobPosition); err != nil {
		return err
	}

	allocationID, ok := k.jobIDToAllocationID[jobID]
	if !ok {
		return fmt.Errorf("job with ID %s has no valid task address", jobID)
	}
	containerID, ok := k.allocationIDToContainerID[allocationID]
	if !ok {
		return fmt.Errorf("job with ID %s has no valid containerID", jobID)
	}

	k.podsService.ChangePosition(ChangePosition{PodID: containerID})

	return nil
}

func (k *kubernetesResourcePool) correctJobQInfo(
	reqs []*sproto.AllocateRequest,
	q map[model.JobID]*sproto.RMJobInfo,
) map[model.JobID]*sproto.RMJobInfo {
	jobIDToAllocatedSlots := map[model.JobID]int{}
	for _, req := range reqs {
		runningPods := k.allocationIDToRunningPods[req.AllocationID]
		if req.SlotsNeeded <= k.maxSlotsPerPod {
			jobIDToAllocatedSlots[req.JobID] += runningPods * req.SlotsNeeded
		} else {
			jobIDToAllocatedSlots[req.JobID] += runningPods * k.maxSlotsPerPod
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

func (k *kubernetesResourcePool) assignResources(
	req *sproto.AllocateRequest,
) {
	numPods := 1
	slotsPerPod := req.SlotsNeeded
	if req.SlotsNeeded > 1 {
		if k.maxSlotsPerPod == 0 {
			k.syslog.WithField("allocation-id", req.AllocationID).Error(
				"set max_slots_per_pod > 0 to schedule tasks with slots")
			return
		}

		if req.SlotsNeeded <= k.maxSlotsPerPod {
			numPods = 1
			slotsPerPod = req.SlotsNeeded
		} else {
			if req.SlotsNeeded%k.maxSlotsPerPod != 0 {
				k.syslog.WithField("allocation-id", req.AllocationID).Errorf(
					"task number of slots (%d) is not schedulable on the configured "+
						"max_slots_per_pod (%d)", req.SlotsNeeded, k.maxSlotsPerPod)
				return
			}

			numPods = req.SlotsNeeded / k.maxSlotsPerPod
			slotsPerPod = k.maxSlotsPerPod
		}
	}

	group := k.groups[req.JobID]
	if group == nil {
		k.syslog.WithField("allocation-id", req.AllocationID).Errorf(
			"cannot find group for job %s", req.JobID)
		return
	}
	k.slotsUsedPerGroup[group] += req.SlotsNeeded

	var resources []*k8sPodResources
	if req.Restore {
		var err error
		resources, err = k.restoreResources(req, slotsPerPod, numPods)
		if err != nil {
			k.syslog.
				WithField("allocation-id", req.AllocationID).
				WithError(err).Error("unable to restore allocation")
			unknownExit := sproto.ExitCode(-1)
			rmevents.Publish(req.AllocationID, &sproto.ResourcesRestoreError{
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
		err := k.podsService.RefreshPodStates(refreshPodStates{allocationID: req.AllocationID})
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
			podsService:     k.podsService,
			containerID:     cproto.NewID(),
			slots:           slotsPerPod,
			group:           k.groups[req.JobID],
			initialPosition: k.queuePositions[k.allocationIDToJobID[req.AllocationID]],
			namespace:       k.poolConfig.KubernetesNamespace,
		})
	}
	return resources
}

func (k *kubernetesResourcePool) restoreResources(
	req *sproto.AllocateRequest, slotsPerPod, numPods int,
) ([]*k8sPodResources, error) {
	restoreResponses, err := k.podsService.ReattachAllocationPods(reattachAllocationPods{
		req:          req,
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
			podsService:     k.podsService,
			containerID:     cproto.ID(restoreResponse.containerID),
			slots:           slotsPerPod,
			group:           k.groups[req.JobID],
			initialPosition: k.queuePositions[k.allocationIDToJobID[req.AllocationID]],
			namespace:       k.poolConfig.KubernetesNamespace,

			started: restoreResponse.started,
		})
	}

	return resources, nil
}

func (k *kubernetesResourcePool) resourcesReleased(
	msg sproto.ResourcesReleased,
) {
	req, ok := k.reqList.TaskByID(msg.AllocationID)
	if !ok {
		k.syslog.Debugf("ignoring release for task not allocated to pool %s", msg.AllocationID)
		return
	}

	if msg.ResourcesID != nil {
		// Just ignore this minor optimization in Kubernetes.
		return
	}

	k.syslog.Infof("resources are released for %s", msg.AllocationID)
	group := k.groups[req.JobID]
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

func (k *kubernetesResourcePool) getOrCreateGroup(jobID model.JobID) *tasklist.Group {
	if g, ok := k.groups[jobID]; ok {
		return g
	}
	priority := config.KubernetesDefaultPriority
	g := &tasklist.Group{JobID: jobID, Weight: 1, Priority: &priority}

	k.groups[jobID] = g
	k.slotsUsedPerGroup[g] = 0

	tasklist.GroupPriorityChangeRegistry.OnDelete(jobID, func() {
		k.JobStopped(jobID)
	})
	return g
}

func (k *kubernetesResourcePool) schedulePendingTasks() {
	for it := k.reqList.Iterator(); it.Next(); {
		req := it.Value()
		group := k.groups[req.JobID]
		if group == nil {
			k.syslog.Warnf("schedulePendingTasks cannot find group for job %s", req.JobID)
			continue
		}
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
	podsService     *pods
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
			aproto.ID("pods"): make([]device.Device, p.slots),
		},

		ContainerID: &p.containerID,
		Started:     p.started,
	}
}

// Start notifies the pods actor that it should launch a pod for the provided task spec.
func (p k8sPodResources) Start(
	logCtx logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
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
	return p.podsService.StartTaskPod(StartTaskPod{
		Req:          p.req,
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
func (p k8sPodResources) Kill(_ logger.Context) {
	p.podsService.KillPod(KillTaskPod{
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
