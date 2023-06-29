package agentrm

import (
	"fmt"
	"sort"

	"github.com/determined-ai/determined/master/internal/rm/tasklist"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

type priorityScheduler struct {
	preemptionEnabled      bool
	allowHeterogeneousFits bool
}

// NewPriorityScheduler creates a new scheduler that schedules tasks via priority.
func NewPriorityScheduler(config *config.SchedulerConfig) Scheduler {
	return &priorityScheduler{
		preemptionEnabled:      config.Priority.Preemption,
		allowHeterogeneousFits: config.AllowHeterogeneousFits,
	}
}

func (p priorityScheduler) Schedule(rp *resourcePool) ([]*sproto.AllocateRequest, []model.AllocationID) {
	return p.prioritySchedule(
		rp.taskList,
		rp.groups,
		rp.queuePositions,
		rp.agentStatesCache,
		rp.fittingMethod,
	)
}

func (p priorityScheduler) JobQInfo(rp *resourcePool) map[model.JobID]*sproto.RMJobInfo {
	reqs := tasklist.SortTasksWithPosition(rp.taskList, rp.groups, rp.queuePositions, false)
	jobQInfo := tasklist.ReduceToJobQInfo(reqs)
	return jobQInfo
}

func (p priorityScheduler) prioritySchedule(
	taskList *tasklist.TaskList,
	groups map[*actor.Ref]*tasklist.Group,
	jobPositions tasklist.JobSortState,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
) ([]*sproto.AllocateRequest, []model.AllocationID) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	toRelease := make([]model.AllocationID, 0)

	// Schedule zero-slot and non-zero-slot tasks independently of each other, e.g., a lower priority
	// zero-slot task can be started while a higher priority non-zero-slot task is pending, and
	// vice versa.
	for _, zeroSlots := range []bool{false, true} {
		allocate, release := p.prioritySchedulerWithFilter(
			taskList,
			groups,
			jobPositions,
			agents,
			fittingMethod,
			taskFilter(zeroSlots),
		)
		toAllocate = append(toAllocate, allocate...)
		toRelease = append(toRelease, release...)
	}

	return toAllocate, toRelease
}

// prioritySchedulerWithFilter defines the logic of each scheduling circle.
// 1. Schedule pending tasks without preemption.
// 2. Search if preempting any lower-priority tasks can make space.
// 3. Back-fill lower-priority pending tasks if there are no tasks to preempt.
func (p priorityScheduler) prioritySchedulerWithFilter(
	taskList *tasklist.TaskList,
	groups map[*actor.Ref]*tasklist.Group,
	jobPositions tasklist.JobSortState,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
	filter func(*sproto.AllocateRequest) bool,
) ([]*sproto.AllocateRequest, []model.AllocationID) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	toRelease := make(map[model.AllocationID]bool)

	// Sort tasks by priorities and timestamps. This sort determines the order in which
	// tasks are scheduled and preempted.
	//nolint:lll // There isn't a great way to break this line that makes it more readable.
	priorityToPendingTasksMap, priorityToScheduledTaskMap := sortTasksByPriorityAndPositionAndTimestamp(
		taskList,
		groups,
		jobPositions,
		filter,
	)

	localAgentsState := deepCopyAgents(agents)

	// If there exist any tasks that cannot be scheduled, all the tasks of lower priorities
	// can only be backfilled if they are preemptible.
	backfilling := false

	for _, priority := range getOrderedPriorities(priorityToPendingTasksMap) {
		allocationRequests := priorityToPendingTasksMap[priority]
		log.Debugf("processing priority %d with %d pending tasks (backfilling: %v)",
			priority, len(allocationRequests), backfilling)

		successfulAllocations, unSuccessfulAllocations := p.trySchedulingPendingTasksInPriority(
			allocationRequests,
			localAgentsState,
			fittingMethod,
		)

		// Only start tasks if there are no tasks of higher priorities to preempt.
		if len(toRelease) == 0 {
			if !backfilling {
				for _, allocatedTask := range successfulAllocations {
					log.Debugf("scheduled task: %s", allocatedTask.Name)
					toAllocate = append(toAllocate, allocatedTask)
				}
			} else if p.preemptionEnabled {
				for _, allocatedTask := range successfulAllocations {
					if !allocatedTask.Preemptible {
						continue
					}
					log.Debugf("scheduled task via backfilling: %s", allocatedTask.Name)
					allocatedTask.State = sproto.SchedulingStateScheduledBackfilled
					toAllocate = append(toAllocate, allocatedTask)
				}
			}
		}

		// Scheduling the tasks of lower priority than the current one is considered to
		// back-filling.
		if len(unSuccessfulAllocations) > 0 {
			backfilling = true
		}

		if p.preemptionEnabled {
			for _, prioritizedAllocation := range unSuccessfulAllocations {
				// Check if we still need to preempt tasks to schedule this task.
				if fits := findFits(
					prioritizedAllocation,
					localAgentsState,
					fittingMethod,
					p.allowHeterogeneousFits,
				); len(
					fits,
				) > 0 {
					log.Debugf(
						"Not preempting tasks for task %s as it will be able to launch "+
							"once already scheduled preemptions complete", prioritizedAllocation.Name)
					addTaskToAgents(fits)
					continue
				}

				taskPlaced, updatedLocalAgentState, preemptedTasks := p.trySchedulingTaskViaPreemption(
					taskList,
					prioritizedAllocation,
					priority,
					jobPositions,
					fittingMethod,
					localAgentsState,
					priorityToScheduledTaskMap,
					toRelease,
					filter,
				)

				if taskPlaced {
					localAgentsState = updatedLocalAgentState
					for preemptedTask := range preemptedTasks {
						log.Debugf(
							"preempting task %s for task %s",
							preemptedTask.ToTaskID(),
							prioritizedAllocation.Name,
						)
						toRelease[preemptedTask] = true
					}
				}
			}
		}
	}

	toReleaseSlice := make([]model.AllocationID, 0)
	for r := range toRelease {
		toReleaseSlice = append(toReleaseSlice, r)
	}
	return toAllocate, toReleaseSlice
}

// trySchedulingTaskViaPreemption checks whether preempting lower priority tasks
// would allow this task to be scheduled.
func (p priorityScheduler) trySchedulingTaskViaPreemption(
	taskList *tasklist.TaskList,
	allocationRequest *sproto.AllocateRequest,
	allocationPriority int,
	jobPositions tasklist.JobSortState,
	fittingMethod SoftConstraint,
	agents map[*actor.Ref]*agentState,
	priorityToScheduledTaskMap map[int][]*sproto.AllocateRequest,
	tasksAlreadyPreempted map[model.AllocationID]bool,
	filter func(*sproto.AllocateRequest) bool,
) (bool, map[*actor.Ref]*agentState, map[model.AllocationID]bool) {
	localAgentsState := deepCopyAgents(agents)
	preemptedTasks := make(map[model.AllocationID]bool)
	log.Debugf("trying to schedule task %s by preempting other tasks", allocationRequest.Name)

	for priority := model.MaxUserSchedulingPriority; priority >= allocationPriority; priority-- {
		for i := len(priorityToScheduledTaskMap[priority]) - 1; i >= 0; i-- {
			allocationJobID := allocationRequest.JobID
			candidateJobID := priorityToScheduledTaskMap[priority][i].JobID
			if priority == allocationPriority &&
				jobPositions[allocationJobID].GreaterThanOrEqual(jobPositions[candidateJobID]) {
				break
			}
			preemptionCandidate := priorityToScheduledTaskMap[priority][i]
			if !preemptionCandidate.Preemptible || !filter(preemptionCandidate) {
				continue
			}

			if _, ok := tasksAlreadyPreempted[preemptionCandidate.AllocationID]; ok {
				continue
			}

			allocated := taskList.Allocation(preemptionCandidate.AllocationID)
			removeTaskFromAgents(localAgentsState, allocated)
			preemptedTasks[preemptionCandidate.AllocationID] = true

			if fits := findFits(
				allocationRequest,
				localAgentsState,
				fittingMethod,
				p.allowHeterogeneousFits,
			); len(fits) > 0 {
				addTaskToAgents(fits)
				return true, localAgentsState, preemptedTasks
			}
		}
	}

	return false, localAgentsState, preemptedTasks
}

// trySchedulingPendingTasksInPriority tries to schedule all the tasks in the
// current priority. Note tasks are scheduled based on the order in which they
// are listed.
func (p priorityScheduler) trySchedulingPendingTasksInPriority(
	allocationRequests []*sproto.AllocateRequest,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
) ([]*sproto.AllocateRequest, []*sproto.AllocateRequest) {
	successfulAllocations := make([]*sproto.AllocateRequest, 0)
	unSuccessfulAllocations := make([]*sproto.AllocateRequest, 0)

	for _, allocationRequest := range allocationRequests {
		fits := findFits(allocationRequest, agents, fittingMethod, p.allowHeterogeneousFits)
		if len(fits) == 0 {
			unSuccessfulAllocations = append(unSuccessfulAllocations, allocationRequest)
			continue
		}
		addTaskToAgents(fits)
		successfulAllocations = append(successfulAllocations, allocationRequest)
	}

	return successfulAllocations, unSuccessfulAllocations
}

// sortTasksByPriorityAndPositionAndTimestamp sorts all pending and scheduled tasks
// separately by priority. Within each priority, tasks are ordered
// based on their queue position and then creation time.
func sortTasksByPriorityAndPositionAndTimestamp(
	taskList *tasklist.TaskList,
	groups map[*actor.Ref]*tasklist.Group,
	jobPositions tasklist.JobSortState,
	filter func(*sproto.AllocateRequest) bool,
) (map[int][]*sproto.AllocateRequest, map[int][]*sproto.AllocateRequest) {
	// Sort all non-zero slot tasks by priority.
	priorityToPendingTasksMap := make(map[int][]*sproto.AllocateRequest)
	priorityToScheduledTaskMap := make(map[int][]*sproto.AllocateRequest)

	for _, req := range tasklist.SortTasksWithPosition(taskList, groups, jobPositions, false) {
		if !filter(req) {
			continue
		}

		priority := groups[req.Group].Priority
		if priority == nil {
			panic(fmt.Sprintf("priority not set for task %s", req.Name))
		}

		if taskList.IsScheduled(req.AllocationID) {
			priorityToScheduledTaskMap[*priority] = append(
				priorityToScheduledTaskMap[*priority],
				req,
			)
		} else {
			priorityToPendingTasksMap[*priority] = append(priorityToPendingTasksMap[*priority], req)
		}
	}

	return priorityToPendingTasksMap, priorityToScheduledTaskMap
}

func deepCopyAgents(agents map[*actor.Ref]*agentState) map[*actor.Ref]*agentState {
	copiedAgents := make(map[*actor.Ref]*agentState)
	for key, agent := range agents {
		copiedAgents[key] = agent.deepCopy()
	}
	return copiedAgents
}

func addTaskToAgents(fits []*fittingState) {
	for _, fit := range fits {
		if _, err := fit.Agent.allocateFreeDevices(fit.Slots, cproto.NewID()); err != nil {
			panic(errors.Wrap(err, "can't add task to agents"))
		}
	}
}

func removeTaskFromAgents(
	agents map[*actor.Ref]*agentState,
	resourcesAllocated *sproto.ResourcesAllocated,
) {
	for _, allocation := range resourcesAllocated.Resources {
		allocation := allocation.(*containerResources)

		// TODO properly handle this case since this will likely
		// lead to issues in many cases.
		agentState := agents[allocation.agent.Handler]
		if agentState == nil {
			log.Errorf("tried to remove an allocation (allocationID: %s containerID: %s) "+
				"from an agent: (agentID: %+v) but scheduler could not find the agent",
				allocation.req.AllocationID, allocation.containerID,
				allocation.agent.Handler.Address().Local(),
			)
			continue
		}

		agentState.deallocateContainer(allocation.containerID)
	}
}

func getOrderedPriorities(allocationsByPriority map[int][]*sproto.AllocateRequest) []int {
	keys := make([]int, 0, len(allocationsByPriority))
	for k := range allocationsByPriority {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func taskFilter(zeroSlots bool) func(*sproto.AllocateRequest) bool {
	return func(request *sproto.AllocateRequest) bool {
		return (request.SlotsNeeded == 0) == zeroSlots
	}
}
