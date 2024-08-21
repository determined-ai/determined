package kubernetesrm

import (
	"context"
	"database/sql"
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
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

const resourcePoolEnvVar = "DET_K8S_RESOURCE_POOL"

type kubernetesResourcePool struct {
	mu sync.Mutex

	maxSlotsPerPod   int
	poolConfig       *config.ResourcePoolConfig
	defaultNamespace string
	clusterName      string

	reqList *tasklist.TaskList
	groups  map[model.JobID]*tasklist.Group
	// TODO(DET-9613): Jobs have many allocs.
	jobIDToAllocationID       map[model.JobID]model.AllocationID
	allocationIDToJobID       map[model.AllocationID]model.JobID
	slotsUsedPerGroup         map[*tasklist.Group]int
	allocationIDToRunningPods map[model.AllocationID]int

	jobsService *jobsService

	queuePositions       tasklist.JobSortState
	tryAdmitPendingTasks bool

	db *db.PgDB

	syslog *logrus.Entry
}

func newResourcePool(
	maxSlotsPerPod int,
	poolConfig *config.ResourcePoolConfig,
	jobsService *jobsService,
	db *db.PgDB,
	defaultNamespace string,
	clusterName string,
) *kubernetesResourcePool {
	return &kubernetesResourcePool{
		maxSlotsPerPod:            maxSlotsPerPod,
		poolConfig:                poolConfig,
		reqList:                   tasklist.New(),
		groups:                    map[model.JobID]*tasklist.Group{},
		jobIDToAllocationID:       map[model.JobID]model.AllocationID{},
		allocationIDToJobID:       map[model.AllocationID]model.JobID{},
		slotsUsedPerGroup:         map[*tasklist.Group]int{},
		allocationIDToRunningPods: map[model.AllocationID]int{},
		jobsService:               jobsService,
		queuePositions:            tasklist.InitializeJobSortState(true),
		db:                        db,
		syslog:                    logrus.WithField("component", "k8s-rp"),
		defaultNamespace:          defaultNamespace,
		clusterName:               clusterName,
	}
}

func (k *kubernetesResourcePool) SetGroupMaxSlots(msg sproto.SetGroupMaxSlots) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	k.getOrCreateGroup(msg.JobID).MaxSlots = msg.MaxSlots
}

func (k *kubernetesResourcePool) AllocateRequest(msg sproto.AllocateRequest) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	k.addTask(msg)
}

func (k *kubernetesResourcePool) ResourcesReleased(msg sproto.ResourcesReleased) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	k.resourcesReleased(msg)
}

func (k *kubernetesResourcePool) JobSchedulingStateChanged(msg jobSchedulingStateChanged) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	for it := k.reqList.Iterator(); it.Next(); {
		req := it.Value()
		if req.AllocationID == msg.AllocationID {
			req.State = msg.State
			if sproto.ScheduledStates[req.State] {
				k.allocationIDToRunningPods[msg.AllocationID] = msg.NumPods
			}
		}
	}
}

func (k *kubernetesResourcePool) PendingPreemption(msg sproto.PendingPreemption) error {
	return rmerrors.ErrNotSupported
}

func (k *kubernetesResourcePool) GetJobQ() map[model.JobID]*sproto.RMJobInfo {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	return k.jobQInfo()
}

func (k *kubernetesResourcePool) GetJobQStats() *jobv1.QueueStats {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	return tasklist.JobStats(k.reqList)
}

func (k *kubernetesResourcePool) GetJobQStatsAPI(msg *apiv1.GetJobQueueStatsRequest) *apiv1.GetJobQueueStatsResponse {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

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
	k.tryAdmitPendingTasks = true

	return rmerrors.UnsupportedError("set group weight is unsupported in k8s")
}

func (k *kubernetesResourcePool) SetGroupPriority(msg sproto.SetGroupPriority) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

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
			k.jobsService.ChangePriority(req.AllocationID)
		}
	}
	return nil
}

func (k *kubernetesResourcePool) DeleteJob(msg sproto.DeleteJob) sproto.DeleteJobResponse {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	// For now, there is nothing to cleanup in k8s.
	return sproto.EmptyDeleteJobResponse()
}

func (k *kubernetesResourcePool) RecoverJobPosition(msg sproto.RecoverJobPosition) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	k.queuePositions.RecoverJobPosition(msg.JobID, msg.JobPosition)
}

func (k *kubernetesResourcePool) GetAllocationSummaries() map[model.AllocationID]sproto.AllocationSummary {
	k.mu.Lock()
	defer k.mu.Unlock()

	return k.reqList.TaskSummaries(k.groups, kubernetesScheduler)
}

func (k *kubernetesResourcePool) GetAllocationSummary(id model.AllocationID) *sproto.AllocationSummary {
	k.mu.Lock()
	defer k.mu.Unlock()

	return k.reqList.TaskSummary(id, k.groups, kubernetesScheduler)
}

func (k *kubernetesResourcePool) getResourceSummary() (*resourceSummary, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	slotsUsed := 0
	for _, slotsUsedByGroup := range k.slotsUsedPerGroup {
		slotsUsed += slotsUsedByGroup
	}
	pods, err := k.summarizePods()
	if err != nil {
		return nil, err
	}

	return &resourceSummary{
		numAgents:              pods.numAgentsUsed,
		numTotalSlots:          pods.slotsAvailable,
		numActiveSlots:         slotsUsed,
		maxNumAuxContainers:    1,
		numActiveAuxContainers: 0,
		slotType:               "",
	}, nil
}

func (k *kubernetesResourcePool) ValidateResources(
	msg sproto.ValidateResourcesRequest,
) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.tryAdmitPendingTasks = true

	fulfillable := k.maxSlotsPerPod >= msg.Slots

	if msg.IsSingleNode {
		if !fulfillable {
			return fmt.Errorf(
				"invalid resource request: slots (%d) must be < max_slots_per_pod (%d) on single pod",
				msg.Slots,
				k.maxSlotsPerPod,
			)
		}
		return nil
	}
	fulfillable = fulfillable || msg.Slots%k.maxSlotsPerPod == 0
	if !fulfillable {
		return fmt.Errorf(
			"invalid resource request: slots (%d) must be < or a multiple of max_slots_per_pod (%d)",
			msg.Slots,
			k.maxSlotsPerPod,
		)
	}
	return nil
}

func (k *kubernetesResourcePool) Admit() {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.tryAdmitPendingTasks {
		k.admitPendingTasks()
	}
	k.tryAdmitPendingTasks = false
}

func (k *kubernetesResourcePool) summarizePods() (*computeUsageSummary, error) {
	resp, err := k.jobsService.SummarizeResources(k.poolConfig.PoolName)
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

		if req.SlotsNeeded > k.maxSlotsPerPod {
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
		k.syslog.WithField("allocation-id", req.AllocationID).Errorf("cannot find group for job %s", req.JobID)
		return
	}
	k.slotsUsedPerGroup[group] += req.SlotsNeeded

	var resources *k8sJobResource
	if req.Restore {
		var err error
		resources, err = k.restoreResources(req, slotsPerPod, numPods)
		if err != nil {
			k.syslog.
				WithField("allocation-id", req.AllocationID).
				WithError(err).Error("unable to restore allocation")
			unknownExit := sproto.ExitCode(-1)
			rmevents.Publish(req.AllocationID, &sproto.ResourcesFailedError{
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
	allocations[resources.Summary().ResourcesID] = resources

	assigned := sproto.ResourcesAllocated{
		ID:                req.AllocationID,
		Resources:         allocations,
		JobSubmissionTime: req.JobSubmissionTime,
		Recovered:         req.Restore,
	}
	k.reqList.AddAllocationRaw(req.AllocationID, &assigned)
	rmevents.Publish(req.AllocationID, assigned.Clone())

	if req.Restore {
		k.syslog.
			WithField("allocation-id", req.AllocationID).
			WithField("task-handler", req.Name).
			WithField("num-pods", numPods).
			Infof("restored kubernetes job")
	} else {
		k.syslog.
			WithField("allocation-id", req.AllocationID).
			WithField("task-handler", req.Name).
			WithField("num-pods", numPods).
			Infof("admitting kubernetes job")
	}

	if req.Restore {
		// This call must happen after we publish ResourcesAllocated, otherwise the allocation will
		// receive an update for resources it does not know about, ignore it, then hang if it missed
		// the termination.
		err := k.jobsService.RefreshStates(req.AllocationID)
		if err != nil {
			k.syslog.WithError(err).Error("failed to refresh pod states after reattach")
		}
	}
}

func (k *kubernetesResourcePool) createResources(
	req *sproto.AllocateRequest, slotsPerPod, numPods int,
) *k8sJobResource {
	return &k8sJobResource{
		numPods:          numPods,
		req:              req,
		jobsService:      k.jobsService,
		slots:            slotsPerPod,
		group:            k.groups[req.JobID],
		initialPosition:  k.queuePositions[k.allocationIDToJobID[req.AllocationID]],
		defaultNamespace: k.defaultNamespace,
		clusterName:      k.clusterName,
	}
}

func (k *kubernetesResourcePool) restoreResources(
	req *sproto.AllocateRequest, slotsPerPod, numPods int,
) (*k8sJobResource, error) {
	restored, err := k.jobsService.ReattachJob(reattachJobRequest{
		req:          req,
		allocationID: req.AllocationID,
		numPods:      numPods,
		slots:        slotsPerPod,
		logContext:   req.LogContext,
	})
	if err != nil {
		return nil, err
	}

	return &k8sJobResource{
		req:              req,
		jobsService:      k.jobsService,
		slots:            slotsPerPod,
		group:            k.groups[req.JobID],
		initialPosition:  k.queuePositions[k.allocationIDToJobID[req.AllocationID]],
		defaultNamespace: k.defaultNamespace,
		clusterName:      k.clusterName,

		started: restored.started,
	}, nil
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
	delete(k.allocationIDToRunningPods, msg.AllocationID)

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

func (k *kubernetesResourcePool) admitPendingTasks() {
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

type k8sJobResource struct {
	req              *sproto.AllocateRequest
	jobsService      *jobsService
	group            *tasklist.Group
	slots            int
	numPods          int
	initialPosition  decimal.Decimal
	defaultNamespace string
	clusterName      string

	started *sproto.ResourcesStarted
}

// Summary summarizes a container allocation.
func (p k8sJobResource) Summary() sproto.ResourcesSummary {
	return sproto.ResourcesSummary{
		AllocationID:  p.req.AllocationID,
		ResourcesID:   sproto.ResourcesID(p.req.AllocationID),
		ResourcesType: sproto.ResourcesTypeK8sJob,
		AgentDevices: map[aproto.ID][]device.Device{
			// TODO: Make it more obvious k8s can't be trusted.
			aproto.ID("pods"): make([]device.Device, p.slots*p.numPods),
		},

		Started: p.started,
	}
}

// Start notifies the pods actor that it should launch a pod for the provided task spec.
func (p k8sJobResource) Start(
	logCtx logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
) error {
	p.setPosition(&spec)
	spec.ContainerID = string(p.req.AllocationID)
	spec.ResourcesID = string(p.req.AllocationID)
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

	if spec.ExtraEnvVars == nil {
		spec.ExtraEnvVars = map[string]string{}
	}
	spec.ExtraEnvVars[sproto.ResourcesTypeEnvVar] = string(sproto.ResourcesTypeK8sJob)
	spec.ExtraEnvVars[resourcePoolEnvVar] = p.req.ResourcePool

	ns, err := workspace.GetNamespaceFromWorkspace(context.TODO(), spec.Workspace, p.clusterName)
	if errors.Is(err, db.ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		ns = p.defaultNamespace
	} else if err != nil {
		return fmt.Errorf("getting namespace for workspace: %w", err)
	}

	return p.jobsService.StartJob(startJob{
		req:          p.req,
		allocationID: p.req.AllocationID,
		spec:         spec,
		slots:        p.slots,
		rank:         rri.AgentRank,
		resourcePool: p.req.ResourcePool,
		namespace:    ns,
		numPods:      p.numPods,
		logContext:   logCtx,
	})
}

func (p k8sJobResource) setPosition(spec *tasks.TaskSpec) {
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
func (p k8sJobResource) Kill(_ logger.Context) {
	p.jobsService.KillJob(p.req.AllocationID)
}

func (p k8sJobResource) Persist() error {
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
