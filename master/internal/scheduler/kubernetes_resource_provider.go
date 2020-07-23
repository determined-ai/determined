package scheduler

import (
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
	proxy                 *actor.Ref
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig

	tasksByHandler     map[*actor.Ref]*Task
	tasksByID          map[TaskID]*Task
	tasksByContainerID map[ContainerID]*Task
	groups             map[*actor.Ref]*group

	assignmentsByTaskHandler map[*actor.Ref][]podAssignment

	// Represent all pods as a single agent.
	agent *agentState
}

// NewKubernetesResourceProvider initializes a new kubernetesResourceProvider.
func NewKubernetesResourceProvider(
	clusterID string,
	config *KubernetesResourceProviderConfig,
	proxy *actor.Ref,
	harnessPath string,
	taskContainerDefaults model.TaskContainerDefaultsConfig,
) actor.Actor {
	return &kubernetesResourceProvider{
		clusterID:             clusterID,
		config:                config,
		proxy:                 proxy,
		harnessPath:           harnessPath,
		taskContainerDefaults: taskContainerDefaults,

		tasksByHandler:     make(map[*actor.Ref]*Task),
		tasksByID:          make(map[TaskID]*Task),
		tasksByContainerID: make(map[ContainerID]*Task),
		groups:             make(map[*actor.Ref]*group),

		assignmentsByTaskHandler: make(map[*actor.Ref][]podAssignment),
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
			k.config.Namespace,
			k.config.MasterServiceName,
			k.config.LeaveKubernetesResources,
		)

		k.agent = newAgentState(sproto.AddAgent{Agent: podsActor})

	case AddTask:
		k.receiveAddTask(ctx, msg)

	case SetMaxSlots, SetWeight:
		// These parameters are not supported by the Kubernetes RP.

	case SetTaskName:
		k.receiveSetTaskName(ctx, msg)

	case StartTask:
		k.receiveStartTask(ctx, msg)

	case sproto.PodStarted:
		k.receivePodStarted(ctx, msg)

	case sproto.PodTerminated:
		k.receivePodTerminated(ctx, msg, false)

	case taskStopped:
		k.receiveTaskStopped(ctx, msg)

	case groupStopped:

	case TerminateTask:
		k.receiveTerminateTask(ctx, msg)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (k *kubernetesResourceProvider) receiveAddTask(ctx *actor.Context, msg AddTask) {
	actors.NotifyOnStop(ctx, msg.TaskHandler, taskStopped{Ref: msg.TaskHandler})

	if task, ok := k.tasksByHandler[msg.TaskHandler]; ok {
		if ctx.ExpectingResponse() {
			ctx.Respond(task)
		}
		return
	}

	if msg.Group == nil {
		msg.Group = msg.TaskHandler
	}
	group := k.getOrCreateGroup(ctx, msg.Group)

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
		if k.config.SlotsPerNode == 0 {
			ctx.Log().WithField("task-id", task.ID).Error(
				"set slots_per_node > 0 to schedule tasks with slots")
			return
		}

		if task.SlotsNeeded()%k.config.SlotsPerNode != 0 {
			ctx.Log().WithField("task-id", task.ID).Error(
				"task number of slots (%d) is not schedulable on the configured "+
					"slots_per_node (%d)", task.SlotsNeeded(), k.config.SlotsPerNode)
			return
		}
		numPods = task.SlotsNeeded() / k.config.SlotsPerNode
		slotsPerNode = k.config.SlotsPerNode
	}

	for pod := 0; pod < numPods; pod++ {
		k.assignPod(ctx, task, slotsPerNode)
	}

	ctx.Log().WithField("task-id", task.ID).Infof(
		"task assigned by scheduler with %d pods", numPods)
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
	k.assignmentsByTaskHandler[task.handler] = append(
		k.assignmentsByTaskHandler[task.handler],
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
	ctx *actor.Context,
	handler *actor.Ref,
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
		ctx.Log().WithField("address", msg.TaskHandler.Address()).Error("unknown task trying to start")
		return
	}

	assignments := k.assignmentsByTaskHandler[msg.TaskHandler]
	if len(assignments) == 0 {
		ctx.Log().WithField("name", task.name).Error("task is trying to start without any assignments")
		return
	}

	for _, a := range assignments {
		a.StartTask(msg.Spec)
	}
	delete(k.assignmentsByTaskHandler, msg.TaskHandler)
}

func (k *kubernetesResourceProvider) receivePodStarted(ctx *actor.Context, msg sproto.PodStarted) {
	task, ok := k.tasksByContainerID[ContainerID(msg.ContainerID)]
	if !ok {
		ctx.Log().Warnf("received pod start from unknown container %s", msg.ContainerID)
	}

	container := task.containers[ContainerID(msg.ContainerID)]
	container.addresses = constructAddresses(msg.IP, msg.Ports)
	container.mustTransition(containerRunning)
	handler := container.task.handler
	handler.System().Tell(handler, ContainerStarted{Container: container})

	// TODO (DET-3422): add in proxying initialization.
}

func (k *kubernetesResourceProvider) receivePodTerminated(
	ctx *actor.Context,
	msg sproto.PodTerminated,
	aborted bool,
) {
	cid := ContainerID(msg.ContainerID)
	task := k.tasksByContainerID[cid]
	if task == nil {
		ctx.Log().WithField("container-id", cid).Info(
			"ignoring stale terminated message for container",
		)
		return
	}

	container := task.containers[cid]
	container.mustTransition(containerTerminated)
	container.exitStatus = msg.ContainerStopped

	// TODO(DET-3422): de-register proxying info.

	delete(container.agent.containers, container.id)
	delete(container.task.containers, container.id)
	delete(k.tasksByContainerID, container.id)

	// A task is terminated if and only if all of its containers are terminated.
	for _, container := range task.containers {
		if container.state != containerTerminated {
			return
		}
	}

	if task.state != taskTerminated {
		k.taskTerminated(task, aborted)
	}
}

func (k *kubernetesResourceProvider) taskTerminated(task *Task, aborted bool) {
	task.mustTransition(taskTerminated)

	delete(k.tasksByID, task.ID)
	delete(k.tasksByHandler, task.handler)

	for id := range task.containers {
		delete(k.tasksByContainerID, id)
	}

	task.handler.System().Tell(task.handler, TaskTerminated{})
	// This is somewhat redundant with the message above, but we're transitioning between them.
	if aborted {
		task.handler.System().Tell(task.handler, TaskAborted{})
	}
}

func (k *kubernetesResourceProvider) receiveTaskStopped(ctx *actor.Context, msg taskStopped) {
	task := k.tasksByHandler[msg.Ref]
	if task == nil {
		return
	}

	// Clean up a task even if it does not have any containers yet.
	if task.state != taskTerminated {
		ctx.Log().WithField("task", task.ID).Warnf("task stopped without terminating")
		k.taskTerminated(task, true)
	}
}

func (k *kubernetesResourceProvider) receiveTerminateTask(ctx *actor.Context, msg TerminateTask) {
	task := k.tasksByID[msg.TaskID]
	if task == nil {
		if ctx.ExpectingResponse() {
			ctx.Respond(task)
		}
		return
	}

	k.terminateTask(task, msg.Forcible)

	if ctx.ExpectingResponse() {
		ctx.Respond(task)
	}
}

// terminateTask sends the appropriate actor messages to terminate a task and
// deallocate its cluster data structures. The task may not be terminated if it
// is in the right state unless forcible is true.
func (k *kubernetesResourceProvider) terminateTask(task *Task, forcible bool) {
	switch {
	case task.state == taskTerminated:
		// The task has already been terminated so this is a noop.

	case len(task.containers) == 0 || task.state == taskPending:
		// The task is not running so there is no need to request the task to terminate. The task is
		// marked as aborted.
		k.taskTerminated(task, true)

	case forcible:
		// Notify the agent to kill the task.
		task.mustTransition(taskTerminating)
		for _, c := range task.containers {
			if c.state != containerTerminated {
				c.mustTransition(containerTerminating)
			}
			c.agent.handler.System().Tell(
				c.agent.handler, sproto.StopPod{ContainerID: string(c.id)})
		}

	case task.state != taskTerminating && task.canTerminate:
		// Notify the running task that it should shut down gracefully.
		task.mustTransition(taskTerminating)
		for _, c := range task.containers {
			if c.state != containerTerminated {
				c.mustTransition(containerTerminating)
			}
		}
		task.handler.System().Tell(task.handler, TerminateRequest{})
	}
}

func constructAddresses(ip string, ports []int) []Address {
	addresses := make([]Address, 0, len(ports))
	for _, port := range ports {
		addresses = append(addresses, Address{
			ContainerIP:   ip,
			ContainerPort: port,
			HostIP:        ip,
			HostPort:      port,
		})
	}

	return addresses
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
		TaskHandler: p.task.handler,
		Spec:        spec,
		Slots:       p.container.Slots(),
		Rank:        p.container.ordinal,
	})
}
