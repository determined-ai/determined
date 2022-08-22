package rm

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

type priorityScheduler struct {
	preemptionEnabled bool
}

// AllocReqs is an alias for a list of Allocate Requests.
type AllocReqs = []*sproto.AllocateRequest

// NewPriorityScheduler creates a new scheduler that schedules tasks via priority.
func NewPriorityScheduler(config *config.SchedulerConfig) Scheduler {
	return &priorityScheduler{
		preemptionEnabled: config.Priority.Preemption,
	}
}

func (p *priorityScheduler) Schedule(rp *ResourcePool) ([]*sproto.AllocateRequest, []*actor.Ref) {
	return p.prioritySchedule(rp.taskList, rp.groups, rp.queuePositions,
		rp.agentStatesCache, rp.fittingMethod)
}

func (p *priorityScheduler) JobQInfo(rp *ResourcePool) map[model.JobID]*sproto.RMJobInfo {
	reqs := sortTasksWithPosition(rp.taskList, rp.groups, rp.queuePositions, false)
	jobQInfo := reduceToJobQInfo(reqs)
	return jobQInfo
}

func (p *priorityScheduler) prioritySchedule(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	jobPositions jobSortState,
	agents map[*actor.Ref]*AgentState,
	fittingMethod SoftConstraint,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	toRelease := make([]*actor.Ref, 0)

	// Since labels are a hard scheduling constraint, process every label independently.
	for label, agentsWithLabel := range splitAgentsByLabel(agents) {
		// Schedule zero-slot and non-zero-slot tasks independently of each other, e.g., a lower priority
		// zero-slot task can be started while a higher priority non-zero-slot task is pending, and
		// vice versa.
		for _, zeroSlots := range []bool{false, true} {
			allocate, release := p.prioritySchedulerWithFilter(
				taskList,
				groups,
				jobPositions,
				agentsWithLabel,
				fittingMethod,
				taskFilter(label, zeroSlots),
			)
			toAllocate = append(toAllocate, allocate...)
			toRelease = append(toRelease, release...)
		}
	}

	return toAllocate, toRelease
}

// prioritySchedulerWithFilter defines the logic of each scheduling circle.
// 1. Schedule pending tasks without preemption.
// 2. Search if preempting any lower-priority tasks can make space.
// 3. Back-fill lower-priority pending tasks if there are no tasks to preempt.
func (p *priorityScheduler) prioritySchedulerWithFilter(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	jobPositions jobSortState,
	agents map[*actor.Ref]*AgentState,
	fittingMethod SoftConstraint,
	filter func(*sproto.AllocateRequest) bool,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	toRelease := make(map[*actor.Ref]bool)

	// Sort tasks by priorities and timestamps. This sort determines the order in which
	// tasks are scheduled and preempted.
	//nolint:lll // There isn't a great way to break this line that makes it more readable.
	priorityToPendingTasksMap, priorityToScheduledTaskMap := sortTasksByPriorityAndPositionAndTimestamp(taskList, groups, jobPositions, filter)

	localAgentsState := deepCopyAgents(agents)

	// If there exist any tasks that cannot be scheduled, all the tasks of lower priorities
	// can only be backfilled if they are preemptible.
	backfilling := false

	for _, priority := range getOrderedPriorities(priorityToPendingTasksMap) {
		allocationRequests := priorityToPendingTasksMap[priority]
		log.Debugf("processing priority %d with %d pending tasks (backfilling: %v)",
			priority, len(allocationRequests), backfilling)

		successfulAllocations, unSuccessfulAllocations := trySchedulingPendingTasksInPriority(
			allocationRequests, localAgentsState, fittingMethod)

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
				if fits := findFits(prioritizedAllocation, localAgentsState, fittingMethod); len(
					fits,
				) > 0 {
					log.Debugf(
						"Not preempting tasks for task %s as it will be able to launch "+
							"once already scheduled preemptions complete", prioritizedAllocation.Name)
					addTaskToAgents(fits)
					continue
				}

				taskPlaced, updatedLocalAgentState, preemptedTasks := trySchedulingTaskViaPreemption(
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
						log.Debugf("preempting task %s for task %s",
							preemptedTask.Address().Local(), prioritizedAllocation.Name)
						toRelease[preemptedTask] = true
					}
				}
			}
		}
	}

	toReleaseSlice := make([]*actor.Ref, 0)
	for r := range toRelease {
		toReleaseSlice = append(toReleaseSlice, r)
	}
	return toAllocate, toReleaseSlice
}

// trySchedulingTaskViaPreemption checks whether preempting lower priority tasks
// would allow this task to be scheduled.
func trySchedulingTaskViaPreemption(
	taskList *taskList,
	allocationRequest *sproto.AllocateRequest,
	allocationPriority int,
	jobPositions jobSortState,
	fittingMethod SoftConstraint,
	agents map[*actor.Ref]*AgentState,
	priorityToScheduledTaskMap map[int][]*sproto.AllocateRequest,
	tasksAlreadyPreempted map[*actor.Ref]bool,
	filter func(*sproto.AllocateRequest) bool,
) (bool, map[*actor.Ref]*AgentState, map[*actor.Ref]bool) {
	localAgentsState := deepCopyAgents(agents)
	preemptedTasks := make(map[*actor.Ref]bool)
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

			if _, ok := tasksAlreadyPreempted[preemptionCandidate.AllocationRef]; ok {
				continue
			}

			resourcesAllocated := taskList.GetAllocations(preemptionCandidate.AllocationRef)
			removeTaskFromAgents(localAgentsState, resourcesAllocated)
			preemptedTasks[preemptionCandidate.AllocationRef] = true

			if fits := findFits(allocationRequest, localAgentsState, fittingMethod); len(fits) > 0 {
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
func trySchedulingPendingTasksInPriority(
	allocationRequests []*sproto.AllocateRequest,
	agents map[*actor.Ref]*AgentState,
	fittingMethod SoftConstraint,
) ([]*sproto.AllocateRequest, []*sproto.AllocateRequest) {
	successfulAllocations := make([]*sproto.AllocateRequest, 0)
	unSuccessfulAllocations := make([]*sproto.AllocateRequest, 0)

	for _, allocationRequest := range allocationRequests {
		fits := findFits(allocationRequest, agents, fittingMethod)
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
	taskList *taskList,
	groups map[*actor.Ref]*group,
	jobPositions jobSortState,
	filter func(*sproto.AllocateRequest) bool,
) (map[int][]*sproto.AllocateRequest, map[int][]*sproto.AllocateRequest) {
	// Sort all non-zero slot tasks by priority.
	priorityToPendingTasksMap := make(map[int][]*sproto.AllocateRequest)
	priorityToScheduledTaskMap := make(map[int][]*sproto.AllocateRequest)

	for _, req := range sortTasksWithPosition(taskList, groups, jobPositions, false) {
		if !filter(req) {
			continue
		}

		priority := groups[req.Group].priority
		if priority == nil {
			panic(fmt.Sprintf("priority not set for task %s", req.Name))
		}

		assigned := taskList.GetAllocations(req.AllocationRef)
		if assignmentIsScheduled(assigned) {
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

// comparePositions returns the following:
// 1 if a is in front of b.
// 0 if a is equal to b in position.
// -1 if a is behind b.
func comparePositions(a, b *sproto.AllocateRequest, jobPositions jobSortState) int {
	aPosition, aOk := jobPositions[a.JobID]
	bPosition, bOk := jobPositions[b.JobID]
	zero := decimal.NewFromInt(0)
	if !aOk || !bOk {
		// we shouldn't run into this situation once k8 support is implemented other than
		// when testing.
		return aReqComparator(a, b) * -1
	}
	switch {
	case aPosition == bPosition:
		return aReqComparator(a, b) * -1
	case aPosition.LessThan(zero) || bPosition.LessThan(zero):
		if aPosition.GreaterThan(zero) {
			return 1
		}
		return -1
	case aPosition.LessThan(bPosition):
		return 1
	default:
		return -1
	}
}

func sortTasksWithPosition(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	jobPositions jobSortState,
	k8s bool,
) []*sproto.AllocateRequest {
	var reqs []*sproto.AllocateRequest
	for it := taskList.iterator(); it.next(); {
		reqs = append(reqs, it.value())
	}
	sort.Slice(reqs, func(i, j int) bool {
		p1 := *groups[reqs[i].Group].priority
		p2 := *groups[reqs[j].Group].priority
		if k8s { // in k8s, higher priority == more prioritized
			switch {
			case p1 > p2:
				return true
			case p2 > p1:
				return false
			}
		} else {
			switch {
			case p1 > p2:
				return false
			case p2 > p1:
				return true
			}
		}

		return comparePositions(reqs[i], reqs[j], jobPositions) > 0
	})

	return reqs
}

func deepCopyAgents(agents map[*actor.Ref]*AgentState) map[*actor.Ref]*AgentState {
	copiedAgents := make(map[*actor.Ref]*AgentState)
	for key, agent := range agents {
		copiedAgents[key] = agent.DeepCopy()
	}
	return copiedAgents
}

func addTaskToAgents(fits []*fittingState) {
	for _, fit := range fits {
		if _, err := fit.Agent.AllocateFreeDevices(fit.Slots, cproto.NewID()); err != nil {
			panic(errors.Wrap(err, "can't add task to agents"))
		}
	}
}

func removeTaskFromAgents(
	agents map[*actor.Ref]*AgentState,
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

		agentState.DeallocateContainer(allocation.containerID)
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

func splitAgentsByLabel(
	agents map[*actor.Ref]*AgentState,
) map[string]map[*actor.Ref]*AgentState {
	agentsSplitByLabel := make(map[string]map[*actor.Ref]*AgentState, len(agents))
	for agentRef, agentState := range agents {
		if _, ok := agentsSplitByLabel[agentState.Label]; !ok {
			agentsSplitByLabel[agentState.Label] = make(map[*actor.Ref]*AgentState)
		}
		agentsSplitByLabel[agentState.Label][agentRef] = agentState
	}
	return agentsSplitByLabel
}

func taskFilter(label string, zeroSlots bool) func(*sproto.AllocateRequest) bool {
	return func(request *sproto.AllocateRequest) bool {
		return request.Label == label && (request.SlotsNeeded == 0) == zeroSlots
	}
}
