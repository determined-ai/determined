package scheduler

import (
	"github.com/determined-ai/determined/master/internal/kubernetes"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/model"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// kubernetesResourceProvider manages the lifecycle of k8 resources.
type kubernetesResourceProvider struct {
	clusterID             string
	namespace             string
	slotsPerNode          int
	masterServiceName     string
	proxy                 *actor.Ref
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig

	tasksByHandler     map[*actor.Ref]*Task
	tasksByID          map[TaskID]*Task
	tasksByContainerID map[ContainerID]*Task
	groups             map[*actor.Ref]*group

	assigmentByTaskHandler map[*actor.Ref][]podAssignment

	// Represent all pods as a single agent.
	agent *agentState
}

// NewKubernetesResourceProvider initializes a new kubernetesResourceProvider.
func NewKubernetesResourceProvider(
	clusterID string,
	namespace string,
	slotsPerNode int,
	masterServiceName string,
	proxy *actor.Ref,
	harnessPath string,
	taskContainerDefaults model.TaskContainerDefaultsConfig,
) actor.Actor {
	return &kubernetesResourceProvider{
		clusterID:             clusterID,
		namespace:             namespace,
		slotsPerNode:          slotsPerNode,
		masterServiceName:     masterServiceName,
		proxy:                 proxy,
		harnessPath:           harnessPath,
		taskContainerDefaults: taskContainerDefaults,

		tasksByHandler:     make(map[*actor.Ref]*Task),
		tasksByID:          make(map[TaskID]*Task),
		tasksByContainerID: make(map[ContainerID]*Task),
		groups:             make(map[*actor.Ref]*group),

		assigmentByTaskHandler: make(map[*actor.Ref][]podAssignment),
	}
}

func (k *kubernetesResourceProvider) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:

	case sproto.ConfigureEndpoints:
		ctx.Log().Infof("initializing endpoints for pods")
		podsActor := kubernetes.Initialize(
			msg.System,
			msg.Echo,
			ctx.Self(),
			k.namespace,
			k.masterServiceName,
		)

		k.agent = newAgentState(sproto.AddAgent{Agent: podsActor})

	case AddTask:
		k.receiveAddTask(ctx, msg)

	case SetMaxSlots, SetWeight:

	case SetTaskName:
		k.receiveSetTaskName(ctx, msg)

	case StartTask:
		k.receiveStartTask(ctx, msg)

	default:
		ctx.Log().Errorf("Unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourceProvider) receiveAddTask(ctx *actor.Context, msg AddTask) {
	actors.NotifyOnStop(ctx, msg.TaskHandler, taskStopped{Ref: msg.TaskHandler})

	if task, ok := k.tasksByHandler[ctx.Sender()]; ok {
		if ctx.ExpectingResponse() {
			ctx.Respond(task)
		}
		return
	}

	if msg.Group == nil {
		msg.Group = msg.TaskHandler
	}
	group := k.getOrCreateGroup(msg.Group, ctx)

	var taskID TaskID
	if msg.ID != nil {
		taskID = *msg.ID
	}

	name := msg.Name
	if len(name) == 0 {
		name = "Unnamed-k8-Task"
	}

	task := newTask(&Task{
		ID:                  taskID,
		group:               group,
		handler:             msg.TaskHandler,
		name:                name,
		slotsNeeded:         msg.SlotsNeeded,
		canTerminate:        msg.CanTerminate,
		agentLabel:          msg.Label,
		fittingRequirements: msg.FittingRequirements,
	})

	k.tasksByID[task.ID] = task
	k.tasksByHandler[task.handler] = task

	if ctx.ExpectingResponse() {
		ctx.Respond(task)
	}

	k.scheduleTask(ctx, task)
}

func (k *kubernetesResourceProvider) scheduleTask(ctx *actor.Context, task *Task) {
	numPods := 1
	slotsPerNode := task.SlotsNeeded()
	if task.SlotsNeeded() > 1 {
		if k.slotsPerNode == 0 {
			ctx.Log().WithField("task ID", task.ID).Error(
				"set slots_per_node > 0 to schedule tasks with slots")
			return
		}

		if task.SlotsNeeded()%k.slotsPerNode != 0 {
			ctx.Log().WithField("task ID", task.ID).Error(
				"task number of slots is not schedulable on the configured slots_per_node")
			return
		}
		numPods = task.SlotsNeeded() / k.slotsPerNode
		slotsPerNode = k.slotsPerNode
	}

	for pod := 0; pod < numPods; pod++ {
		k.assignPod(ctx, task, slotsPerNode)
	}

	ctx.Log().WithField("task ID", task.ID).Infof("task assigned by scheduler")
	task.handler.System().Tell(task.handler, TaskAssigned{NumContainers: numPods})
}

func (k *kubernetesResourceProvider) assignPod(ctx *actor.Context, task *Task, slots int) {
	if task.state != taskRunning {
		task.mustTransition(taskRunning)
	}
	container := newContainer(task, k.agent, slots, len(task.containers))
	k.agent.containers[container.id] = container
	task.containers[container.id] = container
	k.tasksByContainerID[container.id] = task
	k.assigmentByTaskHandler[task.handler] = append(
		k.assigmentByTaskHandler[task.handler],
		podAssignment{
			task:        task,
			agent:       k.agent,
			container:   container,
			clusterID:   k.clusterID,
			harnessPath: k.harnessPath,

			taskContainerDefaults: k.taskContainerDefaults,
		})
}

func (k *kubernetesResourceProvider) getOrCreateGroup(
	handler *actor.Ref,
	ctx *actor.Context,
) *group {
	if g, ok := k.groups[handler]; ok {
		return g
	}
	g := &group{handler: handler, weight: 1}
	k.groups[handler] = g
	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, groupStopped{})
	}
	return g
}

func (k *kubernetesResourceProvider) receiveSetTaskName(ctx *actor.Context, msg SetTaskName) {
	if task, ok := k.tasksByHandler[msg.TaskHandler]; ok {
		task.name = msg.Name
	}
}

func (k *kubernetesResourceProvider) receiveStartTask(ctx *actor.Context, msg StartTask) {
	task := k.tasksByHandler[msg.TaskHandler]
	if task == nil {
		ctx.Log().WithField("address", msg.TaskHandler.Address()).Errorf("unknown task trying to start")
		return
	}

	assignments := k.assigmentByTaskHandler[msg.TaskHandler]
	if len(assignments) == 0 {
		ctx.Log().WithField("name", task.name).Error("task is trying to start without any assignments")
		return
	}

	for _, a := range assignments {
		a.StartTask(msg.Spec)
	}
}

type podAssignment struct {
	task                  *Task
	container             *container
	agent                 *agentState
	clusterID             string
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
}

// StartTask notifies the pods actor that it should launch a pod for the provided task spec.
func (p *podAssignment) StartTask(spec image.TaskSpec) {
	handler := p.agent.handler
	spec.ClusterID = p.clusterID
	spec.ContainerID = string(p.container.ID())
	spec.TaskID = string(p.task.ID)
	spec.HarnessPath = p.harnessPath
	spec.TaskContainerDefaults = p.taskContainerDefaults
	handler.System().Tell(handler, sproto.StartPod{
		Task:  p.task.handler,
		Spec:  spec,
		Slots: p.container.Slots(),
		Rank:  p.container.ordinal,
	})
}
