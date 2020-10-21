package resourcemanagers

import (
	"testing"

	image "github.com/determined-ai/determined/master/pkg/tasks"

	"github.com/google/uuid"

	"github.com/pkg/errors"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
)

func newMaxSlot(maxSlot int) *int {
	return &maxSlot
}

type mockGroup struct {
	id       string
	maxSlots *int
	weight   float64
}

func (g *mockGroup) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
	case actor.PostStop:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

type (
	SendRequestResourcesToResourceManager  struct{}
	SendResourcesReleasedToResourceManager struct{}
	ThrowError                             struct{}
	ThrowPanic                             struct{}
)

var errMock = errors.New("mock error")

type mockTask struct {
	rmRef *actor.Ref

	id               TaskID
	group            *mockGroup
	slotsNeeded      int
	nonPreemptible   bool
	label            string
	resourcePool     string
	allocatedAgent   *mockAgent
	containerStarted bool
}

func (t *mockTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
	case actor.PostStop:
	case SendRequestResourcesToResourceManager:
		task := AllocateRequest{
			ID:             t.id,
			Name:           string(t.id),
			SlotsNeeded:    t.slotsNeeded,
			NonPreemptible: t.nonPreemptible,
			Label:          t.label,
			ResourcePool:   t.resourcePool,
			TaskActor:      ctx.Self(),
		}
		if t.group == nil {
			task.Group = ctx.Self()
		} else {
			task.Group = ctx.Self().System().Get(actor.Addr(t.group.id))
		}
		if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(t.rmRef, task).Get())
		} else {
			ctx.Tell(t.rmRef, task)
		}
	case SendResourcesReleasedToResourceManager:
		task := ResourcesReleased{TaskActor: ctx.Self()}
		if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(t.rmRef, task).Get())
		} else {
			ctx.Tell(t.rmRef, task)
		}
	case ThrowError:
		return errMock
	case ThrowPanic:
		panic(errMock)

	case ResourcesAllocated:
		for _, allocation := range msg.Allocations {
			allocation.Start(ctx, image.TaskSpec{})
		}
	case ReleaseResources:
		ctx.Tell(t.rmRef, ResourcesReleased{TaskActor: ctx.Self()})

	case sproto.TaskContainerStateChanged:

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

type mockAgent struct {
	id                    string
	label                 string
	slots                 int
	slotsUsed             int
	maxZeroSlotContainers *int
	zeroSlotContainers    int
}

func newMockAgent(
	id string,
	label string,
	slots int,
	slotsUsed int,
	maxZeroSlotContainers *int,
	zeroSlotContainers int,
) *mockAgent {
	return &mockAgent{
		id:                    id,
		label:                 label,
		slots:                 slots,
		slotsUsed:             slotsUsed,
		maxZeroSlotContainers: maxZeroSlotContainers,
		zeroSlotContainers:    zeroSlotContainers,
	}
}

func (m *mockAgent) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
	case actor.PostStop:
	case sproto.StartTaskContainer:
	case sproto.KillTaskContainer:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func setupResourcePool(
	t *testing.T,
	system *actor.System,
	config *ResourcePoolConfig,
	mockTasks []*mockTask,
	mockGroups []*mockGroup,
	mockAgents []*mockAgent,
) (*ResourcePool, *actor.Ref) {
	if config == nil {
		config = &ResourcePoolConfig{PoolName: "pool"}
	}
	rp := NewResourcePool(config, nil, NewFairShareScheduler(), BestFit)
	rp.taskList, rp.groups, rp.agents = setupSchedulerStates(
		t, system, mockTasks, mockGroups, mockAgents,
	)
	rp.saveNotifications = true
	ref, created := system.ActorOf(actor.Addr(rp.config.PoolName), rp)
	assert.Assert(t, created)
	system.Ask(ref, actor.Ping{}).Get()

	for _, task := range mockTasks {
		task.rmRef = ref
	}
	return rp, ref
}

func forceAddAgent(
	t *testing.T,
	system *actor.System,
	agents map[*actor.Ref]*agentState,
	agentID string,
	numSlots int,
	numUsedSlots int,
	numZeroSlotContainers int,
) *agentState {
	ref, created := system.ActorOf(actor.Addr(agentID), &mockAgent{id: agentID, slots: numSlots})
	assert.Assert(t, created)
	state := newAgentState(sproto.AddAgent{Agent: ref}, nil)
	for i := 0; i < numSlots; i++ {
		state.devices[device.Device{ID: i}] = nil
	}
	i := 0
	for ix := range state.devices {
		if i < numUsedSlots {
			id := cproto.ID(uuid.New().String())
			state.devices[ix] = &id
		}
	}
	for i := 0; i < numZeroSlotContainers; i++ {
		state.zeroSlotContainers[cproto.ID(uuid.New().String())] = true
	}
	agents[state.handler] = state
	return state
}

func newFakeAgentState(
	t *testing.T,
	system *actor.System,
	id string,
	label string,
	slots int,
	slotsUsed int,
	maxZeroSlotContainers *int,
	zeroSlotContainers int,
) *agentState {
	ref, created := system.ActorOf(actor.Addr(id), &mockAgent{id: id, slots: slots, label: label})
	assert.Assert(t, created)
	state := newAgentState(sproto.AddAgent{Agent: ref, Label: label}, maxZeroSlotContainers)
	for i := 0; i < slots; i++ {
		state.devices[device.Device{ID: i}] = nil
	}

	if slotsUsed > 0 {
		req := &AllocateRequest{
			SlotsNeeded:    slotsUsed,
			NonPreemptible: false,
		}
		container := newContainer(req, state, req.SlotsNeeded)
		state.allocateFreeDevices(req.SlotsNeeded, container.id)
	}

	for i := 0; i < zeroSlotContainers; i++ {
		req := &AllocateRequest{}
		container := newContainer(req, state, req.SlotsNeeded)
		state.allocateFreeDevices(req.SlotsNeeded, container.id)
	}
	return state
}

func forceAddTask(
	t *testing.T,
	system *actor.System,
	taskList *taskList,
	taskID string,
	numAllocated int,
	slotsNeeded int,
) {
	task := &mockTask{id: TaskID(taskID), slotsNeeded: slotsNeeded}
	ref, created := system.ActorOf(actor.Addr(taskID), task)
	assert.Assert(t, created)

	req := &AllocateRequest{
		ID:          TaskID(taskID),
		TaskActor:   ref,
		Group:       ref,
		SlotsNeeded: slotsNeeded,
	}
	taskList.AddTask(req)
	forceSetTaskAllocations(t, taskList, taskID, numAllocated)
}

func forceSetTaskAllocations(
	t *testing.T,
	taskList *taskList,
	taskID string,
	numAllocated int,
) {
	req, ok := taskList.GetTaskByID(TaskID(taskID))
	assert.Check(t, ok)
	if numAllocated > 0 {
		allocated := &ResourcesAllocated{ID: TaskID(taskID), Allocations: []Allocation{}}
		for i := 0; i < numAllocated; i++ {
			allocated.Allocations = append(allocated.Allocations, containerAllocation{})
		}
		taskList.SetAllocations(req.TaskActor, allocated)
	} else {
		taskList.SetAllocations(req.TaskActor, nil)
	}
}

func setupSchedulerStates(
	t *testing.T,
	system *actor.System,
	mockTasks []*mockTask,
	mockGroups []*mockGroup,
	mockAgents []*mockAgent,
) (
	*taskList,
	map[*actor.Ref]*group,
	map[*actor.Ref]*agentState,
) {
	agents := make(map[*actor.Ref]*agentState)
	for _, mockAgent := range mockAgents {
		ref, created := system.ActorOf(actor.Addr(mockAgent.id), mockAgent)
		assert.Assert(t, created)

		agent := &agentState{
			handler:            ref,
			label:              mockAgent.label,
			devices:            make(map[device.Device]*cproto.ID),
			zeroSlotContainers: make(map[cproto.ID]bool),
		}
		for i := 0; i < mockAgent.slots; i++ {
			agent.devices[device.Device{ID: i}] = nil
		}
		agents[ref] = agent
	}

	groups := make(map[*actor.Ref]*group)
	groupActors := make(map[*mockGroup]*actor.Ref)
	for _, mockGroup := range mockGroups {
		ref, created := system.ActorOf(actor.Addr(mockGroup.id), mockGroup)
		assert.Assert(t, created)

		group := &group{
			handler:  ref,
			maxSlots: mockGroup.maxSlots,
			weight:   mockGroup.weight,
		}
		groups[ref] = group
		groupActors[mockGroup] = ref
	}

	taskList := newTaskList()
	for _, mockTask := range mockTasks {
		ref, created := system.ActorOf(actor.Addr(mockTask.id), mockTask)
		assert.Assert(t, created)

		groups[ref] = &group{handler: ref}

		req := &AllocateRequest{
			ID:             mockTask.id,
			SlotsNeeded:    mockTask.slotsNeeded,
			Label:          mockTask.label,
			TaskActor:      ref,
			NonPreemptible: mockTask.nonPreemptible,
		}
		if mockTask.group == nil {
			req.Group = ref
		} else {
			req.Group = groupActors[mockTask.group]
		}
		taskList.AddTask(req)

		if mockTask.allocatedAgent != nil {
			assert.Assert(t, mockTask.allocatedAgent.slots >= mockTask.slotsNeeded)
			agentRef := system.Get(actor.Addr(mockTask.allocatedAgent.id))
			agentState := agents[agentRef]
			container := newContainer(req, agentState, req.SlotsNeeded)

			allocated := &ResourcesAllocated{
				ID: req.ID,
				Allocations: []Allocation{
					containerAllocation{
						req:       req,
						agent:     agentState,
						container: container,
					},
				},
			}
			taskList.SetAllocations(req.TaskActor, allocated)

			if mockTask.containerStarted {
				if mockTask.slotsNeeded == 0 {
					agentState.zeroSlotContainers[container.id] = true
				} else {
					i := 0
					for d, containerID := range agentState.devices {
						if containerID != nil {
							continue
						}
						if i < mockTask.slotsNeeded {
							agentState.devices[d] = &container.id
							i++
						}
					}
					assert.Assert(t, i == mockTask.slotsNeeded,
						"over allocated to agent %s", mockTask.allocatedAgent.id)
				}
			}
		}
	}

	return taskList, groups, agents
}

func assertEqualToAllocate(
	t *testing.T,
	actual []*AllocateRequest,
	expected []*mockTask,
) {
	expectedMap := map[TaskID]bool{}
	for _, task := range expected {
		expectedMap[task.id] = true
	}
	for _, task := range actual {
		_, ok := expectedMap[task.ID]
		assert.Assert(t, ok)
	}
	assert.Equal(t, len(actual), len(expected),
		"actual tasks and expected tasks must have the same length")
}

func assertEqualToRelease(
	t *testing.T,
	taskList *taskList,
	actual []*actor.Ref,
	expected []*mockTask,
) {
	expectedMap := map[TaskID]bool{}
	for _, task := range expected {
		expectedMap[task.id] = true
	}
	for _, taskActor := range actual {
		task, _ := taskList.GetTaskByHandler(taskActor)
		assert.Assert(t, task != nil)

		if task != nil {
			_, ok := expectedMap[task.ID]
			assert.Assert(t, ok)
		}
	}
	assert.Equal(t, len(actual), len(expected),
		"actual tasks and expected tasks must have the same length")
}
