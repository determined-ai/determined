package resourcemanagers

import (
	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/check"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	image "github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const kubernetesScheduler = "kubernetes"
const kubernetesDummyResourcePool = "kubernetes"

// kubernetesResourceProvider manages the lifecycle of k8s resources.
type kubernetesResourceManager struct {
	config *KubernetesResourceManagerConfig

	reqList           *taskList
	groups            map[*actor.Ref]*group
	slotsUsedPerGroup map[*group]int

	// Represent all pods as a single agent.
	agent *agentState

	reschedule bool
}

func newKubernetesResourceManager(
	config *KubernetesResourceManagerConfig,
) actor.Actor {
	return &kubernetesResourceManager{
		config: config,

		reqList:           newTaskList(),
		groups:            make(map[*actor.Ref]*group),
		slotsUsedPerGroup: make(map[*group]int),
	}
}

func (k *kubernetesResourceManager) Receive(ctx *actor.Context) error {
	reschedule := true
	defer func() {
		// Default to scheduling every 500ms if a message was received, but allow messages
		// that don't affect the cluster to be skipped.
		k.reschedule = k.reschedule || reschedule
	}()

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	case sproto.SetPods:
		check.Panic(check.True(k.agent == nil, "should only set pods once"))
		k.agent = &agentState{
			handler:            msg.Pods,
			devices:            make(map[device.Device]*cproto.ID),
			zeroSlotContainers: make(map[cproto.ID]bool),
		}

	case
		groupActorStopped,
		sproto.SetGroupMaxSlots,
		sproto.SetGroupWeight,
		sproto.SetGroupPriority,
		sproto.SetTaskName,
		sproto.AllocateRequest,
		sproto.ResourcesReleased:
		return k.receiveRequestMsg(ctx)

	case sproto.GetTaskSummary:
		if resp := getTaskSummary(k.reqList, *msg.ID, k.groups, kubernetesScheduler); resp != nil {
			ctx.Respond(*resp)
		}
		reschedule = false

	case sproto.GetTaskSummaries:
		reschedule = false
		ctx.Respond(getTaskSummaries(k.reqList, k.groups, kubernetesScheduler))

	case *apiv1.GetResourcePoolsRequest:
		resourcePoolSummary := k.summarizeDummyResourcePool(ctx)
		resp := &apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{resourcePoolSummary},
		}
		ctx.Respond(resp)

	case sproto.GetDefaultGPUResourcePoolRequest:
		ctx.Respond(sproto.GetDefaultGPUResourcePoolResponse{PoolName: "default"})

	case sproto.GetDefaultCPUResourcePoolRequest:
		ctx.Respond(sproto.GetDefaultCPUResourcePoolResponse{PoolName: "default"})

	case schedulerTick:
		if k.reschedule {
			k.schedulePendingTasks(ctx)
		}
		k.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	default:
		reschedule = false
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourceManager) summarizeDummyResourcePool(
	ctx *actor.Context,
) *resourcepoolv1.ResourcePool {
	slotsUsed := 0
	for _, slotsUsedByGroup := range k.slotsUsedPerGroup {
		slotsUsed += slotsUsedByGroup
	}
	return &resourcepoolv1.ResourcePool{
		Name:                         kubernetesDummyResourcePool,
		Description:                  "Kubernetes-managed pool of resources",
		Type:                         resourcepoolv1.ResourcePoolType_RESOURCE_POOL_TYPE_K8S,
		NumAgents:                    1,
		NumSlots:               int32(k.agent.numSlots()),
		NumSlotsInUse:                    int32(k.agent.numUsedSlots()),
		CpuContainerCapacity:         int32(k.agent.maxZeroSlotContainers),
		CpuContainersRunning:         int32(k.agent.numZeroSlotContainers()),
		DefaultGpuPool:               true,
		DefaultCpuPool:               true,
		Preemptible:                  false,
		MinAgents:                    0,
		MaxAgents:                    0,
		CpuContainerCapacityPerAgent: 0,
		SchedulerType:                resourcepoolv1.SchedulerType_SCHEDULER_TYPE_KUBERNETES,
		SchedulerFittingPolicy:       resourcepoolv1.FittingPolicy_FITTING_POLICY_KUBERNETES,
		Location:                     "kubernetes",
		ImageId:                      "",
		InstanceType:                 "kubernetes",
		Details:                      nil,
	}
}

func (k *kubernetesResourceManager) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case groupActorStopped:
		delete(k.slotsUsedPerGroup, k.groups[msg.Ref])
		delete(k.groups, msg.Ref)

	case sproto.SetGroupMaxSlots:
		k.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

	case sproto.SetGroupWeight, sproto.SetGroupPriority:
		// SetGroupWeight and SetGroupPriority are not supported by the Kubernetes RP.

	case sproto.SetTaskName:
		k.receiveSetTaskName(ctx, msg)

	case sproto.AllocateRequest:
		k.addTask(ctx, msg)

	case sproto.ResourcesReleased:
		k.resourcesReleased(ctx, msg.TaskActor)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (k *kubernetesResourceManager) addTask(ctx *actor.Context, msg sproto.AllocateRequest) {
	actors.NotifyOnStop(ctx, msg.TaskActor, sproto.ResourcesReleased{TaskActor: msg.TaskActor})

	if len(msg.ID) == 0 {
		msg.ID = sproto.TaskID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.TaskActor
	}
	k.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed-k8-Task"
	}

	ctx.Log().Infof(
		"resources are requested by %s (Task ID: %s)",
		msg.TaskActor.Address(), msg.ID,
	)
	k.reqList.AddTask(&msg)
}

func (k *kubernetesResourceManager) receiveSetTaskName(ctx *actor.Context, msg sproto.SetTaskName) {
	if task, found := k.reqList.GetTaskByHandler(msg.TaskHandler); found {
		task.Name = msg.Name
	}
}

func (k *kubernetesResourceManager) assignResources(
	ctx *actor.Context, req *sproto.AllocateRequest,
) {
	numPods := 1
	slotsPerPod := req.SlotsNeeded
	if req.SlotsNeeded > 1 {
		if k.config.MaxSlotsPerPod == 0 {
			ctx.Log().WithField("task-id", req.ID).Error(
				"set max_slots_per_pod > 0 to schedule tasks with slots")
			return
		}

		if req.SlotsNeeded <= k.config.MaxSlotsPerPod {
			numPods = 1
			slotsPerPod = req.SlotsNeeded
		} else {
			if req.SlotsNeeded%k.config.MaxSlotsPerPod != 0 {
				ctx.Log().WithField("task-id", req.ID).Errorf(
					"task number of slots (%d) is not schedulable on the configured "+
						"max_slots_per_pod (%d)", req.SlotsNeeded, k.config.MaxSlotsPerPod)
				return
			}

			numPods = req.SlotsNeeded / k.config.MaxSlotsPerPod
			slotsPerPod = k.config.MaxSlotsPerPod
		}
	}

	k.slotsUsedPerGroup[k.groups[req.Group]] += req.SlotsNeeded

	allocations := make([]sproto.Allocation, 0, numPods)
	for pod := 0; pod < numPods; pod++ {
		container := newContainer(req, k.agent, slotsPerPod)
		allocations = append(allocations, &podAllocation{
			req:       req,
			agent:     k.agent,
			container: container,
		})
	}

	assigned := sproto.ResourcesAllocated{ID: req.ID, Allocations: allocations}
	k.reqList.SetAllocations(req.TaskActor, &assigned)
	req.TaskActor.System().Tell(req.TaskActor, assigned)

	ctx.Log().
		WithField("task-id", req.ID).
		WithField("task-handler", req.TaskActor.Address()).
		Infof("resources assigned with %d pods", numPods)
}

func (k *kubernetesResourceManager) resourcesReleased(ctx *actor.Context, handler *actor.Ref) {
	ctx.Log().Infof("resources are released for %s", handler.Address())
	k.reqList.RemoveTaskByHandler(handler)

	if req, ok := k.reqList.GetTaskByHandler(handler); ok {
		group := k.groups[handler]

		if group != nil {
			k.slotsUsedPerGroup[group] -= req.SlotsNeeded
		}
	}
}

func (k *kubernetesResourceManager) getOrCreateGroup(
	ctx *actor.Context,
	handler *actor.Ref,
) *group {
	if g, ok := k.groups[handler]; ok {
		return g
	}
	g := &group{handler: handler, weight: 1}
	k.groups[handler] = g
	k.slotsUsedPerGroup[g] = 0

	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, groupActorStopped{})
	}
	return g
}

func (k *kubernetesResourceManager) schedulePendingTasks(ctx *actor.Context) {
	for it := k.reqList.iterator(); it.next(); {
		req := it.value()
		group := k.groups[req.Group]
		assigned := k.reqList.GetAllocations(req.TaskActor)
		if unassigned := assigned == nil || len(assigned.Allocations) == 0; unassigned {
			if maxSlots := group.maxSlots; maxSlots != nil {
				if k.slotsUsedPerGroup[group]+req.SlotsNeeded > *maxSlots {
					continue
				}
			}

			k.assignResources(ctx, req)
		}
	}
}

type podAllocation struct {
	req       *sproto.AllocateRequest
	container *container
	agent     *agentState
}

// Summary summarizes a container allocation.
func (p podAllocation) Summary() sproto.ContainerSummary {
	return sproto.ContainerSummary{
		TaskID: p.req.ID,
		ID:     p.container.id,
		Agent:  p.agent.handler.Address().Local(),
	}
}

// Start notifies the pods actor that it should launch a pod for the provided task spec.
func (p podAllocation) Start(ctx *actor.Context, spec image.TaskSpec) {
	handler := p.agent.handler
	spec.ContainerID = string(p.container.id)
	spec.TaskID = string(p.req.ID)
	ctx.Tell(handler, sproto.StartTaskPod{
		TaskActor: p.req.TaskActor,
		Spec:      spec,
		Slots:     p.container.slots,
	})
}

// Kill notifies the pods actor that it should stop the pod.
func (p podAllocation) Kill(ctx *actor.Context) {
	handler := p.agent.handler
	ctx.Tell(handler, sproto.KillTaskPod{
		PodID: p.container.id,
	})
}
