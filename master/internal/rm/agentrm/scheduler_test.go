package agentrm

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/internal/rm/tasklist"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"

	"github.com/google/uuid"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
)

func newMaxSlot(maxSlot int) *int {
	return &maxSlot
}

func setupResourcePool(
	t *testing.T,
	db db.DB,
	system *actor.System,
	conf *config.ResourcePoolConfig,
	mockTasks []*MockTask,
	mockGroups []*MockGroup,
	mockAgents []*MockAgent,
) (*resourcePool, *actor.Ref) {
	if conf == nil {
		conf = &config.ResourcePoolConfig{PoolName: "pool"}
	}
	if conf.Scheduler == nil {
		conf.Scheduler = &config.SchedulerConfig{
			FairShare:     &config.FairShareSchedulerConfig{},
			FittingPolicy: best,
		}
	}

	rp := newResourcePool(
		conf, db, nil, MakeScheduler(conf.Scheduler),
		MakeFitFunction(conf.Scheduler.FittingPolicy))
	rp.taskList, rp.groups, rp.agentStatesCache = setupSchedulerStates(
		t, system, mockTasks, mockGroups, mockAgents,
	)
	rp.saveNotifications = true
	ref, created := system.ActorOf(actor.Addr(rp.config.PoolName), rp)
	assert.Assert(t, created)
	system.Ask(ref, actor.Ping{}).Get()

	for _, task := range mockTasks {
		task.RMRef = ref
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
	ref, created := system.ActorOf(actor.Addr(agentID), &MockAgent{ID: agentID, Slots: numSlots})
	assert.Assert(t, created)
	state := newAgentState(sproto.AddAgent{Agent: ref}, 100)
	for i := 0; i < numSlots; i++ {
		state.Devices[device.Device{ID: device.ID(i)}] = nil
	}
	i := 0
	for ix := range state.Devices {
		if i < numUsedSlots {
			id := cproto.ID(uuid.New().String())
			state.Devices[ix] = &id
		}
	}
	for i := 0; i < numZeroSlotContainers; i++ {
		_, err := state.allocateFreeDevices(0, cproto.NewID())
		assert.NilError(t, err)
	}
	agents[state.Handler] = state
	return state
}

func newFakeAgentState(
	t *testing.T,
	system *actor.System,
	id string,
	slots int,
	slotsUsed int,
	maxZeroSlotContainers int,
	zeroSlotContainers int,
) *agentState {
	ref, created := system.ActorOf(actor.Addr(id), &MockAgent{ID: id, Slots: slots})
	assert.Assert(t, created)
	state := newAgentState(sproto.AddAgent{Agent: ref}, maxZeroSlotContainers)
	for i := 0; i < slots; i++ {
		state.Devices[device.Device{ID: device.ID(i)}] = nil
	}

	if slotsUsed > 0 {
		req := &sproto.AllocateRequest{
			SlotsNeeded: slotsUsed,
			Preemptible: true,
		}
		if _, err := state.allocateFreeDevices(req.SlotsNeeded, cproto.NewID()); err != nil {
			panic(err)
		}
	}

	for i := 0; i < zeroSlotContainers; i++ {
		req := &sproto.AllocateRequest{}
		if _, err := state.allocateFreeDevices(req.SlotsNeeded, cproto.NewID()); err != nil {
			panic(err)
		}
	}
	return state
}

func forceAddTask(
	t *testing.T,
	system *actor.System,
	taskList *tasklist.TaskList,
	taskID string,
	numAllocated int,
	slotsNeeded int,
) {
	task := &MockTask{ID: model.AllocationID(taskID), SlotsNeeded: slotsNeeded}
	ref, created := system.ActorOf(actor.Addr(taskID), task)
	assert.Assert(t, created)

	req := &sproto.AllocateRequest{
		AllocationID:  model.AllocationID(taskID),
		AllocationRef: ref,
		Group:         ref,
		SlotsNeeded:   slotsNeeded,
	}
	taskList.AddTask(req)
	forceSetTaskAllocations(t, taskList, taskID, numAllocated)
}

func forceSetTaskAllocations(
	t *testing.T,
	taskList *tasklist.TaskList,
	taskID string,
	numAllocated int,
) {
	req, ok := taskList.TaskByID(model.AllocationID(taskID))
	assert.Check(t, ok)
	if numAllocated > 0 {
		allocated := &sproto.ResourcesAllocated{
			ID:        model.AllocationID(taskID),
			Resources: map[sproto.ResourcesID]sproto.Resources{},
		}
		for i := 0; i < numAllocated; i++ {
			allocated.Resources[sproto.ResourcesID(uuid.NewString())] = containerResources{}
		}
		taskList.AddAllocation(req.AllocationID, allocated)
	} else {
		taskList.AddAllocation(req.AllocationID, nil)
	}
}

func assertEqualToAllocate(
	t *testing.T,
	actual []*sproto.AllocateRequest,
	expected []*MockTask,
) {
	expectedMap := map[model.AllocationID]bool{}
	for _, task := range expected {
		expectedMap[task.ID] = true
	}
	for _, task := range actual {
		_, ok := expectedMap[task.AllocationID]
		assert.Assert(t, ok)
	}
	assert.Equal(t, len(actual), len(expected),
		"actual allocated tasks and expected tasks must have the same length")
}

func assertEqualToAllocateOrdered(
	t *testing.T,
	actual []*sproto.AllocateRequest,
	expected []*MockTask,
) {
	assert.Equal(t, len(actual), len(expected),
		"actual tasks and expected tasks must have the same length")
	for i := range expected {
		assert.Equal(t, expected[i].ID, actual[i].AllocationID)
	}
}

func assertEqualToRelease(
	t *testing.T,
	taskList *tasklist.TaskList,
	actual []*actor.Ref,
	expected []*MockTask,
) {
	expectedMap := map[model.AllocationID]bool{}
	for _, task := range expected {
		expectedMap[task.ID] = true
	}
	for _, taskActor := range actual {
		// HACK: Holdover until the scheduler interface doesn't have actors.
		var task *sproto.AllocateRequest
		for it := taskList.Iterator(); it.Next(); {
			req := it.Value()
			if req.AllocationRef == taskActor {
				task = req
				break
			}
		}
		assert.Assert(t, task != nil)

		if task != nil {
			_, ok := expectedMap[task.AllocationID]
			assert.Assert(t, ok)
		}
	}
	assert.Equal(t, len(actual), len(expected),
		"actual released tasks and expected tasks must have the same length")
}

func TestJobStats(t *testing.T) {
	prepMockData := func() ([]*MockTask, []*MockGroup, []*MockAgent) {
		lowerPriority := 50
		higherPriority := 40

		agents := []*MockAgent{
			{ID: "agent1", Slots: 1, MaxZeroSlotContainers: 1},
		}
		groups := []*MockGroup{
			{ID: "group1", Priority: &lowerPriority, Weight: 0.5},
			{ID: "group2", Priority: &higherPriority, Weight: 1},
		}
		tasks := []*MockTask{
			{ID: "task1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
			{ID: "task2", JobID: "job2", SlotsNeeded: 1, Group: groups[1]},
			{ID: "task3", JobID: "job3", SlotsNeeded: 0, Group: groups[0]},
			{ID: "task4", JobID: "job4", SlotsNeeded: 0, Group: groups[0]},
		}

		return tasks, groups, agents
	}

	assertStatsEqual := func(
		t *testing.T,
		actual *jobv1.QueueStats,
		expected *jobv1.QueueStats,
	) {
		assert.Equal(t, actual.QueuedCount, expected.QueuedCount)
		assert.Equal(t, actual.ScheduledCount, expected.ScheduledCount)
	}

	testPriority := func(
		t *testing.T,
		tasks []*MockTask,
		groups []*MockGroup,
		agents []*MockAgent,
		expectedStats *jobv1.QueueStats,
	) {
		p := &priorityScheduler{}
		system := actor.NewSystem(t.Name())
		taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
		toAllocate, _ := p.prioritySchedule(taskList, groupMap,
			make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
		AllocateTasks(toAllocate, agentMap, taskList)
		p.prioritySchedule(taskList, groupMap,
			make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
		assertStatsEqual(t, tasklist.JobStats(taskList), expectedStats)
	}
	testFairshare := func(
		t *testing.T,
		tasks []*MockTask,
		groups []*MockGroup,
		agents []*MockAgent,
		expectedStats *jobv1.QueueStats,
	) {
		system := actor.NewSystem(t.Name())
		taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
		toAllocate, _ := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
		AllocateTasks(toAllocate, agentMap, taskList)
		fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)

		assertStatsEqual(t, tasklist.JobStats(taskList), expectedStats)
	}

	tasks, groups, agents := prepMockData()
	testPriority(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(2), ScheduledCount: int32(2)})

	tasks, groups, agents = prepMockData()
	testFairshare(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(2), ScheduledCount: int32(2)})

	_, groups, agents = prepMockData()
	tasks = []*MockTask{
		{ID: "task1.1", JobID: "job1", SlotsNeeded: 2, Group: groups[0]}, // same job
		{ID: "task1.2", JobID: "job1", SlotsNeeded: 2, Group: groups[0]},
		{ID: "task1.3", JobID: "job1", SlotsNeeded: 2, Group: groups[0]},
		{ID: "task2", JobID: "job2", SlotsNeeded: 2, Group: groups[1]},
		{ID: "task3", JobID: "job3", SlotsNeeded: 2, Group: groups[0]},
		{ID: "task4", JobID: "job4", SlotsNeeded: 2, Group: groups[0]},
	}
	testPriority(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(4), ScheduledCount: int32(0)})

	_, groups, agents = prepMockData()
	tasks = []*MockTask{
		{ID: "task1.1", JobID: "job1", SlotsNeeded: 2, Group: groups[0]}, // same job
		{ID: "task1.2", JobID: "job1", SlotsNeeded: 2, Group: groups[0]},
		{ID: "task1.3", JobID: "job1", SlotsNeeded: 2, Group: groups[0]},
		{ID: "task2", JobID: "job2", SlotsNeeded: 2, Group: groups[1]},
		{ID: "task3", JobID: "job3", SlotsNeeded: 2, Group: groups[0]},
		{ID: "task4", JobID: "job4", SlotsNeeded: 2, Group: groups[0]},
	}
	testFairshare(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(4), ScheduledCount: int32(0)})
}

func TestJobOrder(t *testing.T) {
	prepMockData := func() ([]*MockGroup, []*MockAgent) {
		lowerPriority := 50
		higherPriority := 40

		agents := []*MockAgent{
			{ID: "agent1", Slots: 1, MaxZeroSlotContainers: 1},
		}
		groups := []*MockGroup{
			{ID: "group1", Priority: &lowerPriority, Weight: 0.5},
			{ID: "group2", Priority: &higherPriority, Weight: 1},
		}

		return groups, agents
	}

	setupPriority := func(
		tasks []*MockTask,
		groups []*MockGroup,
		agents []*MockAgent,
	) map[model.JobID]*sproto.RMJobInfo {
		p := &priorityScheduler{preemptionEnabled: false}
		system := actor.NewSystem(t.Name())
		taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
		toAllocate, _ := p.prioritySchedule(taskList, groupMap,
			make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
		AllocateTasks(toAllocate, agentMap, taskList)
		return p.JobQInfo(&resourcePool{taskList: taskList, groups: groupMap})
	}

	setupFairshare := func(
		tasks []*MockTask,
		groups []*MockGroup,
		agents []*MockAgent,
	) map[model.JobID]*sproto.RMJobInfo {
		system := actor.NewSystem(t.Name())
		taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
		toAllocate, _ := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
		AllocateTasks(toAllocate, agentMap, taskList)
		fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
		f := fairShare{}
		return f.JobQInfo(&resourcePool{taskList: taskList, groups: groupMap})
	}

	groups, agents := prepMockData()
	tasks := []*MockTask{
		{ID: "task1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task1.1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task2", JobID: "job2", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task3", JobID: "job3", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task4", JobID: "job4", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task4.1", JobID: "job4", SlotsNeeded: 0, Group: groups[0]},
	}
	jobInfo := setupPriority(tasks, groups, agents)
	assert.Equal(t, len(jobInfo), 4)
	assert.Equal(t, jobInfo["job2"].State, sproto.SchedulingStateScheduled)
	assert.Equal(t, jobInfo["job2"].JobsAhead, 0)
	assert.Equal(t, jobInfo["job2"].AllocatedSlots, 1)
	assert.Equal(t, jobInfo["job2"].RequestedSlots, 1)
	assert.Equal(t, jobInfo["job1"].State, sproto.SchedulingStateQueued)
	assert.Equal(t, jobInfo["job1"].AllocatedSlots, 0)
	assert.Equal(t, jobInfo["job1"].JobsAhead, 1)
	assert.Equal(t, jobInfo["job3"].JobsAhead, 2)
	assert.Equal(t, jobInfo["job4"].JobsAhead, 3)

	groups, agents = prepMockData()
	tasks = []*MockTask{
		{ID: "task1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task1.1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task2", JobID: "job2", SlotsNeeded: 1, Group: groups[1]},
		{ID: "task3", JobID: "job3", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task4", JobID: "job4", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task4.1", JobID: "job4", SlotsNeeded: 0, Group: groups[0]},
	}
	jobInfo = setupFairshare(tasks, groups, agents)
	assert.Equal(t, len(jobInfo), 4)
	assert.Equal(t, jobInfo["job2"].JobsAhead, -1)
	assert.Equal(t, jobInfo["job1"].JobsAhead, -1)
	assert.Equal(t, jobInfo["job4"].State, sproto.SchedulingStateScheduled)
}

func TestJobOrderPriority(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*MockAgent{
		{ID: "agent1", Slots: 1, MaxZeroSlotContainers: 1},
	}
	groups := []*MockGroup{
		{ID: "group1", Priority: &lowerPriority, Weight: 0.5},
		{ID: "group2", Priority: &higherPriority, Weight: 1},
	}

	tasks := []*MockTask{
		{ID: "task1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task1.1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
	}

	p := &priorityScheduler{preemptionEnabled: false}
	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	AllocateTasks(toAllocate, agentMap, taskList)
	jobInfo := p.JobQInfo(&resourcePool{taskList: taskList, groups: groupMap})
	assert.Equal(t, len(jobInfo), 1)
	assert.Equal(t, jobInfo["job1"].State, sproto.SchedulingStateScheduled)
	assert.Equal(t, jobInfo["job1"].AllocatedSlots, 1)
	assert.Equal(t, jobInfo["job1"].JobsAhead, 0)

	newTasks := []*MockTask{
		{ID: "task2", JobID: "job2", SlotsNeeded: 1, Group: groups[1]},
	}

	AddUnallocatedTasks(t, newTasks, system, taskList)
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	assert.Equal(t, len(toRelease), 0)
	AllocateTasks(toAllocate, agentMap, taskList)
	jobInfo = p.JobQInfo(&resourcePool{taskList: taskList, groups: groupMap})
	assert.Equal(t, len(jobInfo), 2)
	assert.Equal(t, jobInfo["job1"].State, sproto.SchedulingStateScheduled)
	assert.Equal(t, jobInfo["job1"].AllocatedSlots, 1)
	assert.Equal(t, jobInfo["job1"].JobsAhead, 1)
	assert.Equal(t, jobInfo["job2"].JobsAhead, 0)
	assert.Equal(t, jobInfo["job2"].AllocatedSlots, 0)
	assert.Equal(t, jobInfo["job2"].State, sproto.SchedulingStateQueued)
}

func setupSchedulerStates(
	t *testing.T,
	system *actor.System,
	mockTasks []*MockTask,
	mockGroups []*MockGroup,
	mockAgents []*MockAgent,
) (
	*tasklist.TaskList,
	map[*actor.Ref]*tasklist.Group,
	map[*actor.Ref]*agentState,
) {
	agents := make(map[*actor.Ref]*agentState, len(mockAgents))
	for _, mockAgent := range mockAgents {
		ref, created := system.ActorOf(actor.Addr(mockAgent.ID), mockAgent)
		assert.Assert(t, created)

		agent := newAgentState(sproto.AddAgent{
			Agent: ref,
		}, mockAgent.MaxZeroSlotContainers)

		for i := 0; i < mockAgent.Slots; i++ {
			agent.Devices[device.Device{ID: device.ID(i)}] = nil
		}
		agents[ref] = agent
	}

	groups := make(map[*actor.Ref]*tasklist.Group, len(mockGroups))
	groupActors := make(map[*MockGroup]*actor.Ref, len(mockGroups))
	for _, mockGroup := range mockGroups {
		ref, created := system.ActorOf(actor.Addr(mockGroup.ID), mockGroup)
		assert.Assert(t, created)

		group := &tasklist.Group{
			Handler:  ref,
			MaxSlots: mockGroup.MaxSlots,
			Weight:   mockGroup.Weight,
			Priority: mockGroup.Priority,
		}
		groups[ref] = group
		groupActors[mockGroup] = ref
	}

	taskList := tasklist.New()
	for _, mockTask := range mockTasks {
		ref, created := system.ActorOf(actor.Addr(mockTask.ID), mockTask)
		system.Ask(ref, actor.Ping{})
		assert.Assert(t, created)

		groups[ref] = &tasklist.Group{Handler: ref}

		req := MockTaskToAllocateRequest(mockTask, ref)
		if mockTask.Group == nil {
			req.Group = ref
		} else {
			req.Group = groupActors[mockTask.Group]
		}
		taskList.AddTask(req)

		if mockTask.AllocatedAgent != nil {
			assert.Assert(t, mockTask.AllocatedAgent.Slots >= mockTask.SlotsNeeded)
			agentRef := system.Get(actor.Addr(mockTask.AllocatedAgent.ID))
			agentState := agents[agentRef]
			containerID := cproto.NewID()

			devices := make([]device.Device, 0)
			if mockTask.ContainerStarted {
				if mockTask.SlotsNeeded == 0 {
					_, err := agentState.allocateFreeDevices(0, containerID)
					assert.NilError(t, err)
				} else {
					i := 0
					for d, currContainerID := range agentState.Devices {
						if currContainerID != nil {
							continue
						}
						if i < mockTask.SlotsNeeded {
							agentState.Devices[d] = &containerID
							devices = append(devices, d)
							i++
						}
					}
					assert.Assert(t, i == mockTask.SlotsNeeded,
						"over allocated to agent %s", mockTask.AllocatedAgent.ID)
				}
			}

			allocated := &sproto.ResourcesAllocated{
				ID: req.AllocationID,
				Resources: map[sproto.ResourcesID]sproto.Resources{
					sproto.ResourcesID(containerID): &containerResources{
						req:         req,
						agent:       agentState,
						containerID: containerID,
						devices:     devices,
					},
				},
			}
			taskList.AddAllocation(req.AllocationID, allocated)
		}
	}

	return taskList, groups, agents
}
