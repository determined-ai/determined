package resourcemanagers

import (
	"testing"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"

	"github.com/google/uuid"

	"github.com/pkg/errors"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/job"
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

	id               model.AllocationID
	jobID            string
	group            *mockGroup
	slotsNeeded      int
	nonPreemptible   bool
	label            string
	resourcePool     string
	allocatedAgent   *mockAgent
	containerStarted bool
	jobSubmissionTime *time.Time
}

func (t *mockTask) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
	case actor.PostStop:
	case SendRequestResourcesToResourceManager:
		task := sproto.AllocateRequest{
			AllocationID: t.id,
			Name:         string(t.id),
			SlotsNeeded:  t.slotsNeeded,
			Preemptible:  !t.nonPreemptible,
			Label:        t.label,
			ResourcePool: t.resourcePool,
			TaskActor:    ctx.Self(),
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
		task := sproto.ResourcesReleased{TaskActor: ctx.Self()}
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
		for rank, allocation := range msg.Reservations {
			allocation.Start(ctx, tasks.TaskSpec{}, sproto.ReservationRuntimeInfo{
				Token:        "",
				AgentRank:    rank,
				IsMultiAgent: len(msg.Reservations) > 1,
			})
		}
	case sproto.ReleaseResources:
		ctx.Tell(t.rmRef, sproto.ResourcesReleased{TaskActor: ctx.Self()})

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
	system *actor.System,
	config *ResourcePoolConfig,
	mockTasks []*mockTask,
	mockGroups []*mockGroup,
	mockAgents []*mockAgent,
) (*ResourcePool, *actor.Ref) {
	if config == nil {
		config = &ResourcePoolConfig{PoolName: "pool"}
	}
	if config.Scheduler == nil {
		config.Scheduler = &SchedulerConfig{
			FairShare:     &FairShareSchedulerConfig{},
			FittingPolicy: best,
		}
	}

	rp := NewResourcePool(
		config, nil, MakeScheduler(config.Scheduler),
		MakeFitFunction(config.Scheduler.FittingPolicy))
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
	state := newAgentState(sproto.AddAgent{Agent: ref}, 100)
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
	maxZeroSlotContainers int,
	zeroSlotContainers int,
) *agentState {
	ref, created := system.ActorOf(actor.Addr(id), &mockAgent{id: id, slots: slots, label: label})
	assert.Assert(t, created)
	state := newAgentState(sproto.AddAgent{Agent: ref, Label: label}, maxZeroSlotContainers)
	for i := 0; i < slots; i++ {
		state.devices[device.Device{ID: i}] = nil
	}

	if slotsUsed > 0 {
		req := &sproto.AllocateRequest{
			SlotsNeeded: slotsUsed,
			Preemptible: true,
		}
		container := newContainer(req, req.SlotsNeeded)
		state.allocateFreeDevices(req.SlotsNeeded, container.id)
	}

	for i := 0; i < zeroSlotContainers; i++ {
		req := &sproto.AllocateRequest{}
		container := newContainer(req, req.SlotsNeeded)
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
	task := &mockTask{id: model.AllocationID(taskID), slotsNeeded: slotsNeeded}
	ref, created := system.ActorOf(actor.Addr(taskID), task)
	assert.Assert(t, created)

	req := &sproto.AllocateRequest{
		AllocationID: model.AllocationID(taskID),
		TaskActor:    ref,
		Group:        ref,
		SlotsNeeded:  slotsNeeded,
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
			ID:           model.AllocationID(taskID),
			Reservations: []sproto.Reservation{},
		}
		for i := 0; i < numAllocated; i++ {
			allocated.Reservations = append(allocated.Reservations, containerReservation{})
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
			handler:               ref,
			label:                 mockAgent.label,
			devices:               make(map[device.Device]*cproto.ID),
			zeroSlotContainers:    make(map[cproto.ID]bool),
			maxZeroSlotContainers: mockAgent.maxZeroSlotContainers,
			enabled:               true,
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

		var jobID *model.JobID
		if mockTask.jobID != "" {
			jid := model.JobID(mockTask.jobID)
			jobID = &jid
		}

		req := &sproto.AllocateRequest{
			AllocationID: mockTask.id,
			JobID:        jobID,
			SlotsNeeded:  mockTask.slotsNeeded,
			Label:        mockTask.label,
			TaskActor:    ref,
			Preemptible:  !mockTask.nonPreemptible,
			JobSubmissionTime: mockTask.jobSubmissionTime,
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
			container := newContainer(req, req.SlotsNeeded)

			devices := make([]device.Device, 0)
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
				Reservations: []sproto.Reservation{
					&containerReservation{
						req:       req,
						agent:     agentState,
						container: container,
						devices:   devices,
					},
				},
			}
			taskList.SetAllocations(req.TaskActor, allocated)
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
		"actual tasks and expected tasks must have the same length")
}

func assertEqualToAllocateOrdered(
	t *testing.T,
	actual []*sproto.AllocateRequest,
	expected []*mockTask,
) {
	assert.Equal(t, len(actual), len(expected),
		"actual tasks and expected tasks must have the same length")
	for i, _ := range expected {
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
		task, _ := taskList.GetTaskByHandler(taskActor)
		assert.Assert(t, task != nil)

		if task != nil {
			_, ok := expectedMap[task.AllocationID]
			assert.Assert(t, ok)
		}
	}
	assert.Equal(t, len(actual), len(expected),
		"actual tasks and expected tasks must have the same length")
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
		toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
		AllocateTasks(toAllocate, agentMap, taskList)
		p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
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
	) map[model.JobID]*job.RMJobInfo {
		p := &priorityScheduler{preemptionEnabled: false}
		system := actor.NewSystem(t.Name())
		taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)
		toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
		AllocateTasks(toAllocate, agentMap, taskList)
		return p.JobQInfo(&ResourcePool{taskList: taskList, groups: groupMap})
	}

	setupFairshare := func(
		tasks []*mockTask,
		groups []*mockGroup,
		agents []*mockAgent,
	) map[model.JobID]*job.RMJobInfo {
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
	assert.Equal(t, jobInfo["job2"].State, job.SchedulingStateScheduled)
	assert.Equal(t, jobInfo["job2"].JobsAhead, 0)
	assert.Equal(t, jobInfo["job2"].AllocatedSlots, 1)
	assert.Equal(t, jobInfo["job2"].RequestedSlots, 1)
	assert.Equal(t, jobInfo["job1"].State, job.SchedulingStateQueued)
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
	assert.Equal(t, jobInfo["job4"].State, job.SchedulingStateScheduled)
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
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	AllocateTasks(toAllocate, agentMap, taskList)
	jobInfo := p.JobQInfo(&ResourcePool{taskList: taskList, groups: groupMap})
	assert.Equal(t, len(jobInfo), 1)
	assert.Equal(t, jobInfo["job1"].State, job.SchedulingStateScheduled)
	assert.Equal(t, jobInfo["job1"].AllocatedSlots, 1)
	assert.Equal(t, jobInfo["job1"].JobsAhead, 0)

	newTasks := []*mockTask{
		{id: "task2", jobID: "job2", slotsNeeded: 1, group: groups[1]},
	}

	AddUnallocatedTasks(t, newTasks, system, taskList)
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)
	assert.Equal(t, len(toRelease), 0)
	AllocateTasks(toAllocate, agentMap, taskList)
	jobInfo = p.JobQInfo(&ResourcePool{taskList: taskList, groups: groupMap})
	assert.Equal(t, len(jobInfo), 2)
	assert.Equal(t, jobInfo["job1"].State, job.SchedulingStateScheduled)
	assert.Equal(t, jobInfo["job1"].AllocatedSlots, 1)
	assert.Equal(t, jobInfo["job1"].JobsAhead, 1)
	assert.Equal(t, jobInfo["job2"].JobsAhead, 0)
	assert.Equal(t, jobInfo["job2"].AllocatedSlots, 0)
	assert.Equal(t, jobInfo["job2"].State, job.SchedulingStateQueued)
}
