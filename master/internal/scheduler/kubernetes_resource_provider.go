package scheduler

import (
	"crypto/tls"

	"github.com/google/uuid"

	cproto "github.com/determined-ai/determined/master/pkg/container"

	"github.com/determined-ai/determined/master/internal/kubernetes"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/model"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// kubernetesResourceProvider manages the lifecycle of k8s resources.
type kubernetesResourceProvider struct {
	clusterID             string
	config                *KubernetesResourceProviderConfig
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
	masterCert            *tls.Certificate

	reqList           *assignRequestList
	groups            map[*actor.Ref]*group
	slotsUsedPerGroup map[*group]int

	// Represent all pods as a single agent.
	agent *agentState

	reschedule bool
}

// NewKubernetesResourceProvider initializes a new kubernetesResourceProvider.
func NewKubernetesResourceProvider(
	clusterID string,
	config *KubernetesResourceProviderConfig,
	harnessPath string,
	taskContainerDefaults model.TaskContainerDefaultsConfig,
	masterCert *tls.Certificate,
) actor.Actor {
	return &kubernetesResourceProvider{
		clusterID:             clusterID,
		config:                config,
		harnessPath:           harnessPath,
		taskContainerDefaults: taskContainerDefaults,
		masterCert:            masterCert,

		reqList:           newAssignRequestList(),
		groups:            make(map[*actor.Ref]*group),
		slotsUsedPerGroup: make(map[*group]int),
	}
}

func (k *kubernetesResourceProvider) Receive(ctx *actor.Context) error {
	reschedule := true
	defer func() {
		// Default to scheduling every 500ms if a message was received, but allow messages
		// that don't affect the cluster to be skipped.
		k.reschedule = k.reschedule || reschedule
	}()

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	case sproto.ConfigureEndpoints:
		ctx.Log().Infof("initializing endpoints for pods")
		podsActor := kubernetes.Initialize(
			msg.System,
			msg.Echo,
			ctx.Self(),
			k.clusterID,
			k.config.Namespace,
			k.config.MasterServiceName,
			k.config.LeaveKubernetesResources,
		)
		k.agent = newAgentState(sproto.AddAgent{Agent: podsActor})

	case
		groupActorStopped,
		SetMaxSlots,
		SetWeight,
		AssignRequest,
		ResourceReleased:
		return k.receiveRequestMsg(ctx)

	case GetTaskSummary:
		if resp := getTaskSummary(k.reqList, *msg.ID); resp != nil {
			ctx.Respond(*resp)
		}
		reschedule = false

	case GetTaskSummaries:
		reschedule = false
		ctx.Respond(getTaskSummaries(k.reqList))

	case sproto.GetEndpointActorAddress:
		reschedule = false
		ctx.Respond("/pods")

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

func (k *kubernetesResourceProvider) receiveRequestMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case groupActorStopped:
		delete(k.slotsUsedPerGroup, k.groups[msg.Ref])
		delete(k.groups, msg.Ref)

	case SetMaxSlots:
		k.getOrCreateGroup(ctx, msg.Handler).maxSlots = msg.MaxSlots

	case SetWeight:
		// SetWeight is not supported by the Kubernetes RP.

	case AssignRequest:
		k.addAssignRequest(ctx, msg)

	case ResourceReleased:
		k.resourcesReleased(ctx, msg.Handler)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (k *kubernetesResourceProvider) addAssignRequest(ctx *actor.Context, msg AssignRequest) {
	actors.NotifyOnStop(ctx, msg.Handler, ResourceReleased{Handler: msg.Handler})

	if len(msg.ID) == 0 {
		msg.ID = RequestID(uuid.New().String())
	}
	if msg.Group == nil {
		msg.Group = msg.Handler
	}
	k.getOrCreateGroup(ctx, msg.Group)
	if len(msg.Name) == 0 {
		msg.Name = "Unnamed-k8-Task"
	}

	ctx.Log().Infof(
		"resources are requested by %s (request ID: %s)",
		msg.Handler.Address(), msg.ID,
	)
	k.reqList.Add(&msg)
}

func (k *kubernetesResourceProvider) assignResources(ctx *actor.Context, req *AssignRequest) {
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

	assignments := make([]Assignment, 0, numPods)
	for pod := 0; pod < numPods; pod++ {
		container := newContainer(req, k.agent, slotsPerPod, len(assignments))
		assignments = append(assignments, &podAssignment{
			req:         req,
			agent:       k.agent,
			container:   container,
			clusterID:   k.clusterID,
			harnessPath: k.harnessPath,

			taskContainerDefaults: k.taskContainerDefaults,
			masterCert:            k.masterCert,
		})
	}

	assigned := ResourceAssigned{Assignments: assignments}
	k.reqList.SetAssignments(req.Handler, &assigned)
	req.Handler.System().Tell(req.Handler, assigned)

	ctx.Log().
		WithField("task-id", req.ID).
		WithField("task-handler", req.Handler.Address()).
		Infof("resources assigned with %d pods", numPods)
}

func (k *kubernetesResourceProvider) resourcesReleased(ctx *actor.Context, handler *actor.Ref) {
	ctx.Log().Infof("resources are released for %s", handler.Address())
	k.reqList.Remove(handler)

	if req, ok := k.reqList.Get(handler); ok {
		group := k.groups[handler]

		if group != nil {
			k.slotsUsedPerGroup[group] -= req.SlotsNeeded
		}
	}
}

func (k *kubernetesResourceProvider) getOrCreateGroup(
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

func (k *kubernetesResourceProvider) schedulePendingTasks(ctx *actor.Context) {
	for it := k.reqList.iterator(); it.next(); {
		req := it.value()
		group := k.groups[req.Group]
		assigned := k.reqList.GetAssignments(req.Handler)
		if assigned == nil || len(assigned.Assignments) == 0 {
			if maxSlots := group.maxSlots; maxSlots != nil {
				if k.slotsUsedPerGroup[group]+req.SlotsNeeded > *maxSlots {
					continue
				}
			}

			k.assignResources(ctx, req)
		}
	}
}

type podAssignment struct {
	req                   *AssignRequest
	container             *container
	agent                 *agentState
	clusterID             string
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
	masterCert            *tls.Certificate
}

// Summary summerizes a container assignment.
func (p podAssignment) Summary() ContainerSummary {
	return ContainerSummary{
		TaskID: p.req.ID,
		ID:     p.container.id,
		Agent:  p.agent.handler.Address().Local(),
	}
}

// Start notifies the pods actor that it should launch a pod for the provided task spec.
func (p podAssignment) StartContainer(ctx *actor.Context, spec image.TaskSpec) {
	handler := p.agent.handler
	spec.ClusterID = p.clusterID
	spec.ContainerID = string(p.container.ID())
	spec.TaskID = string(p.req.ID)
	spec.HarnessPath = p.harnessPath
	spec.TaskContainerDefaults = p.taskContainerDefaults
	spec.MasterCert = p.masterCert
	ctx.Tell(handler, sproto.StartPod{
		TaskHandler: p.req.Handler,
		Spec:        spec,
		Slots:       p.container.Slots(),
		Rank:        p.container.ordinal,
	})
}

// Kill notifies the pods actor that it should stop the pod.
func (p podAssignment) KillContainer(ctx *actor.Context) {
	handler := p.agent.handler
	ctx.Tell(handler, sproto.KillContainer{
		ContainerID: cproto.ID(p.container.ID()),
	})
}
