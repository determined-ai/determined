package rm

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"

	"github.com/google/uuid"

	"github.com/pkg/errors"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

func newMaxSlot(maxSlot int) *int {
	return &maxSlot
}

type mockGroup struct {
	id       string
	maxSlots *int
	weight   float64
	priority *int
}

func (g *mockGroup) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart:
	case actor.PostStop:
	case *sproto.RMJobInfo:
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

	id             model.AllocationID
	jobID          string
	group          *mockGroup
	slotsNeeded    int
	nonPreemptible bool
	label          string
	resourcePool   string
	allocatedAgent *mockAgent
	// Any test that set this to false is half wrong. It is used as a proxy to oversubscribe agents.
	containerStarted  bool
	jobSubmissionTime time.Time
}

func (t *mockTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
	case actor.PostStop:
	case SendRequestResourcesToResourceManager:
		task := sproto.AllocateRequest{
			AllocationID:      t.id,
			JobID:             model.JobID(t.jobID),
			JobSubmissionTime: t.jobSubmissionTime,
			Name:              string(t.id),
			SlotsNeeded:       t.slotsNeeded,
			Preemptible:       !t.nonPreemptible,
			Label:             t.label,
			ResourcePool:      t.resourcePool,
			AllocationRef:     ctx.Self(),
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
		task := sproto.ResourcesReleased{AllocationRef: ctx.Self()}
		if ctx.ExpectingResponse() {
			ctx.Respond(ctx.Ask(t.rmRef, task).Get())
		} else {
			ctx.Tell(t.rmRef, task)
		}
	case ThrowError:
		return errMock
	case ThrowPanic:
		panic(errMock)

	case sproto.ResourcesAllocated:
		rank := 0
		for _, allocation := range msg.Resources {
			if err := allocation.Start(ctx, nil, tasks.TaskSpec{}, sproto.ResourcesRuntimeInfo{
				Token:        "",
				AgentRank:    rank,
				IsMultiAgent: len(msg.Resources) > 1,
			}); err != nil {
				ctx.Respond(err)
				return nil
			}
			rank++
		}
	case sproto.ReleaseResources:
		ctx.Tell(t.rmRef, sproto.ResourcesReleased{AllocationRef: ctx.Self()})

	case sproto.ResourcesStateChanged:

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
	maxZeroSlotContainers int
	zeroSlotContainers    int
}

func newMockAgent(
	id string,
	label string,
	slots int,
	slotsUsed int,
	maxZeroSlotContainers int,
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
	db db.DB,
	system *actor.System,
	conf *config.ResourcePoolConfig,
	mockTasks []*mockTask,
	mockGroups []*mockGroup,
	mockAgents []*mockAgent,
) (*ResourcePool, *actor.Ref) {
	if conf == nil {
		conf = &config.ResourcePoolConfig{PoolName: "pool"}
	}
	if conf.Scheduler == nil {
		conf.Scheduler = &config.SchedulerConfig{
			FairShare:     &config.FairShareSchedulerConfig{},
			FittingPolicy: best,
		}
	}

	rp := NewResourcePool(
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
		task.rmRef = ref
	}
	return rp, ref
}

func forceAddAgent(
	t *testing.T,
	system *actor.System,
	agents map[*actor.Ref]*AgentState,
	agentID string,
	numSlots int,
	numUsedSlots int,
	numZeroSlotContainers int,
) *AgentState {
	ref, created := system.ActorOf(actor.Addr(agentID), &mockAgent{id: agentID, slots: numSlots})
	assert.Assert(t, created)
	state := NewAgentState(sproto.AddAgent{Agent: ref}, 100)
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
		_, err := state.AllocateFreeDevices(0, cproto.NewID())
		assert.NilError(t, err)
	}
	agents[state.Handler] = state
	return state
}

func newFakeAgentState(
	t *testing.T,
	system *actor.System,
	id string,
	label string,
	slots int,
	slotsUsed int,
	maxZeroSlotContainers int,
	zeroSlotContainers int,
) *AgentState {
	ref, created := system.ActorOf(actor.Addr(id), &mockAgent{id: id, slots: slots, label: label})
	assert.Assert(t, created)
	state := NewAgentState(sproto.AddAgent{Agent: ref, Label: label}, maxZeroSlotContainers)
	for i := 0; i < slots; i++ {
		state.Devices[device.Device{ID: device.ID(i)}] = nil
	}

	if slotsUsed > 0 {
		req := &sproto.AllocateRequest{
			SlotsNeeded: slotsUsed,
			Preemptible: true,
		}
		if _, err := state.AllocateFreeDevices(req.SlotsNeeded, cproto.NewID()); err != nil {
			panic(err)
		}
	}

	for i := 0; i < zeroSlotContainers; i++ {
		req := &sproto.AllocateRequest{}
		if _, err := state.AllocateFreeDevices(req.SlotsNeeded, cproto.NewID()); err != nil {
			panic(err)
		}
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
	task := &mockTask{id: model.AllocationID(taskID), slotsNeeded: slotsNeeded}
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
	taskList *taskList,
	taskID string,
	numAllocated int,
) {
	req, ok := taskList.GetTaskByID(model.AllocationID(taskID))
	assert.Check(t, ok)
	if numAllocated > 0 {
		allocated := &sproto.ResourcesAllocated{
			ID:        model.AllocationID(taskID),
			Resources: map[sproto.ResourcesID]sproto.Resources{},
		}
		for i := 0; i < numAllocated; i++ {
			allocated.Resources[sproto.ResourcesID(uuid.NewString())] = containerResources{}
		}
		taskList.SetAllocations(req.AllocationRef, allocated)
	} else {
		taskList.SetAllocations(req.AllocationRef, nil)
	}
}

func mockTaskToAllocateRequest(
	mockTask *mockTask, allocationRef *actor.Ref,
) *sproto.AllocateRequest {
	jobID := mockTask.jobID
	jobSubmissionTime := mockTask.jobSubmissionTime

	if jobID == "" {
		jobID = string(mockTask.id)
	}
	if jobSubmissionTime.IsZero() {
		jobSubmissionTime = allocationRef.RegisteredTime()
	}

	req := &sproto.AllocateRequest{
		AllocationID:      mockTask.id,
		JobID:             model.JobID(jobID),
		SlotsNeeded:       mockTask.slotsNeeded,
		Label:             mockTask.label,
		IsUserVisible:     true,
		AllocationRef:     allocationRef,
		Preemptible:       !mockTask.nonPreemptible,
		JobSubmissionTime: jobSubmissionTime,
	}
	return req
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
	map[*actor.Ref]*AgentState,
) {
	agents := make(map[*actor.Ref]*AgentState, len(mockAgents))
	for _, mockAgent := range mockAgents {
		ref, created := system.ActorOf(actor.Addr(mockAgent.id), mockAgent)
		assert.Assert(t, created)

		agent := NewAgentState(sproto.AddAgent{
			Agent: ref,
			Label: mockAgent.label,
		}, mockAgent.maxZeroSlotContainers)

		for i := 0; i < mockAgent.slots; i++ {
			agent.Devices[device.Device{ID: device.ID(i)}] = nil
		}
		agents[ref] = agent
	}

	groups := make(map[*actor.Ref]*group, len(mockGroups))
	groupActors := make(map[*mockGroup]*actor.Ref, len(mockGroups))
	for _, mockGroup := range mockGroups {
		ref, created := system.ActorOf(actor.Addr(mockGroup.id), mockGroup)
		assert.Assert(t, created)

		group := &group{
			handler:  ref,
			maxSlots: mockGroup.maxSlots,
			weight:   mockGroup.weight,
			priority: mockGroup.priority,
		}
		groups[ref] = group
		groupActors[mockGroup] = ref
	}

	taskList := newTaskList()
	for _, mockTask := range mockTasks {
		ref, created := system.ActorOf(actor.Addr(mockTask.id), mockTask)
		assert.Assert(t, created)

		groups[ref] = &group{handler: ref}

		req := mockTaskToAllocateRequest(mockTask, ref)
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
			containerID := cproto.NewID()

			devices := make([]device.Device, 0)
			if mockTask.containerStarted {
				if mockTask.slotsNeeded == 0 {
					_, err := agentState.AllocateFreeDevices(0, containerID)
					assert.NilError(t, err)
				} else {
					i := 0
					for d, currContainerID := range agentState.Devices {
						if currContainerID != nil {
							continue
						}
						if i < mockTask.slotsNeeded {
							agentState.Devices[d] = &containerID
							devices = append(devices, d)
							i++
						}
					}
					assert.Assert(t, i == mockTask.slotsNeeded,
						"over allocated to agent %s", mockTask.allocatedAgent.id)
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
			taskList.SetAllocations(req.AllocationRef, allocated)
		}
	}

	return taskList, groups, agents
}

func assertEqualToAllocate(
	t *testing.T,
	actual []*sproto.AllocateRequest,
	expected []*mockTask,
) {
	expectedMap := map[model.AllocationID]bool{}
	for _, task := range expected {
		expectedMap[task.id] = true
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
	expected []*mockTask,
) {
	assert.Equal(t, len(actual), len(expected),
		"actual tasks and expected tasks must have the same length")
	for i := range expected {
		assert.Equal(t, expected[i].id, actual[i].AllocationID)
	}
}

func assertEqualToRelease(
	t *testing.T,
	taskList *taskList,
	actual []*actor.Ref,
	expected []*mockTask,
) {
	expectedMap := map[model.AllocationID]bool{}
	for _, task := range expected {
		expectedMap[task.id] = true
	}
	for _, taskActor := range actual {
		task, _ := taskList.GetAllocationByHandler(taskActor)
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
	prepMockData := func() ([]*mockTask, []*mockGroup, []*mockAgent) {
		lowerPriority := 50
		higherPriority := 40

		agents := []*mockAgent{
			{id: "agent1", slots: 1, maxZeroSlotContainers: 1},
		}
		groups := []*mockGroup{
			{id: "group1", priority: &lowerPriority, weight: 0.5},
			{id: "group2", priority: &higherPriority, weight: 1},
		}
		tasks := []*mockTask{
			{id: "task1", jobID: "job1", slotsNeeded: 1, group: groups[0]},
			{id: "task2", jobID: "job2", slotsNeeded: 1, group: groups[1]},
			{id: "task3", jobID: "job3", slotsNeeded: 0, group: groups[0]},
			{id: "task4", jobID: "job4", slotsNeeded: 0, group: groups[0]},
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
		tasks []*mockTask,
		groups []*mockGroup,
		agents []*mockAgent,
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
		assertStatsEqual(t, jobStats(taskList), expectedStats)
	}
	testFairshare := func(
		t *testing.T,
		tasks []*mockTask,
		groups []*mockGroup,
		agents []*mockAgent,
		expectedStats *jobv1.QueueStats,
	) {
		system := actor.NewSystem(t.Name())
		taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
		toAllocate, _ := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
		AllocateTasks(toAllocate, agentMap, taskList)
		fairshareSchedule(taskList, groupMap, agentMap, BestFit)

		assertStatsEqual(t, jobStats(taskList), expectedStats)
	}

	tasks, groups, agents := prepMockData()
	testPriority(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(2), ScheduledCount: int32(2)})

	tasks, groups, agents = prepMockData()
	testFairshare(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(2), ScheduledCount: int32(2)})

	_, groups, agents = prepMockData()
	tasks = []*mockTask{
		{id: "task1.1", jobID: "job1", slotsNeeded: 2, group: groups[0]}, // same job
		{id: "task1.2", jobID: "job1", slotsNeeded: 2, group: groups[0]},
		{id: "task1.3", jobID: "job1", slotsNeeded: 2, group: groups[0]},
		{id: "task2", jobID: "job2", slotsNeeded: 2, group: groups[1]},
		{id: "task3", jobID: "job3", slotsNeeded: 2, group: groups[0]},
		{id: "task4", jobID: "job4", slotsNeeded: 2, group: groups[0]},
	}
	testPriority(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(4), ScheduledCount: int32(0)})

	_, groups, agents = prepMockData()
	tasks = []*mockTask{
		{id: "task1.1", jobID: "job1", slotsNeeded: 2, group: groups[0]}, // same job
		{id: "task1.2", jobID: "job1", slotsNeeded: 2, group: groups[0]},
		{id: "task1.3", jobID: "job1", slotsNeeded: 2, group: groups[0]},
		{id: "task2", jobID: "job2", slotsNeeded: 2, group: groups[1]},
		{id: "task3", jobID: "job3", slotsNeeded: 2, group: groups[0]},
		{id: "task4", jobID: "job4", slotsNeeded: 2, group: groups[0]},
	}
	testFairshare(t, tasks, groups, agents,
		&jobv1.QueueStats{QueuedCount: int32(4), ScheduledCount: int32(0)})
}

func TestJobOrder(t *testing.T) {
	prepMockData := func() ([]*mockGroup, []*mockAgent) {
		lowerPriority := 50
		higherPriority := 40

		agents := []*mockAgent{
			{id: "agent1", slots: 1, maxZeroSlotContainers: 1},
		}
		groups := []*mockGroup{
			{id: "group1", priority: &lowerPriority, weight: 0.5},
			{id: "group2", priority: &higherPriority, weight: 1},
		}

		return groups, agents
	}

	setupPriority := func(
		tasks []*mockTask,
		groups []*mockGroup,
		agents []*mockAgent,
	) map[model.JobID]*sproto.RMJobInfo {
		p := &priorityScheduler{preemptionEnabled: false}
		system := actor.NewSystem(t.Name())
		taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
		toAllocate, _ := p.prioritySchedule(taskList, groupMap,
			make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
		AllocateTasks(toAllocate, agentMap, taskList)
		return p.JobQInfo(&ResourcePool{taskList: taskList, groups: groupMap})
	}

	setupFairshare := func(
		tasks []*mockTask,
		groups []*mockGroup,
		agents []*mockAgent,
	) map[model.JobID]*sproto.RMJobInfo {
		system := actor.NewSystem(t.Name())
		taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
		toAllocate, _ := fairshareSchedule(taskList, groupMap, agentMap, BestFit)
		AllocateTasks(toAllocate, agentMap, taskList)
		fairshareSchedule(taskList, groupMap, agentMap, BestFit)
		f := fairShare{}
		return f.JobQInfo(&ResourcePool{taskList: taskList, groups: groupMap})
	}

	groups, agents := prepMockData()
	tasks := []*mockTask{
		{id: "task1", jobID: "job1", slotsNeeded: 1, group: groups[0]},
		{id: "task1.1", jobID: "job1", slotsNeeded: 1, group: groups[0]},
		{id: "task2", jobID: "job2", slotsNeeded: 1, group: groups[1]},
		{id: "task3", jobID: "job3", slotsNeeded: 1, group: groups[0]},
		{id: "task4", jobID: "job4", slotsNeeded: 1, group: groups[0]},
		{id: "task4.1", jobID: "job4", slotsNeeded: 0, group: groups[0]},
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
	tasks = []*mockTask{
		{id: "task1", jobID: "job1", slotsNeeded: 1, group: groups[0]},
		{id: "task1.1", jobID: "job1", slotsNeeded: 1, group: groups[0]},
		{id: "task2", jobID: "job2", slotsNeeded: 1, group: groups[1]},
		{id: "task3", jobID: "job3", slotsNeeded: 1, group: groups[0]},
		{id: "task4", jobID: "job4", slotsNeeded: 1, group: groups[0]},
		{id: "task4.1", jobID: "job4", slotsNeeded: 0, group: groups[0]},
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

	agents := []*mockAgent{
		{id: "agent1", slots: 1, maxZeroSlotContainers: 1},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority, weight: 0.5},
		{id: "group2", priority: &higherPriority, weight: 1},
	}

	tasks := []*mockTask{
		{id: "task1", jobID: "job1", slotsNeeded: 1, group: groups[0]},
		{id: "task1.1", jobID: "job1", slotsNeeded: 1, group: groups[0]},
	}

	p := &priorityScheduler{preemptionEnabled: false}
	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
	toAllocate, _ := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	AllocateTasks(toAllocate, agentMap, taskList)
	jobInfo := p.JobQInfo(&ResourcePool{taskList: taskList, groups: groupMap})
	assert.Equal(t, len(jobInfo), 1)
	assert.Equal(t, jobInfo["job1"].State, sproto.SchedulingStateScheduled)
	assert.Equal(t, jobInfo["job1"].AllocatedSlots, 1)
	assert.Equal(t, jobInfo["job1"].JobsAhead, 0)

	newTasks := []*mockTask{
		{id: "task2", jobID: "job2", slotsNeeded: 1, group: groups[1]},
	}

	AddUnallocatedTasks(t, newTasks, system, taskList)
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap,
		make(map[model.JobID]decimal.Decimal), agentMap, BestFit)
	assert.Equal(t, len(toRelease), 0)
	AllocateTasks(toAllocate, agentMap, taskList)
	jobInfo = p.JobQInfo(&ResourcePool{taskList: taskList, groups: groupMap})
	assert.Equal(t, len(jobInfo), 2)
	assert.Equal(t, jobInfo["job1"].State, sproto.SchedulingStateScheduled)
	assert.Equal(t, jobInfo["job1"].AllocatedSlots, 1)
	assert.Equal(t, jobInfo["job1"].JobsAhead, 1)
	assert.Equal(t, jobInfo["job2"].JobsAhead, 0)
	assert.Equal(t, jobInfo["job2"].AllocatedSlots, 0)
	assert.Equal(t, jobInfo["job2"].State, sproto.SchedulingStateQueued)
}
