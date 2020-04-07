package scheduler

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
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
	slots int,
	weight float64,
) *actor.Ref {
	ref, created := system.ActorOf(
		actor.Addr(id),
		&mockGroup{maxSlots: &slots, weight: weight},
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
	switch msg := ctx.Message().(type) {
	case Assigned:
		msg.StartTask(tasks.TaskSpec{})
	case getSlots:
		ctx.Respond(t.slotsNeeded)
	case getGroup:
		ctx.Respond(t.group)
	case getLabel:
		ctx.Respond(t.label)
	case TerminateRequest:
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
	state := newAgentState(AddAgent{Agent: ref, Label: label})
	for i := 0; i < slots; i++ {
		state.devices[device.Device{ID: i}] = nil
	}
	return state
}

func (m mockAgent) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case StartTask:
		if ctx.ExpectingResponse() {
			ctx.Respond(newTask(&Task{
				handler: msg.Task,
			}))
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

type schedulerState struct {
	state      taskState
	containers map[*agentState]int
}

func setupCluster(
	scheduler Scheduler, fittingMethod SoftConstraint, agents []*agentState, tasks []*actor.Ref,
) *Cluster {
	c := NewCluster("cluster", scheduler, fittingMethod, nil,
		"/opt/determined", model.ContainerDefaultsConfig{}, nil, 0)
	for _, agent := range agents {
		c.agents[agent.handler] = agent
	}
	for _, handler := range tasks {
		system := handler.System()

		g := system.Ask(handler, getGroup{}).Get().(*actor.Ref)
		slots := system.Ask(handler, getSlots{}).Get().(int)
		label := system.Ask(handler, getLabel{}).Get().(string)

		c.addTask(&Task{
			ID:           TaskID(handler.Address().String()),
			name:         handler.Address().Local(),
			group:        c.getOrCreateGroup(g, nil),
			handler:      handler,
			slotsNeeded:  slots,
			canTerminate: true,
			agentLabel:   label,
		})
		if resp := system.Ask(g, getMaxSlots{}); resp.Get() != nil {
			c.getOrCreateGroup(g, nil).maxSlots = resp.Get().(*int)
		}
		if resp := system.Ask(g, getWeight{}); resp.Get() != nil {
			c.getOrCreateGroup(g, nil).weight = resp.Get().(float64)
		}
	}
	return c
}

func assertSchedulerState(
	t *testing.T, cluster *Cluster, actual []*actor.Ref, expected []schedulerState,
) {
	for index, handler := range actual {
		task := cluster.tasksByHandler[handler]
		expectedState := expected[index]
		assert.Equal(t, task.state, expectedState.state, "task %d has an incorrect state", index)
		if task.state != taskPending {
			actualContainers := make(map[*agentState]int)
			for _, container := range task.containers {
				actualContainers[container.agent] = container.slots
			}
			assert.DeepEqual(t, expectedState.containers, actualContainers)
		} else {
			assert.Equal(t, len(task.containers), 0,
				"Pending task %d has a scheduled container", index)
		}
	}
	assert.Equal(t, len(actual), len(expected),
		"actual tasks and expected task states must have the same length")
}

func forceSchedule(cluster *Cluster, handler *actor.Ref, agent *agentState) {
	task := cluster.tasksByHandler[handler]
	cluster.assignContainer(task, agent, task.SlotsNeeded(), 1)
}
