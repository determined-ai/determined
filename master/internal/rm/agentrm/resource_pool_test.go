package agentrm

import (
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/rm/tasklist"

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
	agents := []*MockAgent{{ID: "agent", Slots: 1}}
	tasks := []*MockTask{{ID: "task", SlotsNeeded: 1}}
	rp, ref := setupResourcePool(t, nil, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetAllocationSummaries{}).Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, ThrowError{})
	assert.ErrorType(t, taskRef.StopAndAwaitTermination(), ErrMock)

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.Len(), 0)
}

func TestCleanUpTaskWhenTaskActorPanics(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*MockAgent{{ID: "agent", Slots: 1}}
	tasks := []*MockTask{{ID: "task", SlotsNeeded: 1}}
	rp, ref := setupResourcePool(t, nil, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetAllocationSummaries{}).Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, ThrowPanic{})
	assert.ErrorType(t, taskRef.StopAndAwaitTermination(), ErrMock)

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.Len(), 0)
}

func TestCleanUpTaskWhenTaskActorStopsNormally(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*MockAgent{{ID: "agent", Slots: 1}}
	tasks := []*MockTask{{ID: "task", SlotsNeeded: 1}}
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
	assert.Equal(t, rp.taskList.Len(), 0)
}

func TestCleanUpTaskWhenTaskActorReleaseResources(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*MockAgent{{ID: "agent", Slots: 1}}
	tasks := []*MockTask{{ID: "task", SlotsNeeded: 1}}
	rp, ref := setupResourcePool(t, nil, system, nil, tasks, nil, agents)

	taskRef := system.Get(actor.Addr("task"))
	system.Ask(taskRef, SendRequestResourcesToResourceManager{}).Get()
	taskSummaries := system.Ask(
		ref, sproto.GetAllocationSummaries{}).Get().(map[model.AllocationID]sproto.AllocationSummary)
	assert.Equal(t, len(taskSummaries), 1)

	system.Ask(taskRef, sproto.ReleaseResources{}).Get()

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.taskList.Len(), 0)
}

func TestScalingInfoAgentSummary(t *testing.T) {
	system := actor.NewSystem(t.Name())
	agents := []*MockAgent{
		{ID: "agent1", Slots: 1},
		{ID: "agent2", Slots: 1},
	}
	tasks := []*MockTask{
		{
			ID:               "allocated-cpu-task1",
			SlotsNeeded:      0,
			AllocatedAgent:   agents[0],
			ContainerStarted: true,
		},
		{
			ID:               "allocated-cpu-task2",
			SlotsNeeded:      0,
			AllocatedAgent:   agents[1],
			ContainerStarted: true,
		},
		{
			ID:               "allocated-gpu-task3",
			SlotsNeeded:      1,
			AllocatedAgent:   agents[1],
			ContainerStarted: true,
		},
		{ID: "unallocated-gpu-task4", SlotsNeeded: 1},
		{ID: "unallocated-gpu-task5", SlotsNeeded: 5},
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
	updatedPriority := 22
	jobID := model.NewJobID()
	assert.Equal(t, tasklist.GroupPriorityChangeRegistry.Add(jobID, nil), nil)
	system.Ask(ref, sproto.SetGroupPriority{Priority: updatedPriority, JobID: jobID}).Get()

	for _, n := range rp.notifications {
		<-n
	}

	assert.NilError(t, ref.StopAndAwaitTermination())
	assert.Equal(t, rp.groups[jobID] == nil, false)
	assert.Equal(t, rp.groups[jobID].Priority == nil, false)
	assert.Equal(t, *rp.groups[jobID].Priority, updatedPriority)
	assert.Equal(t, tasklist.GroupPriorityChangeRegistry.Delete(jobID), nil)
	time.Sleep(time.Second)
	assert.Equal(t, rp.groups[jobID] == nil, true)
}

func TestAddRemoveAgent(t *testing.T) {
	system := actor.NewSystem(t.Name())
	db := &mocks.DB{}
	_, ref := setupResourcePool(t, db, system, nil, nil, nil, nil)
	agentRef, created := system.ActorOf(actor.Addr("agent"), &MockAgent{ID: "agent", Slots: 2})
	assert.Assert(t, created)

	system.Tell(ref, sproto.AddAgent{Agent: agentRef})
	db.On("RecordAgentStats", mock.Anything).Return(nil)

	system.Tell(ref, sproto.RemoveAgent{Agent: agentRef})
}

func setupRPSamePriority(t *testing.T) *resourcePool {
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

	jobOne := model.JobID("job1")
	jobTwo := model.JobID("job2")
	jobThree := model.JobID("job3")

	rp.queuePositions = map[model.JobID]decimal.Decimal{
		jobOne:   decimal.New(100, 1000),
		jobTwo:   decimal.New(200, 1000),
		jobThree: decimal.New(300, 1000),
	}

	rp.groups = map[model.JobID]*tasklist.Group{
		jobOne:   {Priority: &defaultPriority},
		jobTwo:   {Priority: &defaultPriority},
		jobThree: {Priority: &defaultPriority},
	}

	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation1",
		JobID:        jobOne,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation2",
		JobID:        jobTwo,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation3",
		JobID:        jobThree,
	})

	return rp
}

func TestMoveMessagesPromote(t *testing.T) {
	rp := setupRPSamePriority(t)

	// move job3 above job2
	prioChange, secondAnchor, anchorPriority := tasklist.FindAnchor(
		"job3",
		"job2",
		true,
		rp.taskList,
		rp.groups,
		rp.queuePositions,
		false,
	)

	assert.Assert(t, !prioChange)
	assert.Equal(t, secondAnchor, model.JobID("job1"))
	assert.Equal(t, anchorPriority, 50)
}

func TestMoveMessagesPromoteHead(t *testing.T) {
	rp := setupRPSamePriority(t)

	// move job3 ahead of job1, the first job
	prioChange, secondAnchor, anchorPriority := tasklist.FindAnchor(
		"job3",
		"job1",
		true,
		rp.taskList,
		rp.groups,
		rp.queuePositions,
		false,
	)

	assert.Assert(t, !prioChange)
	assert.Equal(t, secondAnchor, sproto.HeadAnchor)
	assert.Equal(t, anchorPriority, 50)
}

func TestMoveMessagesDemote(t *testing.T) {
	rp := setupRPSamePriority(t)

	// move job1 behind job2
	prioChange, secondAnchor, anchorPriority := tasklist.FindAnchor(
		"job1",
		"job2",
		false,
		rp.taskList,
		rp.groups,
		rp.queuePositions,
		false,
	)

	assert.Assert(t, !prioChange)
	assert.Equal(t, secondAnchor, model.JobID("job3"))
	assert.Equal(t, anchorPriority, 50)
}

func TestMoveMessagesDemoteTail(t *testing.T) {
	rp := setupRPSamePriority(t)

	// move job1 behind job3, the last job
	prioChange, secondAnchor, anchorPriority := tasklist.FindAnchor(
		"job1",
		"job3",
		false,
		rp.taskList,
		rp.groups,
		rp.queuePositions,
		false,
	)

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

	jobOne := model.JobID("job1")
	jobTwo := model.JobID("job2")
	jobThree := model.JobID("job3")

	rp.queuePositions = map[model.JobID]decimal.Decimal{
		jobOne:   decimal.New(100, 1000),
		jobTwo:   decimal.New(100, 1000),
		jobThree: decimal.New(100, 1000),
	}

	lowPriority := 60
	highPriority := 40

	rp.groups = map[model.JobID]*tasklist.Group{
		jobOne:   {Priority: &highPriority},
		jobTwo:   {Priority: &defaultPriority},
		jobThree: {Priority: &lowPriority},
	}

	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation1",
		JobID:        jobOne,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation2",
		JobID:        jobTwo,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation3",
		JobID:        jobThree,
	})

	// move job2 ahead of job1
	prioChange, secondAnchor, anchorPriority := tasklist.FindAnchor(
		"job2",
		"job1",
		true,
		rp.taskList,
		rp.groups,
		rp.queuePositions,
		false,
	)

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

	jobOne := model.JobID("job1")
	jobTwo := model.JobID("job2")
	jobThree := model.JobID("job3")

	rp.queuePositions = map[model.JobID]decimal.Decimal{
		jobOne:   decimal.New(100, 1000),
		jobTwo:   decimal.New(100, 1000),
		jobThree: decimal.New(100, 1000),
	}

	lowPriority := 60
	highPriority := 40

	rp.groups = map[model.JobID]*tasklist.Group{
		jobOne:   {Priority: &highPriority},
		jobTwo:   {Priority: &defaultPriority},
		jobThree: {Priority: &lowPriority},
	}

	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation1",
		JobID:        jobOne,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation2",
		JobID:        jobTwo,
	})
	rp.taskList.AddTask(&sproto.AllocateRequest{
		AllocationID: "allocation3",
		JobID:        jobThree,
	})

	// move job1 behind job2
	prioChange, secondAnchor, anchorPriority := tasklist.FindAnchor(
		"job1",
		"job2",
		false,
		rp.taskList,
		rp.groups,
		rp.queuePositions,
		false,
	)

	assert.Assert(t, prioChange)
	assert.Equal(t, secondAnchor, model.JobID("job3"))
	assert.Equal(t, anchorPriority, 50)
}
