package resourcemanagers

import (
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type priorityScheduler struct{
	preemptionEnabled bool
}

// NewPriorityScheduler creates a new scheduler that schedules tasks via priority.
func NewPriorityScheduler() Scheduler {
	return &priorityScheduler{}
}

func (p *priorityScheduler) Schedule(rp *ResourcePool) ([]*AllocateRequest, []*actor.Ref) {
	return p.prioritySchedule(rp.taskList, rp.groups, rp.agents, rp.fittingMethod)
}

func (p *priorityScheduler) prioritySchedule(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
) ([]*AllocateRequest, []*actor.Ref) {
	agentsSplitByLabel := splitAgentsByLabel(agents)
	toAllocate := make([]*AllocateRequest, 0)
	toRelease := make([]*actor.Ref, 0)

	// Since labels are a hard scheduling constraint, process every
	// label independently.
	for label, agentsWithLabel := range agentsSplitByLabel {
		toAllocatedForLabel, toReleaseForLabel := p.priorityScheduleByLabel(
			taskList, groups, agentsWithLabel, fittingMethod, label)
		toAllocate = append(toAllocate, toAllocatedForLabel...)
		toRelease = append(toRelease, toReleaseForLabel...)
	}

	return toAllocate, toRelease
}

func (p *priorityScheduler) priorityScheduleByLabel(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
	label string,
) ([]*AllocateRequest, []*actor.Ref) {
	// All pending zero slot tasks get scheduled right away.
	toAllocate := getAllPendingZeroSlotTasks(taskList, label)
	toRelease := make([]*actor.Ref, 0)

	// Sort tasks by priorities and timestamps.
	priorityToPendingTasksMap, priorityToScheduledTaskMap := sortTasksByPriorityAndTimestamp(taskList, groups, label)

	// Make a local copy of the agent state that we will modify.
	localAgentsState := deepCopyAgents(agents)

	for _, priority := range getOrderedPriorities(priorityToPendingTasksMap) {
		allocationRequests := priorityToPendingTasksMap[priority]
		log.Infof("processing priority %d with %d pending tasks", priority, len(allocationRequests))

		successfulAllocations, unSuccessfulAllocations := trySchedulingPendingTasksInPriority(
			allocationRequests, localAgentsState, fittingMethod)

		// Only add these tasks to the lists of tasks to start if there has been
		// no preemption to start tasks of higher priority.
		if len(toRelease) == 0 {
			toAllocate = append(toAllocate, successfulAllocations...)
		}

		// All pending tasks in this priority were successfully scheduled.
		if len(unSuccessfulAllocations) == 0 {
			continue
		}

		if !p.preemptionEnabled {
			log.Infof(
				"scheduled only %d tasks in priority level and preemption thus breaking out",
				len(successfulAllocations))
			break
		}

		for _, prioritizedAllocation := range unSuccessfulAllocations {
			// Check if we still need to preempt tasks to schedule this task.
			if fits := findFits(prioritizedAllocation, localAgentsState, fittingMethod); len(fits) > 0 {
				simulateFitsPlacement(fits)
				continue
			}




		}
	}

	return toAllocate, toRelease
}

func trySchedulingTaskViaPreemption(
	allocationRequest *AllocateRequest,
	priority int,
	agents map[*actor.Ref]*agentState,
	priorityToScheduledTaskMap map[int][]*AllocateRequest,
) (bool, map[*actor.Ref]*agentState) {
	localAgentsState := deepCopyAgents(agents)
	taskPlacedViaPreemption := false


	return taskPlacedViaPreemption, localAgentsState
}

func trySchedulingPendingTasksInPriority(
	allocationRequests []*AllocateRequest,
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
) ([]*AllocateRequest, []*AllocateRequest) {
	successfulAllocations := make([]*AllocateRequest, 0)
	unSuccessfulAllocations := make([]*AllocateRequest, 0)

	for _, allocationRequest := range allocationRequests {
		log.Infof("trying to schedule task: %s", allocationRequest.ID)
		fits := findFits(allocationRequest, agents, fittingMethod)
		if len(fits) == 0 {
			unSuccessfulAllocations = append(unSuccessfulAllocations, allocationRequest)
			continue
		}
		simulateFitsPlacement(fits)
		successfulAllocations = append(successfulAllocations, allocationRequest)
		log.Infof("successfully scheduled task: %s", allocationRequest.ID)
	}

	return successfulAllocations, unSuccessfulAllocations
}

func getAllPendingZeroSlotTasks(taskList *taskList, label string) []*AllocateRequest {
	pendingZeroSlotTasks := make([]*AllocateRequest, 0)
	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if req.Label != label || req.SlotsNeeded > 0 {
			continue
		}

		assigned := taskList.GetAllocations(req.TaskActor)
		if assigned == nil || len(assigned.Allocations) == 0 {
			log.Infof("scheduling pending zero-slot task: %s", req.ID)
			pendingZeroSlotTasks = append(pendingZeroSlotTasks, req)
		}
	}

	return pendingZeroSlotTasks
}

func sortTasksByPriorityAndTimestamp(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	label string,
) (map[int][]*AllocateRequest, map[int][]*AllocateRequest) {
	// Sort all non-zero slot tasks by priority.
	priorityToPendingTasksMap := make(map[int][]*AllocateRequest)
	priorityToScheduledTaskMap := make(map[int][]*AllocateRequest)

	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if req.Label != label || req.SlotsNeeded == 0 {
			continue
		}

		priority := groups[req.Group].priority
		if priority == nil {
			panic(fmt.Sprintf("priority not set for task %s", req.Name))
		}

		assigned := taskList.GetAllocations(req.TaskActor)
		switch {
		case assigned == nil || len(assigned.Allocations) == 0:
			if _, ok := priorityToPendingTasksMap[*priority]; !ok {
				priorityToPendingTasksMap[*priority] = make([]*AllocateRequest, 0)
			}
			priorityToPendingTasksMap[*priority] = append(priorityToPendingTasksMap[*priority], req)

		default:
			if _, ok := priorityToScheduledTaskMap[*priority]; !ok {
				priorityToScheduledTaskMap[*priority] = make([]*AllocateRequest, 0)
			}
			priorityToScheduledTaskMap[*priority] = append(priorityToScheduledTaskMap[*priority], req)
		}
	}

	// For each priority sort pending tasks by longest to shortest time of existence.
	for priority := range priorityToPendingTasksMap {
		pendingTasks := priorityToPendingTasksMap[priority]
		sort.Slice(pendingTasks, func(i, j int) bool {
			first, second := pendingTasks[i], pendingTasks[j]
			return second.TaskActor.RegisteredTime().Before(first.TaskActor.RegisteredTime())
		})
	}

	// For each priority sort scheduled tasks by shortest to longest time of existence.
	for priority := range priorityToScheduledTaskMap {
		scheduledTasks := priorityToScheduledTaskMap[priority]
		sort.Slice(scheduledTasks, func(i, j int) bool {
			first, second := scheduledTasks[i], scheduledTasks[j]
			return first.TaskActor.RegisteredTime().Before(second.TaskActor.RegisteredTime())
		})
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

func simulateFitsPlacement(fits []*fittingState) {
	for _, fit := range fits {
		fit.Agent.allocateFreeDevices(fit.Slots, "simulation")
	}
}

func getOrderedPriorities(allocationsByPriority map[int][]*AllocateRequest) []int {
	keys := make([]int, 0, len(allocationsByPriority))
	for k := range allocationsByPriority {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func splitAgentsByLabel(agents map[*actor.Ref]*agentState) map[string]map[*actor.Ref]*agentState {
	agentsSplitByLabel := make(map[string]map[*actor.Ref]*agentState)
	for agentRef, agent := range agents {
		if _, ok := agentsSplitByLabel[agent.label]; !ok {
			agentsSplitByLabel[agent.label] = make(map[*actor.Ref]*agentState)
		}
		agentsSplitByLabel[agent.label][agentRef] = agent
	}
	return agentsSplitByLabel
}
