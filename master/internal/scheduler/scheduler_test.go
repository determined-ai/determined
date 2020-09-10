package scheduler

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
)

type getMaxSlots struct{}
type getWeight struct{}

type mockGroup struct {
	maxSlots *int
	weight   float64
}

func newCustomGroup(
	t *testing.T,
	system *actor.System,
	id string,
	maxSlots int,
	weight float64,
) *actor.Ref {
	ref, created := system.ActorOf(
		actor.Addr(id),
		&mockGroup{maxSlots: &maxSlots, weight: weight},
	)
	assert.Assert(t, created)
	return ref
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
	group       *actor.Ref
	slotsNeeded int
	label       string
}

type (
	getSlots struct{}
	getGroup struct{}
	getLabel struct{}
)

func newMockTask(
	t *testing.T,
	system *actor.System,
	group *actor.Ref,
	id string,
	slotsNeeded int,
	label string,
) *actor.Ref {
	ref, created := system.ActorOf(actor.Addr(id),
		&mockTask{group: group, slotsNeeded: slotsNeeded, label: label})
	assert.Assert(t, created)
	return ref
}

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

type mockAgent struct{}

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
	case sproto.StartTaskOnAgent:
		if ctx.ExpectingResponse() {
			ctx.Respond(msg.Task)
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

type schedulerState struct {
	containers map[*agentState]int
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

func assertSchedulerState(
	t *testing.T, rp *DefaultRP, actual []*actor.Ref, expected []schedulerState,
) {
	for index, handler := range actual {
		expectedState := expected[index]
		actualAllocated := rp.taskList.GetAllocations(handler)
		actualContainers := make(map[*agentState]int)
		if actualAllocated != nil {
			for _, allocation := range actualAllocated.Allocations {
				container := allocation.(*containerAllocation).container
				actualContainers[container.agent] = container.slots
			}
		}
		assert.DeepEqual(t, expectedState.containers, actualContainers)
	}
	assert.Equal(t, len(actual), len(expected),
		"actual tasks and expected task states must have the same length")
}

func forceSchedule(rp *DefaultRP, handler *actor.Ref, agent *agentState) {
	req, _ := rp.taskList.GetTaskByHandler(handler)
	allocated := rp.taskList.GetAllocations(handler)
	if allocated == nil {
		allocated = &ResourcesAllocated{ID: req.ID}
	}
	allocated.Allocations = append(allocated.Allocations, &containerAllocation{
		req:       req,
		agent:     agent,
		container: newContainer(req, agent, req.SlotsNeeded, 1),
	})
	rp.taskList.SetAllocations(handler, allocated)
}
