package kubernetesrm

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const (
	// KubernetesDummyResourcePool is the name of the dummy resource pool for kubernetes.
	KubernetesDummyResourcePool = "kubernetes"
	// KubernetesScheduler is the "name" of the kubernetes scheduler, for informational reasons.
	kubernetesScheduler = "kubernetes"
	// ActionCoolDown is the rate limit for job submission.
	ActionCoolDown = 500 * time.Millisecond
)

// SchedulerTick notifies the Resource Manager to submit pending jobs.
type SchedulerTick struct{}

// ResourceManager is a resource manager that manages k8s resources.
type ResourceManager struct {
	*actorrm.ResourceManager
}

// New returns a new ResourceManager, which communicates with
// and submits work to a Kubernetes apiserver.
func New(
	system *actor.System,
	db *db.PgDB,
	echo *echo.Echo,
	config *config.ResourceConfig,
	opts *aproto.MasterSetAgentOptions,
	cert *tls.Certificate,
) ResourceManager {
	tlsConfig, err := model.MakeTLSConfig(cert)
	if err != nil {
		panic(errors.Wrap(err, "failed to set up TLS config"))
	}
	ref, _ := system.ActorOf(
		sproto.K8sRMAddr,
		newKubernetesResourceManager(
			config.ResourceManager.KubernetesRM,
			echo,
			tlsConfig,
			opts.LoggingOptions,
		),
	)
	system.Ask(ref, actor.Ping{}).Get()
	return ResourceManager{ResourceManager: actorrm.Wrap(ref)}
}

// GetResourcePoolRef just returns the k8s RM actor, since it is a superset of the RP API,
// and k8s has no resource pools.
func (k ResourceManager) GetResourcePoolRef(
	ctx actor.Messenger,
	name string,
) (*actor.Ref, error) {
	return k.Ref(), nil
}

// ResolveResourcePool resolves the resource pool completely.
func (k ResourceManager) ResolveResourcePool(
	ctx actor.Messenger,
	name string,
	slots int,
) (string, error) {
	return KubernetesDummyResourcePool, k.ValidateResourcePool(ctx, name)
}

// ValidateResources ensures enough resources are available in the resource pool.
func (k ResourceManager) ValidateResources(
	ctx actor.Messenger,
	name string,
	slots int,
	command bool,
) error {
	return nil
}

// ValidateResourcePool validates a resource pool is none or the k8s dummy pool.
func (k ResourceManager) ValidateResourcePool(ctx actor.Messenger, name string) error {
	if name != "" && name != KubernetesDummyResourcePool {
		return fmt.Errorf("k8s doesn't not support resource pools")
	}
	return nil
}

// ValidateResourcePoolAvailability checks the available resources for a given pool.
func (k ResourceManager) ValidateResourcePoolAvailability(
	ctx actor.Messenger,
	name string,
	slots int,
) ([]command.LaunchWarning, error) {
	if name != "" && name != KubernetesDummyResourcePool {
		return nil, fmt.Errorf("k8s doesn't not support resource pools")
	}
	return nil, nil
}

// GetDefaultComputeResourcePool requests the default compute resource pool.
func (k ResourceManager) GetDefaultComputeResourcePool(
	ctx actor.Messenger,
	msg sproto.GetDefaultComputeResourcePoolRequest,
) (sproto.GetDefaultComputeResourcePoolResponse, error) {
	return sproto.GetDefaultComputeResourcePoolResponse{
		PoolName: KubernetesDummyResourcePool,
	}, nil
}

// GetDefaultAuxResourcePool requests the default aux resource pool.
func (k ResourceManager) GetDefaultAuxResourcePool(
	ctx actor.Messenger,
	msg sproto.GetDefaultAuxResourcePoolRequest,
) (sproto.GetDefaultAuxResourcePoolResponse, error) {
	return sproto.GetDefaultAuxResourcePoolResponse{
		PoolName: KubernetesDummyResourcePool,
	}, nil
}

// GetAgents gets the state of connected agents. Go around the RM and directly to the pods actor
// to avoid blocking through it.
func (k ResourceManager) GetAgents(
	ctx actor.Messenger,
	msg *apiv1.GetAgentsRequest,
) (resp *apiv1.GetAgentsResponse, err error) {
	return resp, actorrm.AskAt(k.Ref().System(), sproto.PodsAddr, msg, &resp)
}

// NotifyContainerRunning receives a notification from the container to let
// the master know that the container is running.
func (k ResourceManager) NotifyContainerRunning(
	ctx actor.Messenger,
	msg sproto.NotifyContainerRunning,
) error {
	// Kubernetes Resource Manager does not implement a handler for the
	// NotifyContainerRunning message, as it is only used on HPC
	// (High Performance Computing).
	return errors.New(
		"the NotifyContainerRunning message is unsupported for KubernetesResourceManager")
}

// kubernetesResourceProvider manages the lifecycle of k8s resources.
type kubernetesResourceManager struct {
	config    *config.KubernetesResourceManagerConfig
	podsActor *actor.Ref
	pools     map[string]*actor.Ref

	echoRef         *echo.Echo
	masterTLSConfig model.TLSClientConfig
	loggingConfig   model.LoggingConfig
}

func newKubernetesResourceManager(
	config *config.KubernetesResourceManagerConfig,
	echoRef *echo.Echo,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
) actor.Actor {
	return &kubernetesResourceManager{
		config: config,

		pools: make(map[string]*actor.Ref),

		echoRef:         echoRef,
		masterTLSConfig: masterTLSConfig,
		loggingConfig:   loggingConfig,
	}
}

func (k *kubernetesResourceManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		k.podsActor = Initialize(
			ctx.Self().System(),
			k.echoRef,
			ctx.Self(),
			k.config.Namespace,
			k.config.MasterServiceName,
			k.masterTLSConfig,
			k.loggingConfig,
			k.config.LeaveKubernetesResources,
			k.config.DefaultScheduler,
			k.config.SlotType,
			config.PodSlotResourceRequests{CPU: k.config.SlotResourceRequests.CPU},
			k.config.Fluent,
			k.config.CredsDir,
			k.config.MasterIP,
			k.config.MasterPort,
		)

		k.pools[KubernetesDummyResourcePool] = ctx.Self().System().MustActorOf(
			actor.Addr(KubernetesDummyResourcePool),
			&kubernetesResourcePool{
				config:            k.config,
				reqList:           tasklist.New(),
				groups:            map[*actor.Ref]*tasklist.Group{},
				addrToContainerID: map[*actor.Ref]cproto.ID{},
				containerIDtoAddr: map[string]*actor.Ref{},
				jobIDtoAddr:       map[model.JobID]*actor.Ref{},
				addrToJobID:       map[*actor.Ref]model.JobID{},
				groupActorToID:    map[*actor.Ref]model.JobID{},
				IDToGroupActor:    map[model.JobID]*actor.Ref{},
				slotsUsedPerGroup: map[*tasklist.Group]int{},
				podsActor:         k.podsActor,
				queuePositions:    tasklist.InitializeJobSortState(true),
			},
		)

	case
		tasklist.GroupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetAllocationName,
		sproto.AllocateRequest,
		sproto.ResourcesReleased,
		sproto.UpdatePodStatus,
		sproto.PendingPreemption,
		sproto.GetJobQ,
		sproto.GetJobQStats,
		sproto.SetGroupWeight,
		sproto.SetGroupPriority,
		sproto.MoveJob,
		sproto.DeleteJob,
		sproto.RecoverJobPosition,
		*apiv1.GetJobQueueStatsRequest,
		sproto.GetAllocationHandler,
		sproto.GetAllocationSummary,
		sproto.GetAllocationSummaries:
		k.forwardToPool(ctx, KubernetesDummyResourcePool, msg)

	case *apiv1.GetResourcePoolsRequest:
		summary := ctx.Ask(k.pools[KubernetesDummyResourcePool], msg).Get().(*resourcepoolv1.ResourcePool)
		resp := &apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{summary},
		}
		ctx.Respond(resp)

	case sproto.ValidateCommandResourcesRequest:
		fulfillable := k.config.MaxSlotsPerPod >= msg.Slots
		ctx.Respond(sproto.ValidateCommandResourcesResponse{Fulfillable: fulfillable})

	case *apiv1.GetAgentsRequest:
		resp := ctx.Ask(k.podsActor, msg)
		ctx.Respond(resp.Get())

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourceManager) forwardToPool(
	ctx *actor.Context, resourcePool string, msg actor.Message,
) {
	if k.pools[resourcePool] == nil {
		sender := "unknown"
		if ctx.Sender() != nil {
			sender = ctx.Sender().Address().String()
		}
		err := errors.Errorf("cannot find resource pool %s for message %T from actor %s",
			resourcePool, ctx.Message(), sender)
		ctx.Log().WithError(err).Error("")
		if ctx.ExpectingResponse() {
			ctx.Respond(err)
		}
		return
	}

	if ctx.ExpectingResponse() {
		response := ctx.Ask(k.pools[resourcePool], msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(k.pools[resourcePool], msg)
	}
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

type k8sPodResources struct {
	req             *sproto.AllocateRequest
	podsActor       *actor.Ref
	group           *tasklist.Group
	containerID     cproto.ID
	slots           int
	initialPosition decimal.Decimal
}

// Summary summarizes a container allocation.
func (p k8sPodResources) Summary() sproto.ResourcesSummary {
	return sproto.ResourcesSummary{
		AllocationID:  p.req.AllocationID,
		ResourcesID:   sproto.ResourcesID(p.containerID),
		ResourcesType: sproto.ResourcesTypeK8sPod,
		AgentDevices: map[aproto.ID][]device.Device{
			// TODO: Make it more obvious k8s can't be trusted.
			aproto.ID(p.podsActor.Address().Local()): nil,
		},

		ContainerID: &p.containerID,
	}
}

// Start notifies the pods actor that it should launch a pod for the provided task spec.
func (p k8sPodResources) Start(
	ctx *actor.Context, logCtx logger.Context, spec tasks.TaskSpec, rri sproto.ResourcesRuntimeInfo,
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
	return ctx.Ask(p.podsActor, StartTaskPod{
		TaskActor:  p.req.AllocationRef,
		Spec:       spec,
		Slots:      p.slots,
		Rank:       rri.AgentRank,
		LogContext: logCtx,
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
func (p k8sPodResources) Kill(ctx *actor.Context, _ logger.Context) {
	ctx.Tell(p.podsActor, KillTaskPod{
		PodID: p.containerID,
	})
}

func (p k8sPodResources) Persist() error {
	return nil
}

// TaskContainerDefaults returns TaskContainerDefaults for the specified pool.
func (k ResourceManager) TaskContainerDefaults(
	ctx actor.Messenger,
	pool string,
	fallbackConfig model.TaskContainerDefaultsConfig,
) (model.TaskContainerDefaultsConfig, error) {
	return fallbackConfig, nil
}
