package scheduler

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
)

type getMaxSlots struct{}
type getWeight struct{}

type mockGroup struct {
	id       string
	maxSlots *int
	weight   float64
}

func newGroup(t *testing.T, system *actor.System, id string) *actor.Ref {
	ref, created := system.ActorOf(actor.Addr(id), &mockGroup{})
	assert.Assert(t, created)
	return ref
}

func (g *mockGroup) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case getMaxSlots:
		ctx.Respond(g.maxSlots)
	case getWeight:
		ctx.Respond(g.weight)
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

type mockTask struct {
	id             TaskID
	slotsNeeded    int
	label          string
	group          *mockGroup
	allocatedAgent *mockAgent
}

type (
	getSlots struct{}
	getGroup struct{}
	getLabel struct{}
)

func (t *mockTask) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case ResourcesAllocated:
	case ReleaseResources:
	case getSlots:
		ctx.Respond(t.slotsNeeded)
	case getGroup:
		ctx.Respond(t.group)
	case getLabel:
		ctx.Respond(t.label)
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

type mockAgent struct {
	id    string
	slots int
	label string
}

func newMockAgent(
	t *testing.T,
	system *actor.System,
	id string,
	slots int,
	label string,
) *agentState {
	ref, created := system.ActorOf(actor.Addr(id), &mockAgent{})
	assert.Assert(t, created)
	state := newAgentState(sproto.AddAgent{Agent: ref, Label: label})
	for i := 0; i < slots; i++ {
		state.devices[device.Device{ID: i}] = nil
	}
	return state
}

func (m mockAgent) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.StartTaskContainer:
		if ctx.ExpectingResponse() {
			ctx.Respond(msg.TaskActor)
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func setupCluster(
	scheduler Scheduler, fittingMethod SoftConstraint, agents []*agentState, tasks []*actor.Ref,
) *DefaultRP {
	d := DefaultRP{
		scheduler:     scheduler,
		fittingMethod: fittingMethod,
		agents:        make(map[*actor.Ref]*agentState),
		groups:        make(map[*actor.Ref]*group),

		taskList:        newTaskList(),
		provisionerView: newProvisionerView(0),

		reschedule: false,
	}

	for _, agent := range agents {
		d.agents[agent.handler] = agent
	}

	for _, handler := range tasks {
		system := handler.System()

		g := system.Ask(handler, getGroup{}).Get().(*actor.Ref)
		slots := system.Ask(handler, getSlots{}).Get().(int)
		label := system.Ask(handler, getLabel{}).Get().(string)

		d.addAllocatedTask(&AllocateRequest{
			ID:           TaskID(handler.Address().String()),
			Name:         handler.Address().Local(),
			Group:        g,
			TaskActor:    handler,
			SlotsNeeded:  slots,
			CanTerminate: true,
			Label:        label,
		}, nil)
		_ = d.getOrCreateGroup(nil, g)
		if resp := system.Ask(g, getMaxSlots{}); resp.Get() != nil {
			d.getOrCreateGroup(nil, g).maxSlots = resp.Get().(*int)
		}
		if resp := system.Ask(g, getWeight{}); resp.Get() != nil {
			d.getOrCreateGroup(nil, g).weight = resp.Get().(float64)
		}
	}
	return &d
}

func setupClusterStates(
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
			ID:          mockTask.id,
			SlotsNeeded: mockTask.slotsNeeded,
			Label:       mockTask.label,
			TaskActor:   ref,
		}
		if mockTask.group == nil {
			req.Group = ref
		} else {
			req.Group = groupActors[mockTask.group]
		}
		taskList.AddTask(req)

		if mockTask.allocatedAgent != nil {
			agentRef := system.Get(actor.Addr(mockTask.allocatedAgent.id))
			agentState := agents[agentRef]

			allocated := &ResourcesAllocated{ID: req.ID}
			allocated.Allocations = append(allocated.Allocations, &containerAllocation{
				req:       req,
				agent:     agentState,
				container: newContainer(req, agentState, req.SlotsNeeded),
			})
			taskList.SetAllocations(req.TaskActor, allocated)
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

func newMaxSlot(maxSlot int) *int {
	return &maxSlot
}
