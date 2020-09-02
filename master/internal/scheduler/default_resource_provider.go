package scheduler

import (
	"crypto/tls"
	"strconv"

	"github.com/determined-ai/determined/master/pkg/device"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/agent"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// DefaultRP manages the agent and task lifecycles.
type DefaultRP struct {
	clusterID             string
	scheduler             Scheduler
	fittingMethod         SoftConstraint
	agents                map[*actor.Ref]*agentState
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
	masterCert            *tls.Certificate

	taskList           *taskList
	tasksByHandler     map[*actor.Ref]*Task
	tasksByID          map[TaskID]*Task
	tasksByContainerID map[ContainerID]*Task
	groups             map[*actor.Ref]*group

	provisioner     *actor.Ref
	provisionerView *FilterableView

	saveNotifications bool
	notifications     []<-chan struct{}

	reschedule bool
}

// NewDefaultRP initializes a new empty default resource provider.
func NewDefaultRP(
	clusterID string,
	scheduler Scheduler,
	fittingMethod SoftConstraint,
	harnessPath string,
	taskContainerDefaults model.TaskContainerDefaultsConfig,
	provisioner *actor.Ref,
	provisionerSlotsPerInstance int,
	masterCert *tls.Certificate,
) actor.Actor {
	d := &DefaultRP{
		clusterID:             clusterID,
		scheduler:             scheduler,
		fittingMethod:         fittingMethod,
		agents:                make(map[*actor.Ref]*agentState),
		groups:                make(map[*actor.Ref]*group),
		harnessPath:           harnessPath,
		taskContainerDefaults: taskContainerDefaults,
		masterCert:            masterCert,

		taskList:           newTaskList(),
		tasksByHandler:     make(map[*actor.Ref]*Task),
		tasksByID:          make(map[TaskID]*Task),
		tasksByContainerID: make(map[ContainerID]*Task),

		provisioner:     provisioner,
		provisionerView: newProvisionerView(provisionerSlotsPerInstance),

		reschedule: false,
	}
	return d
}

func (d *DefaultRP) assignContainer(
	task *Task, a *agentState, slots int, numContainers int,
) *containerAssignment {
	if task.state != taskRunning {
		task.mustTransition(taskRunning)
	}
	container := newContainer(task, a, slots, len(task.containers))
	a.containers[container.id] = container
	task.containers[container.id] = container
	d.tasksByContainerID[container.id] = task

	return &containerAssignment{
		task:                  task,
		agent:                 a,
		container:             container,
		clusterID:             d.clusterID,
		devices:               a.assignFreeDevices(slots, container.id),
		harnessPath:           d.harnessPath,
		taskContainerDefaults: d.taskContainerDefaults,
		masterCert:            d.masterCert,
	}
}

// assignTask allocates cluster data structures and sends the appropriate actor
// messages to start a task if there are enough resources in the cluster to run
// the task. If there are not, assignTask returns false.
func (d *DefaultRP) assignTask(task *Task) bool {
	fits := findFits(task, d.agents, d.fittingMethod)

	if len(fits) == 0 {
		return false
	}

	assignments := make([]Assignment, 0, len(fits))

	for _, fit := range fits {
		assignments = append(
			assignments,
			d.assignContainer(task, fit.Agent, fit.Slots, len(fits)),
		)
	}

	task.handler.System().Tell(task.handler, ResourceAssigned{
		Assignments: assignments,
	})

	return true
}

// terminateTask sends the appropriate actor messages to terminate a task and
// deallocate its cluster data structures. The task may not be terminated if it
// is in the right state unless forcible is true.
func (d *DefaultRP) terminateTask(task *Task, forcible bool) {
	switch {
	case task.state == taskTerminated:
		// The task has already been terminated so this is a noop.

	case len(task.containers) == 0 || task.state == taskPending:
		// The task is not running so there is no need to request the task to terminate. The task is
		// marked as aborted.
		d.taskTerminated(task, true)

	case forcible:
		// Notify the agent to kill the task.
		task.mustTransition(taskTerminating)
		for _, c := range task.containers {
			if c.state != containerTerminated {
				c.mustTransition(containerTerminating)
			}
		}

	case task.state != taskTerminating && task.canTerminate:
		// Notify the running task that it should shut down gracefully.
		task.mustTransition(taskTerminating)
		for _, c := range task.containers {
			if c.state != containerTerminated {
				c.mustTransition(containerTerminating)
			}
		}
	}
}

func (d *DefaultRP) getOrCreateGroup(handler *actor.Ref, ctx *actor.Context) *group {
	if g, ok := d.groups[handler]; ok {
		return g
	}
	g := &group{handler: handler, weight: 1}
	d.groups[handler] = g
	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, groupActorStopped{})
	}
	return g
}

func (d *DefaultRP) getTaskSummary(id TaskID) *TaskSummary {
	if task := d.tasksByID[id]; task != nil {
		summary := newTaskSummary(task)
		return &summary
	}
	return nil
}

func (d *DefaultRP) notifyOnStop(ctx *actor.Context, ref *actor.Ref, msg actor.Message) {
	done := actors.NotifyOnStop(ctx, ref, msg)
	if d.saveNotifications {
		d.notifications = append(d.notifications, done)
	}
}

func (d *DefaultRP) sendProvisionerView(ctx *actor.Context) {
	if d.provisioner != nil {
		if snapshot, updateMade := d.provisionerView.Update(d); updateMade {
			ctx.Tell(d.provisioner, snapshot)
		}
	}
}

// Receive implements the actor.Actor interface.
func (d *DefaultRP) Receive(ctx *actor.Context) error {
	reschedule := true
	defer func() {
		// Default to scheduling every 500ms if a message was received, but allow messages
		// that don't affect the cluster to be skipped.
		d.reschedule = d.reschedule || reschedule
	}()

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	case
		sproto.ConfigureEndpoints,
		sproto.AddAgent,
		sproto.AddDevice,
		sproto.FreeDevice,
		sproto.RemoveDevice,
		sproto.RemoveAgent,
		sproto.TaskStartedOnAgent,
		sproto.TaskTerminatedOnAgent:
		return d.receiveAgentMsg(ctx)

	case
		taskActorStopped,
		groupActorStopped,
		SetMaxSlots,
		SetWeight,
		AssignResource,
		TerminateTask:
		return d.receiveTaskMsg(ctx)

	case SetTaskName:
		reschedule = false
		d.receiveSetTaskName(ctx, msg)

	case GetTaskSummary:
		reschedule = false
		if resp := d.getTaskSummary(*msg.ID); resp != nil {
			ctx.Respond(*resp)
		}

	case GetTaskSummaries:
		reschedule = false
		ctx.Respond(d.taskList.TaskSummaries())

	case sproto.GetEndpointActorAddress:
		reschedule = false
		ctx.Respond("/agents")

	case schedulerTick:
		if d.reschedule {
			d.scheduler.Schedule(d)
			d.sendProvisionerView(ctx)
		}
		d.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, actionCoolDown, schedulerTick{})

	default:
		reschedule = false
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (d *DefaultRP) receiveAgentMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ConfigureEndpoints:
		ctx.Log().Infof("initializing endpoints for agents")
		agent.Initialize(msg.System, msg.Echo, ctx.Self())

	case sproto.AddAgent:
		ctx.Log().Infof("adding agent: %s", msg.Agent.Address().Local())
		d.agents[msg.Agent] = newAgentState(msg)

	case sproto.AddDevice:
		ctx.Log().Infof("adding device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := d.agents[msg.Agent]
		check.Panic(check.True(ok, "error adding device, agent not found: %s", msg.Agent.Address()))
		state.devices[msg.Device] = msg.ContainerID

	case sproto.FreeDevice:
		ctx.Log().Infof("freeing device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := d.agents[msg.Agent]
		check.Panic(check.True(ok, "error freeing device, agent not found: %s", msg.Agent.Address()))
		id, ok := d.agents[msg.Agent].devices[msg.Device]
		check.Panic(check.True(ok, "error freeing device, device not found: %s", msg.Device))
		check.Panic(check.True(id != nil, "error freeing device, device not assigned: %s", msg.Device))
		state.devices[msg.Device] = nil

	case sproto.RemoveDevice:
		ctx.Log().Infof("removing device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := d.agents[msg.Agent]
		check.Panic(check.True(ok, "error removing device, agent not found: %s", msg.Agent.Address()))
		delete(state.devices, msg.Device)

	case sproto.RemoveAgent:
		ctx.Log().Infof("removing agent: %s", msg.Agent.Address().Local())
		delete(d.agents, msg.Agent)

	case sproto.TaskStartedOnAgent:
		cid := ContainerID(msg.ContainerID)
		addresses := toAddresses(
			msg.ContainerStarted.ProxyAddress, msg.ContainerStarted.ContainerInfo)
		d.receiveContainerStartedOnAgent(ctx, cid, addresses)

	case sproto.TaskTerminatedOnAgent:
		cid := ContainerID(msg.ContainerID)
		d.receiveContainerTerminated(ctx, cid, *msg.ContainerStopped, false)
	}
	return nil
}

func (d *DefaultRP) receiveTaskMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case taskActorStopped:
		d.receiveTaskActorStopped(ctx, msg)

	case groupActorStopped:
		delete(d.groups, msg.Ref)

	case SetMaxSlots:
		d.getOrCreateGroup(msg.Handler, ctx).maxSlots = msg.MaxSlots

	case SetWeight:
		d.getOrCreateGroup(msg.Handler, ctx).weight = msg.Weight

	case AssignResource:
		d.receiveAddTask(ctx, msg)

	case TerminateTask:
		d.receiveTerminateTask(ctx, msg)
	}
	return nil
}

func (d *DefaultRP) receiveAddTask(ctx *actor.Context, msg AssignResource) {
	d.notifyOnStop(ctx, msg.TaskHandler, taskActorStopped{Ref: msg.TaskHandler})

	if task, ok := d.tasksByHandler[msg.TaskHandler]; ok {
		if ctx.ExpectingResponse() {
			ctx.Respond(task)
		}
		return
	}

	if msg.Group == nil {
		msg.Group = msg.TaskHandler
	}
	group := d.getOrCreateGroup(msg.Group, ctx)

	var taskID TaskID
	if msg.ID != nil {
		taskID = *msg.ID
	}

	// TODO: Auto-generate a nicer name.
	// TODO: Support for task name prefixes.
	name := msg.Name
	if len(name) == 0 {
		name = "Unnamed Task"
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

	d.tasksByID[task.ID] = task
	d.tasksByHandler[task.handler] = task
	d.taskList.Add(task)

	if ctx.ExpectingResponse() {
		ctx.Respond(task)
	}
}

func (d *DefaultRP) receiveContainerStartedOnAgent(
	ctx *actor.Context,
	containerID ContainerID,
	addresses []Address,
) {
	task := d.tasksByContainerID[containerID]
	if task == nil {
		ctx.Log().Warnf(
			"ignoring stale start message for container %s",
			containerID,
		)
		return
	}

	container := task.containers[containerID]
	container.addresses = addresses
	container.mustTransition(containerRunning)
}

// receiveContainerTerminated performs the necessary updates to the cluster
// state after a container has actually terminated. This may happen gracefully
// as part of responding to a ContainerTerminatedOnAgent message or abruptly
// (e.g., an agent agent actor, task, or task actor has stopped). Because all
// these scenarios can happen concurrently, this function is idempotent.
func (d *DefaultRP) receiveContainerTerminated(
	ctx *actor.Context,
	id ContainerID,
	reason aproto.ContainerStopped,
	aborted bool,
) {
	task := d.tasksByContainerID[id]
	if task == nil {
		ctx.Log().Infof(
			"ignoring stale terminated message for container %s",
			id,
		)
		return
	}

	container := task.containers[id]
	container.mustTransition(containerTerminated)
	container.exitStatus = &reason

	delete(container.agent.containers, container.id)
	delete(container.task.containers, container.id)
	delete(d.tasksByContainerID, container.id)

	// A task is terminated if and only if all of its containers are terminated.
	for _, container := range task.containers {
		if container.state != containerTerminated {
			return
		}
	}

	if task.state != taskTerminated {
		d.taskTerminated(task, aborted)
	}
}

func (d *DefaultRP) receiveTaskActorStopped(ctx *actor.Context, msg taskActorStopped) {
	task := d.tasksByHandler[msg.Ref]
	if task == nil {
		return
	}

	for _, container := range task.containers {
		d.receiveContainerTerminated(ctx, container.ID(), aproto.ContainerError(aproto.TaskError,
			errors.New("task has been stopped")), true)
	}

	// Clean up a task even if it does not have any containers yet.
	// Agents might error out that the containers do not exist.
	if task.state != taskTerminated {
		d.taskTerminated(task, true)
	}
}

func (d *DefaultRP) receiveSetTaskName(ctx *actor.Context, msg SetTaskName) {
	if task, ok := d.tasksByHandler[msg.TaskHandler]; ok {
		task.name = msg.Name
	}
}

func (d *DefaultRP) receiveTerminateTask(ctx *actor.Context, msg TerminateTask) {
	task := d.tasksByID[msg.TaskID]
	if task == nil {
		if ctx.ExpectingResponse() {
			ctx.Respond(task)
		}
		return
	}

	d.terminateTask(task, msg.Forcible)

	if ctx.ExpectingResponse() {
		ctx.Respond(task)
	}
}

func (d *DefaultRP) taskTerminated(task *Task, aborted bool) {
	task.mustTransition(taskTerminated)

	d.taskList.Remove(task)
	delete(d.tasksByID, task.ID)
	delete(d.tasksByHandler, task.handler)

	for id := range task.containers {
		delete(d.tasksByContainerID, id)
	}
}

func toAddresses(proxy string, info types.ContainerJSON) []Address {
	var addresses []Address
	switch info.HostConfig.NetworkMode {
	case "host":
		for port := range info.Config.ExposedPorts {
			addresses = append(addresses, Address{
				ContainerIP:   proxy,
				ContainerPort: port.Int(),
				HostIP:        proxy,
				HostPort:      port.Int(),
			})
		}
	default:
		if info.NetworkSettings == nil {
			return nil
		}
		networks := info.NetworkSettings.Networks
		ipAddresses := make([]string, 0, len(networks))
		for _, network := range networks {
			ipAddresses = append(ipAddresses, network.IPAddress)
		}
		for port, bindings := range info.NetworkSettings.Ports {
			for _, binding := range bindings {
				for _, ip := range ipAddresses {
					hostIP := binding.HostIP
					if hostIP == "" || hostIP == "0.0.0.0" {
						hostIP = proxy
					}
					hostPort, err := strconv.Atoi(binding.HostPort)
					if err != nil {
						panic(errors.Wrapf(err, "unexpected host port: %s", binding.HostPort))
					}
					addresses = append(addresses, Address{
						ContainerIP:   ip,
						ContainerPort: port.Int(),
						HostIP:        hostIP,
						HostPort:      hostPort,
					})
				}
			}
		}
	}
	return addresses
}

// containerAssignment contains information for tasks have been assigned but not yet started.
type containerAssignment struct {
	task                  *Task
	container             *container
	agent                 *agentState
	clusterID             string
	devices               []device.Device
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig
	masterCert            *tls.Certificate
}

// Start notifies the agent to start a container.
func (c containerAssignment) StartContainer(spec image.TaskSpec) {
	handler := c.agent.handler
	spec.ClusterID = c.clusterID
	spec.ContainerID = string(c.container.ID())
	spec.TaskID = string(c.task.ID)
	spec.HarnessPath = c.harnessPath
	spec.TaskContainerDefaults = c.taskContainerDefaults
	spec.MasterCert = c.masterCert
	spec.Devices = c.devices
	handler.System().Tell(handler, sproto.StartTaskOnAgent{
		Task: c.task.handler,
		StartContainer: aproto.StartContainer{
			Container: cproto.Container{
				Parent:  c.task.handler.Address(),
				ID:      cproto.ID(c.container.id),
				State:   cproto.Assigned,
				Devices: c.devices,
			},
			Spec: image.ToContainerSpec(spec),
		},
	})
}

// Kill notifies the agent to kill the container.
func (c containerAssignment) KillContainer() {
	c.agent.handler.System().Tell(c.agent.handler, sproto.KillContainer{
		ContainerID: cproto.ID(c.container.ID()),
	})
}
