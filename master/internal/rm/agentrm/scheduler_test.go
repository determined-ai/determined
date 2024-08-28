package agentrm

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

func newMaxSlot(maxSlot int) *int {
	return &maxSlot
}

func setupResourcePool(
	t *testing.T,
	db db.DB,
	conf *config.ResourcePoolConfig,
	mockTasks []*MockTask,
	mockGroups []*MockGroup,
	mockAgents []*MockAgent,
) *resourcePool {
	if conf == nil {
		conf = &config.ResourcePoolConfig{PoolName: "pool"}
	}
	if conf.Scheduler == nil {
		conf.Scheduler = &config.SchedulerConfig{
			FairShare:     &config.FairShareSchedulerConfig{},
			FittingPolicy: best,
		}
	}

	agentsRef, _ := newAgentService([]config.ResourcePoolConfig{*conf}, &aproto.MasterSetAgentOptions{})

	scheduler, err := MakeScheduler(conf.Scheduler)
	require.NoError(t, err)
	rp, err := newResourcePool(
		conf, db, nil, scheduler,
		MakeFitFunction(conf.Scheduler.FittingPolicy), agentsRef)
	require.NoError(t, err)
	rp.taskList, rp.groups, rp.agentStatesCache = setupSchedulerStates(
		t, mockTasks, mockGroups, mockAgents,
	)
	rp.saveNotifications = true

	for _, task := range mockTasks {
		task.RPRef = rp
	}
	return rp
}

func forceAddAgent(
	t *testing.T,
	agents map[aproto.ID]*agentState,
	agentIDStr string,
	numSlots int,
	numUsedSlots int,
	numZeroSlotContainers int,
) *agentState {
	state := newAgentState(aproto.ID(agentIDStr), 100)
	state.handler = &agent{}
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
	agents[state.id] = state
	return state
}

func newFakeAgentState(
	t *testing.T,
	id string,
	slots int,
	slotsUsed int,
	maxZeroSlotContainers int,
	zeroSlotContainers int,
) *agentState {
	state := newAgentState(aproto.ID(id), maxZeroSlotContainers)
	state.handler = &agent{}
	for i := 0; i < slots; i++ {
		state.Devices[device.Device{ID: device.ID(i)}] = nil
	}

	if slotsUsed > 0 {
		req := &sproto.AllocateRequest{
			SlotsNeeded: slotsUsed,
			Preemption: sproto.PreemptionConfig{
				Preemptible: true,
			},
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
	taskList *tasklist.TaskList,
	taskID string,
	numAllocated int,
	slotsNeeded int,
) {
	req := &sproto.AllocateRequest{
		AllocationID: model.AllocationID(taskID),
		JobID:        model.JobID(taskID),
		SlotsNeeded:  slotsNeeded,
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
		t.Log("expected task", task.ID, "to be allocated")
		expectedMap[task.ID] = true
	}
	for _, task := range actual {
		t.Log("have task", task.AllocationID, "to be allocated")
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
	actual []model.AllocationID,
	expected []*MockTask,
) {
	expectedMap := map[model.AllocationID]bool{}
	for _, task := range expected {
		expectedMap[task.ID] = true
	}
	for _, allocationID := range actual {
		// HACK: Holdover until the scheduler interface doesn't have actors.
		var task *sproto.AllocateRequest
		for it := taskList.Iterator(); it.Next(); {
			req := it.Value()
			if req.AllocationID == allocationID {
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
			{ID: "job1", Priority: &lowerPriority, Weight: 0.5},
			{ID: "job2", Priority: &higherPriority, Weight: 1},
			{ID: "job3", Priority: &lowerPriority, Weight: 0},
			{ID: "job4", Priority: &lowerPriority, Weight: 0},
		}
		tasks := []*MockTask{
			{ID: "task1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
			{ID: "task2", JobID: "job2", SlotsNeeded: 1, Group: groups[1]},
			{ID: "task3", JobID: "job3", SlotsNeeded: 0, Group: groups[2]},
			{ID: "task4", JobID: "job4", SlotsNeeded: 0, Group: groups[3]},
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
		taskList, groupMap, agentMap := setupSchedulerStates(t, tasks, groups, agents)
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
		taskList, groupMap, agentMap := setupSchedulerStates(t, tasks, groups, agents)
		toAllocate, _ := fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)
		AllocateTasks(toAllocate, agentMap, taskList)
		fairshareSchedule(taskList, groupMap, agentMap, BestFit, false)

		assertStatsEqual(t, tasklist.JobStats(taskList), expectedStats)
	}

	t.Log("calling testPriority 1")
	tasks, groups, agents := prepMockData()
	testPriority(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(2), ScheduledCount: int32(2)})

	t.Log("calling testFairshare 1")
	tasks, groups, agents = prepMockData()
	testFairshare(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(2), ScheduledCount: int32(2)})

	t.Log("calling testPriority 2")
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

	t.Log("calling testFairshare 2")
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
			{ID: "job1", Priority: &lowerPriority, Weight: 0.5},
			{ID: "job2", Priority: &higherPriority, Weight: 1},
			{ID: "job3", Priority: &lowerPriority, Weight: 0},
			{ID: "job4", Priority: &lowerPriority, Weight: 0},
		}

		return groups, agents
	}

	setupPriority := func(
		tasks []*MockTask,
		groups []*MockGroup,
		agents []*MockAgent,
	) map[model.JobID]*sproto.RMJobInfo {
		p := &priorityScheduler{preemptionEnabled: false}
		taskList, groupMap, agentMap := setupSchedulerStates(t, tasks, groups, agents)
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
		taskList, groupMap, agentMap := setupSchedulerStates(t, tasks, groups, agents)
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
		{ID: "task3", JobID: "job3", SlotsNeeded: 1, Group: groups[2]},
		{ID: "task4", JobID: "job4", SlotsNeeded: 1, Group: groups[3]},
		{ID: "task4.1", JobID: "job4", SlotsNeeded: 0, Group: groups[3]},
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
		{ID: "job1", Priority: &lowerPriority, Weight: 0.5},
		{ID: "job2", Priority: &higherPriority, Weight: 1},
	}

	tasks := []*MockTask{
		{ID: "task1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
		{ID: "task1.1", JobID: "job1", SlotsNeeded: 1, Group: groups[0]},
	}

	p := &priorityScheduler{preemptionEnabled: false}
	taskList, groupMap, agentMap := setupSchedulerStates(t, tasks, groups, agents)
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

	AddUnallocatedTasks(t, newTasks, taskList)
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
	mockTasks []*MockTask,
	mockGroups []*MockGroup,
	mockAgents []*MockAgent,
) (
	*tasklist.TaskList,
	map[model.JobID]*tasklist.Group,
	map[aproto.ID]*agentState,
) {
	agents := make(map[aproto.ID]*agentState, len(mockAgents))
	for _, mockAgent := range mockAgents {
		state := newAgentState(aproto.ID(mockAgent.ID), mockAgent.MaxZeroSlotContainers)
		state.handler = &agent{}

		for i := 0; i < mockAgent.Slots; i++ {
			state.Devices[device.Device{ID: device.ID(i)}] = nil
		}
		agents[aproto.ID(mockAgent.ID)] = state
	}

	groups := make(map[model.JobID]*tasklist.Group, len(mockGroups))
	for _, mockGroup := range mockGroups {
		group := &tasklist.Group{
			JobID:    model.JobID(mockGroup.ID),
			MaxSlots: mockGroup.MaxSlots,
			Weight:   mockGroup.Weight,
			Priority: mockGroup.Priority,
		}
		groups[model.JobID(mockGroup.ID)] = group
	}

	taskList := tasklist.New()
	for _, mockTask := range mockTasks {
		jobID := model.JobID(mockTask.JobID)
		if jobID == "" {
			if mockTask.Group != nil {
				jobID = model.JobID(mockTask.Group.ID)
			} else {
				jobID = model.JobID(mockTask.ID)
			}
			mockTask.JobID = string(jobID)
		}
		if _, ok := groups[jobID]; !ok {
			groups[jobID] = &tasklist.Group{JobID: jobID}
		}

		req := MockTaskToAllocateRequest(mockTask)
		taskList.AddTask(req)

		if mockTask.AllocatedAgent != nil {
			assert.Assert(t, mockTask.AllocatedAgent.Slots >= mockTask.SlotsNeeded)
			agentState := agents[aproto.ID(mockTask.AllocatedAgent.ID)]
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

func TestRoundRobinResourcePoolDeprecation(t *testing.T) {
	conf := &config.ResourcePoolConfig{PoolName: "pool"}
	conf.Scheduler = &config.SchedulerConfig{
		RoundRobin:    &config.RoundRobinSchedulerConfig{},
		FittingPolicy: best,
	}

	_, err := MakeScheduler(conf.Scheduler)
	require.Error(t, err)
}
