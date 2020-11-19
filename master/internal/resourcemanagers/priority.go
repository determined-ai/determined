package resourcemanagers

import (
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

type priorityScheduler struct {
	preemptionEnabled bool
}

// NewPriorityScheduler creates a new scheduler that schedules tasks via priority.
func NewPriorityScheduler() Scheduler {
	return &priorityScheduler{preemptionEnabled: true}
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
	toRelease := make(map[*actor.Ref]bool)

	// Sort tasks by priorities and timestamps. This sort determines the order in which
	// tasks are scheduled and preempted.
	priorityToPendingTasksMap, priorityToScheduledTaskMap := sortTasksByPriorityAndTimestamp(
		taskList, groups, label)

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
				log.Infof(
					"Not preempting tasks for task %s as it will be able to launch "+
						"once already scheduled preemptions complete", prioritizedAllocation.Name)
				addTaskToAgents(fits)
				continue
			}

			taskPlaced, updatedLocalAgentState, preemptedTasks := trySchedulingTaskViaPreemption(
				taskList, prioritizedAllocation, priority, fittingMethod, localAgentsState,
				priorityToScheduledTaskMap, toRelease)

			if taskPlaced {
				log.Infof("Preempted %d tasks for task: %s", len(preemptedTasks),
					prioritizedAllocation.Name)

				localAgentsState = updatedLocalAgentState
				for preemptedTask := range preemptedTasks {
					log.Infof("preempting task %s for task %s",
						preemptedTask.Address().Local(), prioritizedAllocation.Name)
					toRelease[preemptedTask] = true
				}
			}
		}
	}

	// Convert to a slice which is the expected output format.
	toReleaseSlice := make([]*actor.Ref, 0, len(toRelease))
	for k := range toRelease {
		toReleaseSlice = append(toReleaseSlice, k)
	}

	return toAllocate, toReleaseSlice
}

// trySchedulingTaskViaPreemption checks whether preempting lower priority tasks
// would allow this task to be scheduled.
func trySchedulingTaskViaPreemption(
	taskList *taskList,
	allocationRequest *AllocateRequest,
	allocationPriority int,
	fittingMethod SoftConstraint,
	agents map[*actor.Ref]*agentState,
	priorityToScheduledTaskMap map[int][]*AllocateRequest,
	tasksAlreadyPreempted map[*actor.Ref]bool,
) (bool, map[*actor.Ref]*agentState, map[*actor.Ref]bool) {
	localAgentsState := deepCopyAgents(agents)
	preemptedTasks := make(map[*actor.Ref]bool)
	log.Infof("trying to schedule task %s by preempting other tasks", allocationRequest.Name)

	for priority := model.MaxUserSchedulingPriority; priority > allocationPriority; priority-- {
		if _, ok := priorityToScheduledTaskMap[priority]; !ok {
			continue
		}

		preemptionCandidates := priorityToScheduledTaskMap[priority]
		for _, preemptionCandidate := range preemptionCandidates {
			if preemptionCandidate.NonPreemptible || preemptionCandidate.SlotsNeeded == 0 {
				continue
			}

			if _, ok := tasksAlreadyPreempted[preemptionCandidate.TaskActor]; ok {
				continue
			}

			resourcesAllocated := taskList.GetAllocations(preemptionCandidate.TaskActor)
			removeTaskFromAgents(localAgentsState, resourcesAllocated)
			preemptedTasks[preemptionCandidate.TaskActor] = true

			for _, agent := range localAgentsState {
				log.Infof("agent has %d slots", agent.numEmptySlots())
			}

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
		addTaskToAgents(fits)
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

// sortTasksByPriorityAndTimestamp sorts all pending and scheduled tasks
// separately by priority. Within each priority, tasks are ordered
// based on their creation time.
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

func addTaskToAgents(fits []*fittingState) {
	for _, fit := range fits {
		fit.Agent.allocateFreeDevices(fit.Slots, "simulation")
	}
}

func removeTaskFromAgents(
	agents map[*actor.Ref]*agentState,
	resourcesAllocated *ResourcesAllocated,
) {
	for _, allocation := range resourcesAllocated.Allocations {
		allocation := allocation.(*containerAllocation)
		for _, allocatedDevice := range allocation.devices {
			// Local devices are a deep copy of the originals so we loop over trying to find
			// the device that matches. If we assume that we have homogeneous devices we could
			// just search for the first available device.
			for localDevice, localContainer := range agents[allocation.agent.handler].devices {
				if allocatedDevice.ID == localDevice.ID && localContainer != nil {
					agents[allocation.agent.handler].devices[localDevice] = nil
				}
			}
		}
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
