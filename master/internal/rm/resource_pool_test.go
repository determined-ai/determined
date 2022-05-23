package rm

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
)

func TestCleanUpTaskWhenTaskActorStopsWithError(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{{id: "agent", slots: 1}}
	tasks := []*mockTask{{id: "task", slotsNeeded: 1}}
	rp, ref := setupResourcePool(t, nil, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetAllocationSummaries{}).Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, ThrowError{})
	assert.ErrorType(t, taskRef.StopAndAwaitTermination(), errMock)

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.len(), 0)
}

func TestCleanUpTaskWhenTaskActorPanics(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{{id: "agent", slots: 1}}
	tasks := []*mockTask{{id: "task", slotsNeeded: 1}}
	rp, ref := setupResourcePool(t, nil, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetAllocationSummaries{}).Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, ThrowPanic{})
	assert.ErrorType(t, taskRef.StopAndAwaitTermination(), errMock)

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.len(), 0)
}

func TestCleanUpTaskWhenTaskActorStopsNormally(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{{id: "agent", slots: 1}}
	tasks := []*mockTask{{id: "task", slotsNeeded: 1}}
	rp, ref := setupResourcePool(t, nil, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetAllocationSummaries{}).Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 1)

	assert.NilError(t, taskRef.StopAndAwaitTermination())

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.len(), 0)
}

func TestCleanUpTaskWhenTaskActorReleaseResources(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{{id: "agent", slots: 1}}
	tasks := []*mockTask{{id: "task", slotsNeeded: 1}}
	rp, ref := setupResourcePool(t, nil, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetAllocationSummaries{}).Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, sproto.ReleaseResources{}).Get()

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.len(), 0)
}

func TestScalingInfoAgentSummary(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*mockAgent{
		{id: "agent1", slots: 1},
		{id: "agent2", slots: 1},
	}
	tasks := []*mockTask{
		{id: "allocated-cpu-task1", slotsNeeded: 0, allocatedAgent: agents[0], containerStarted: true},
		{id: "allocated-cpu-task2", slotsNeeded: 0, allocatedAgent: agents[1], containerStarted: true},
		{id: "allocated-gpu-task3", slotsNeeded: 1, allocatedAgent: agents[1], containerStarted: true},
		{id: "unallocated-gpu-task4", slotsNeeded: 1},
		{id: "unallocated-gpu-task5", slotsNeeded: 5},
	}
	rp, _ := setupResourcePool(t, nil, system, nil, tasks, nil, agents)
	rp.slotsPerInstance = 4

	// Test basic.
	updated := rp.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *rp.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent1": {Name: "agent1", IsIdle: false},
			"agent2": {Name: "agent2", IsIdle: false},
		},
	})

	// Test adding agents.
	agent3 := forceAddAgent(t, system, rp.agentStatesCache, "agent3", 4, 0, 0)
	forceAddAgent(t, system, rp.agentStatesCache, "agent4", 4, 1, 0)
	updated = rp.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *rp.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent1": {Name: "agent1", IsIdle: false},
			"agent2": {Name: "agent2", IsIdle: false},
			"agent3": {Name: "agent3", IsIdle: true},
			"agent4": {Name: "agent4", IsIdle: false},
		},
	})

	// Test removing agents.
	agent1 := system.Get(actor.Addr("agent1"))
	delete(rp.agents, agent1)
	delete(rp.agentStatesCache, agent1)
	updated = rp.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *rp.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent2": {Name: "agent2", IsIdle: false},
			"agent3": {Name: "agent3", IsIdle: true},
			"agent4": {Name: "agent4", IsIdle: false},
		},
	})

	// Test agent state change.
	// Allocate a container to a device of the agent2.
	i := 0
	for d := range rp.agentStatesCache[agent3.Handler].Devices {
		if i == 0 {
			id := cproto.ID(uuid.New().String())
			rp.agentStatesCache[agent3.Handler].Devices[d] = &id
		}
		i++
	}
	updated = rp.updateScalingInfo()
	assert.Check(t, updated)
	assert.DeepEqual(t, *rp.scalingInfo, sproto.ScalingInfo{
		DesiredNewInstances: 1,
		Agents: map[string]sproto.AgentSummary{
			"agent2": {Name: "agent2", IsIdle: false},
			"agent3": {Name: "agent3", IsIdle: false},
			"agent4": {Name: "agent4", IsIdle: false},
		},
	})
}

func TestSettingGroupPriority(t *testing.T) {
	system := actor.NewSystem(t.Name())
	defaultPriority := 50
	config := config.ResourcePoolConfig{
		Scheduler: &config.SchedulerConfig{
			Priority: &config.PrioritySchedulerConfig{
				DefaultPriority: &defaultPriority,
			},
			FittingPolicy: best,
		},
	}

	rp, ref := setupResourcePool(t, nil, system, &config, nil, nil, nil)

	// Test setting a non-default priority for a group.
	groupRefOne, created := system.ActorOf(actor.Addr("group1"), &mockGroup{})
	assert.Assert(t, created)
	updatedPriority := 22
	system.Ask(ref, sproto.SetGroupPriority{Priority: updatedPriority, Handler: groupRefOne})

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, *rp.groups[groupRefOne].priority, updatedPriority)
}

func TestAddRemoveAgent(t *testing.T) {
	system := actor.NewSystem(t.Name())
	db := &mocks.DB{}
	_, ref := setupResourcePool(t, db, system, nil, nil, nil, nil)
	agentRef, created := system.ActorOf(actor.Addr("agent"), &mockAgent{id: "agent", slots: 2})
	assert.Assert(t, created)

	system.Tell(ref, sproto.AddAgent{Agent: agentRef})
	db.On("RecordAgentStats", mock.Anything).Return(nil)

	system.Tell(ref, sproto.RemoveAgent{Agent: agentRef})
}

func setupRPSamePriority(t *testing.T) *ResourcePool {
	system := actor.NewSystem(t.Name())
	defaultPriority := 50
	config := config.ResourcePoolConfig{
		Scheduler: &config.SchedulerConfig{
			Priority: &config.PrioritySchedulerConfig{
				DefaultPriority: &defaultPriority,
			},
			FittingPolicy: best,
		},
	}

	rp, _ := setupResourcePool(t, nil, system, &config, nil, nil, nil)

	groupRefOne, created := system.ActorOf(actor.Addr("group1"), &mockGroup{})
	assert.Assert(t, created)
	groupRefTwo, created := system.ActorOf(actor.Addr("group2"), &mockGroup{})
	assert.Assert(t, created)
	groupRefThree, created := system.ActorOf(actor.Addr("group3"), &mockGroup{})
	assert.Assert(t, created)

	rp.queuePositions = map[model.JobID]decimal.Decimal{
		"job1": decimal.New(100, 1000),
		"job2": decimal.New(200, 1000),
		"job3": decimal.New(300, 1000),
	}

	rp.groups = map[*actor.Ref]*group{
		groupRefOne:   {priority: &defaultPriority},
		groupRefTwo:   {priority: &defaultPriority},
		groupRefThree: {priority: &defaultPriority},
	}

	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation1",
		AllocationActor: groupRefOne,
		JobID:           "job1",
		Group:           groupRefOne,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation2",
		AllocationActor: groupRefTwo,
		JobID:           "job2",
		Group:           groupRefTwo,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation3",
		AllocationActor: groupRefThree,
		JobID:           "job3",
		Group:           groupRefThree,
	})

	return rp
}

func TestMoveMessagesPromote(t *testing.T) {
	rp := setupRPSamePriority(t)

	// move job3 above job2
	prioChange, secondAnchor, anchorPriority := findAnchor("job3", "job2", true, rp.taskList,
		rp.groups, rp.queuePositions, false)

	assert.Assert(t, !prioChange)
	assert.Equal(t, secondAnchor, model.JobID("job1"))
	assert.Equal(t, anchorPriority, 50)
}

func TestMoveMessagesPromoteHead(t *testing.T) {
	rp := setupRPSamePriority(t)

	// move job3 ahead of job1, the first job
	prioChange, secondAnchor, anchorPriority := findAnchor("job3", "job1", true, rp.taskList,
		rp.groups, rp.queuePositions, false)

	assert.Assert(t, !prioChange)
	assert.Equal(t, secondAnchor, sproto.HeadAnchor)
	assert.Equal(t, anchorPriority, 50)
}

func TestMoveMessagesDemote(t *testing.T) {
	rp := setupRPSamePriority(t)

	// move job1 behind job2
	prioChange, secondAnchor, anchorPriority := findAnchor("job1", "job2", false, rp.taskList,
		rp.groups, rp.queuePositions, false)

	assert.Assert(t, !prioChange)
	assert.Equal(t, secondAnchor, model.JobID("job3"))
	assert.Equal(t, anchorPriority, 50)
}

func TestMoveMessagesDemoteTail(t *testing.T) {
	rp := setupRPSamePriority(t)

	// move job1 behind job3, the last job
	prioChange, secondAnchor, anchorPriority := findAnchor("job1", "job3", false, rp.taskList,
		rp.groups, rp.queuePositions, false)

	assert.Assert(t, !prioChange)
	assert.Equal(t, secondAnchor, sproto.TailAnchor)
	assert.Equal(t, anchorPriority, 50)
}

func TestMoveMessagesAcrossPrioLanes(t *testing.T) {
	system := actor.NewSystem(t.Name())
	defaultPriority := 50
	config := config.ResourcePoolConfig{
		Scheduler: &config.SchedulerConfig{
			Priority: &config.PrioritySchedulerConfig{
				DefaultPriority: &defaultPriority,
			},
			FittingPolicy: best,
		},
	}

	rp, _ := setupResourcePool(t, nil, system, &config, nil, nil, nil)

	groupRefOne, created := system.ActorOf(actor.Addr("group1"), &mockGroup{})
	assert.Assert(t, created)
	groupRefTwo, created := system.ActorOf(actor.Addr("group2"), &mockGroup{})
	assert.Assert(t, created)
	groupRefThree, created := system.ActorOf(actor.Addr("group3"), &mockGroup{})
	assert.Assert(t, created)

	rp.queuePositions = map[model.JobID]decimal.Decimal{
		"job1": decimal.New(100, 1000),
		"job2": decimal.New(100, 1000),
		"job3": decimal.New(100, 1000),
	}

	lowPriority := 60
	highPriority := 40

	rp.groups = map[*actor.Ref]*group{
		groupRefOne:   {priority: &highPriority},
		groupRefTwo:   {priority: &defaultPriority},
		groupRefThree: {priority: &lowPriority},
	}

	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation1",
		AllocationActor: groupRefOne,
		JobID:           "job1",
		Group:           groupRefOne,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation2",
		AllocationActor: groupRefTwo,
		JobID:           "job2",
		Group:           groupRefTwo,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation3",
		AllocationActor: groupRefThree,
		JobID:           "job3",
		Group:           groupRefThree,
	})

	// move job2 ahead of job1
	prioChange, secondAnchor, anchorPriority := findAnchor("job2", "job1", true, rp.taskList,
		rp.groups, rp.queuePositions, false)

	assert.Assert(t, prioChange)
	assert.Equal(t, secondAnchor, sproto.HeadAnchor)
	assert.Equal(t, anchorPriority, 40)
}

func TestMoveMessagesAcrossPrioLanesBehind(t *testing.T) {
	system := actor.NewSystem(t.Name())
	defaultPriority := 50
	config := config.ResourcePoolConfig{
		Scheduler: &config.SchedulerConfig{
			Priority: &config.PrioritySchedulerConfig{
				DefaultPriority: &defaultPriority,
			},
			FittingPolicy: best,
		},
	}

	rp, _ := setupResourcePool(t, nil, system, &config, nil, nil, nil)

	groupRefOne, created := system.ActorOf(actor.Addr("group1"), &mockGroup{})
	assert.Assert(t, created)
	groupRefTwo, created := system.ActorOf(actor.Addr("group2"), &mockGroup{})
	assert.Assert(t, created)
	groupRefThree, created := system.ActorOf(actor.Addr("group3"), &mockGroup{})
	assert.Assert(t, created)

	rp.queuePositions = map[model.JobID]decimal.Decimal{
		"job1": decimal.New(100, 1000),
		"job2": decimal.New(100, 1000),
		"job3": decimal.New(100, 1000),
	}

	lowPriority := 60
	highPriority := 40

	rp.groups = map[*actor.Ref]*group{
		groupRefOne:   {priority: &highPriority},
		groupRefTwo:   {priority: &defaultPriority},
		groupRefThree: {priority: &lowPriority},
	}

	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation1",
		AllocationActor: groupRefOne,
		JobID:           "job1",
		Group:           groupRefOne,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation2",
		AllocationActor: groupRefTwo,
		JobID:           "job2",
		Group:           groupRefTwo,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID:    "allocation3",
		AllocationActor: groupRefThree,
		JobID:           "job3",
		Group:           groupRefThree,
	})

	// move job1 behind job2
	prioChange, secondAnchor, anchorPriority := findAnchor("job1", "job2", false, rp.taskList,
		rp.groups, rp.queuePositions, false)

	assert.Assert(t, prioChange)
	assert.Equal(t, secondAnchor, model.JobID("job3"))
	assert.Equal(t, anchorPriority, 50)
}
