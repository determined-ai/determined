package scheduler

import (
	"fmt"
	"net/url"
	"strconv"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	actionCooldown = 500 * time.Millisecond
)

// schedulerTick periodically triggers the scheduler to act.
type schedulerTick struct{}

// Cluster manages the agent and task lifecycles.
type Cluster struct {
	clusterID             string
	scheduler             Scheduler
	fittingMethod         SoftConstraint
	agents                map[*actor.Ref]*agentState
	groups                map[*actor.Ref]*group
	proxy                 *actor.Ref
	registeredNames       map[*container][]string
	harnessPath           string
	taskContainerDefaults model.TaskContainerDefaultsConfig

	taskList           *taskList
	tasksByHandler     map[*actor.Ref]*Task
	tasksByID          map[TaskID]*Task
	tasksByContainerID map[ContainerID]*Task

	provisioner     *actor.Ref
	provisionerView *FilterableView

	saveNotifications bool
	notifications     []<-chan struct{}

	reschedule bool
}

// NewCluster initializes a new empty cluster.
func NewCluster(
	clusterID string,
	scheduler Scheduler,
	fittingMethod SoftConstraint,
	proxy *actor.Ref,
	harnessPath string,
	taskContainerDefaults model.TaskContainerDefaultsConfig,
	provisioner *actor.Ref,
	provisionerSlotsPerInstance int,
) *Cluster {
	c := &Cluster{
		clusterID:             clusterID,
		scheduler:             scheduler,
		fittingMethod:         fittingMethod,
		agents:                make(map[*actor.Ref]*agentState),
		groups:                make(map[*actor.Ref]*group),
		registeredNames:       make(map[*container][]string),
		harnessPath:           harnessPath,
		taskContainerDefaults: taskContainerDefaults,

		taskList:           newTaskList(),
		tasksByHandler:     make(map[*actor.Ref]*Task),
		tasksByID:          make(map[TaskID]*Task),
		tasksByContainerID: make(map[ContainerID]*Task),

		proxy:           proxy,
		provisioner:     provisioner,
		provisionerView: newProvisionerView(provisionerSlotsPerInstance),

		reschedule: false,
	}
	return c
}

func (c *Cluster) assignContainer(task *Task, agent *agentState, slots int, numContainers int) {
	if task.state != taskRunning {
		task.mustTransition(taskRunning)
	}
	container := newContainer(task, agent, slots, len(task.containers))
	agent.containers[container.id] = container
	task.containers[container.id] = container
	c.tasksByContainerID[container.id] = task
	assigned := Assigned{
		task:                  task,
		agent:                 agent,
		container:             container,
		numContainers:         numContainers,
		clusterID:             c.clusterID,
		devices:               agent.assignFreeDevices(slots, container.id),
		harnessPath:           c.harnessPath,
		taskContainerDefaults: c.taskContainerDefaults,
	}
	task.handler.System().Tell(task.handler, assigned)
}

// assignTask allocates cluster data structures and sends the appropriate actor
// messages to start a task if there are enough resources in the cluster to run
// the task. If there are not, assignTask returns false.
func (c *Cluster) assignTask(task *Task) bool {
	fits := findFits(task, c.agents, c.fittingMethod)

	for _, fit := range fits {
		c.assignContainer(task, fit.Agent, fit.Slots, len(fits))
	}
	return len(fits) > 0
}

// terminateTask sends the appropriate actor messages to terminate a task and
// deallocate its cluster data structures. The task may not be terminated if it
// is in the right state unless forcible is true.
func (c *Cluster) terminateTask(task *Task, forcible bool) {
	switch {
	case task.state == taskTerminated:
		// The task has already been terminated so this is a noop.

	case len(task.containers) == 0 || task.state == taskPending:
		// The task is not running so there is no need to request the task to terminate. The task is
		// marked as aborted.
		c.taskTerminated(task, true)

	case forcible:
		// Notify the agent to kill the task.
		task.mustTransition(taskTerminating)
		for _, c := range task.containers {
			if c.state != containerTerminated {
				c.mustTransition(containerTerminating)
			}
			c.agent.handler.System().Tell(c.agent.handler, agent.SignalContainer{
				ContainerID: cproto.ID(c.id), Signal: syscall.SIGKILL})
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

func (c *Cluster) getOrCreateGroup(handler *actor.Ref, ctx *actor.Context) *group {
	if g, ok := c.groups[handler]; ok {
		return g
	}
	g := &group{handler: handler, weight: 1}
	c.groups[handler] = g
	if ctx != nil && handler != nil { // ctx is nil only for testing purposes.
		actors.NotifyOnStop(ctx, handler, groupStopped{})
	}
	return g
}

func (c *Cluster) getTaskSummary(id TaskID) *TaskSummary {
	if task := c.tasksByID[id]; task != nil {
		summary := newTaskSummary(task)
		return &summary
	}
	return nil
}

func (c *Cluster) notifyOnStop(ctx *actor.Context, ref *actor.Ref, msg actor.Message) {
	done := actors.NotifyOnStop(ctx, ref, msg)
	if c.saveNotifications {
		c.notifications = append(c.notifications, done)
	}
}

func (c *Cluster) sendProvisionerView(ctx *actor.Context) {
	if c.provisioner != nil {
		if snapshot, updateMade := c.provisionerView.Update(c); updateMade {
			ctx.Tell(c.provisioner, snapshot)
		}
	}
}

// Receive implements the actor.Actor interface.
func (c *Cluster) Receive(ctx *actor.Context) error {
	reschedule := true
	defer func() {
		// Default to scheduling every 500ms if a message was received, but allow messages
		// that don't affect the cluster to be skipped.
		c.reschedule = c.reschedule || reschedule
	}()

	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, actionCooldown, schedulerTick{})

	case AddAgent:
		ctx.Log().Infof("adding agent: %s", msg.Agent.Address().Local())
		c.agents[msg.Agent] = newAgentState(msg)

	case AddDevice:
		ctx.Log().Infof("adding device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := c.agents[msg.Agent]
		check.Panic(check.True(ok, "error adding device, agent not found: %s", msg.Agent.Address()))
		state.devices[msg.Device] = msg.ContainerID

	case FreeDevice:
		ctx.Log().Infof("freeing device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := c.agents[msg.Agent]
		check.Panic(check.True(ok, "error freeing device, agent not found: %s", msg.Agent.Address()))
		id, ok := c.agents[msg.Agent].devices[msg.Device]
		check.Panic(check.True(ok, "error freeing device, device not found: %s", msg.Device))
		check.Panic(check.True(id != nil, "error freeing device, device not assigned: %s", msg.Device))
		state.devices[msg.Device] = nil

	case RemoveDevice:
		ctx.Log().Infof("removing device: %s (%s)", msg.Device.String(), msg.Agent.Address().Local())
		state, ok := c.agents[msg.Agent]
		check.Panic(check.True(ok, "error removing device, agent not found: %s", msg.Agent.Address()))
		delete(state.devices, msg.Device)

	case RemoveAgent:
		ctx.Log().Infof("removing agent: %s", msg.Agent.Address().Local())
		delete(c.agents, msg.Agent)

	case agent.ContainerStateChanged:
		cid := ContainerID(msg.Container.ID)
		switch msg.Container.State {
		case cproto.Running:
			c.receiveContainerStartedOnAgent(ctx, ContainerStartedOnAgent{
				ContainerID: cid,
				Addresses: toAddresses(
					msg.ContainerStarted.ProxyAddress, msg.ContainerStarted.ContainerInfo),
			})
		case cproto.Terminated:
			c.receiveContainerTerminated(ctx, cid, *msg.ContainerStopped, false)
		}

	case taskStopped:
		c.receiveTaskStopped(ctx, msg)

	case groupStopped:
		delete(c.groups, msg.Ref)

	case SetMaxSlots:
		c.getOrCreateGroup(ctx.Sender(), ctx).maxSlots = msg.MaxSlots

	case SetWeight:
		c.getOrCreateGroup(ctx.Sender(), ctx).weight = msg.Weight

	case AddTask:
		c.receiveAddTask(ctx, msg)

	case SetTaskName:
		reschedule = false
		c.receiveSetTaskName(ctx, msg)

	case TerminateTask:
		c.receiveTerminateTask(ctx, msg)

	case GetTaskSummary:
		reschedule = false
		if resp := c.getTaskSummary(*msg.ID); resp != nil {
			ctx.Respond(*resp)
		}

	case GetTaskSummaries:
		reschedule = false
		ctx.Respond(c.taskList.TaskSummaries())

	case schedulerTick:
		if c.reschedule {
			c.scheduler.Schedule(c)
			c.sendProvisionerView(ctx)
		}
		c.reschedule = false
		reschedule = false
		actors.NotifyAfter(ctx, actionCooldown, schedulerTick{})

	default:
		reschedule = false
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (c *Cluster) receiveAddTask(ctx *actor.Context, msg AddTask) {
	c.notifyOnStop(ctx, ctx.Sender(), taskStopped{Ref: ctx.Sender()})

	if task, ok := c.tasksByHandler[ctx.Sender()]; ok {
		if ctx.ExpectingResponse() {
			ctx.Respond(task)
		}
		return
	}

	if msg.Group == nil {
		msg.Group = ctx.Sender()
	}
	group := c.getOrCreateGroup(msg.Group, ctx)

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
		handler:             ctx.Sender(),
		name:                name,
		slotsNeeded:         msg.SlotsNeeded,
		canTerminate:        msg.CanTerminate,
		agentLabel:          msg.Label,
		fittingRequirements: msg.FittingRequirements,
	})

	c.tasksByID[task.ID] = task
	c.tasksByHandler[task.handler] = task
	c.taskList.Add(task)

	if ctx.ExpectingResponse() {
		ctx.Respond(task)
	}
}

func (c *Cluster) receiveContainerStartedOnAgent(ctx *actor.Context, msg ContainerStartedOnAgent) {
	task := c.tasksByContainerID[msg.ContainerID]
	if task == nil {
		ctx.Log().Warnf(
			"ignoring stale start message for container %s",
			msg.ContainerID,
		)
		return
	}

	container := task.containers[msg.ContainerID]
	container.addresses = msg.Addresses
	container.mustTransition(containerRunning)
	handler := container.task.handler
	handler.System().Tell(handler, ContainerStarted{Container: container})

	if len(msg.Addresses) == 0 {
		return
	}

	names := make([]string, 0, len(msg.Addresses))
	for _, address := range msg.Addresses {
		// We are keying on task ID instead of container ID. Revisit this when we need to
		// proxy multi-container tasks or when containers are created prior to being
		// assigned to an agent.
		ctx.Ask(c.proxy, proxy.Register{
			Service: string(task.ID),
			Target: &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", address.HostIP, address.HostPort),
			},
		})
		names = append(names, string(task.ID))
	}

	c.registeredNames[container] = names
}

// receiveContainerTerminated performs the necessary updates to the cluster
// state after a container has actually terminated. This may happen gracefully
// as part of responding to a ContainerTerminatedOnAgent message or abruptly
// (e.g., an agent agent actor, task, or task actor has stopped). Because all
// these scenarios can happen concurrently, this function is idempotent.
func (c *Cluster) receiveContainerTerminated(
	ctx *actor.Context,
	id ContainerID,
	reason agent.ContainerStopped,
	aborted bool,
) {
	task := c.tasksByContainerID[id]
	if task == nil {
		ctx.Log().Infof(
			"ignoring stale terminated message for container %s",
			id,
		)
		return
	}

	container := task.containers[id]
	if names, ok := c.registeredNames[container]; ok {
		for _, name := range names {
			ctx.Tell(c.proxy, proxy.Unregister{Service: name})
		}
		delete(c.registeredNames, container)
	}

	container.mustTransition(containerTerminated)
	container.exitStatus = &reason

	delete(container.agent.containers, container.id)
	delete(container.task.containers, container.id)
	delete(c.tasksByContainerID, container.id)

	// A task is terminated if and only if all of its containers are terminated.
	for _, container := range task.containers {
		if container.state != containerTerminated {
			return
		}
	}

	if task.state != taskTerminated {
		c.taskTerminated(task, aborted)
	}
}

func (c *Cluster) receiveTaskStopped(ctx *actor.Context, msg taskStopped) {
	// TODO(shiyuan): refactor to update agent.py to complain less if we try to kill an
	//  container that does not exist.
	task := c.tasksByHandler[msg.Ref]
	if task == nil {
		return
	}

	for _, container := range task.containers {
		c.receiveContainerTerminated(ctx, container.ID(), agent.ContainerError(agent.TaskError,
			errors.New("task has been stopped")), true)
	}

	// Clean up a task even if it does not have any containers yet.
	if task.state != taskTerminated {
		c.taskTerminated(task, true)
	}
}

func (c *Cluster) receiveSetTaskName(ctx *actor.Context, msg SetTaskName) {
	if task, ok := c.tasksByHandler[ctx.Sender()]; ok {
		task.name = msg.Name
	}
}

func (c *Cluster) receiveTerminateTask(ctx *actor.Context, msg TerminateTask) {
	task := c.tasksByID[msg.TaskID]
	if task == nil {
		if ctx.ExpectingResponse() {
			ctx.Respond(task)
		}
		return
	}

	c.terminateTask(task, msg.Forcible)

	if ctx.ExpectingResponse() {
		ctx.Respond(task)
	}
}

func (c *Cluster) taskTerminated(task *Task, aborted bool) {
	task.mustTransition(taskTerminated)

	c.taskList.Remove(task)
	delete(c.tasksByID, task.ID)
	delete(c.tasksByHandler, task.handler)

	for id := range task.containers {
		delete(c.tasksByContainerID, id)
	}

	task.handler.System().Tell(task.handler, TaskTerminated{
		Task:    newTaskSummary(task),
		Aborted: aborted,
	})
	// This is somewhat redundant with the message above, but we're transitioning between them.
	if aborted {
		task.handler.System().Tell(task.handler, TaskAborted{})
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
				Protocol:      port.Proto(),
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
						Protocol:      port.Proto(),
					})
				}
			}
		}
	}
	return addresses
}
