package resourcemanagers

import (
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
)

type priorityScheduler struct {
	preemptionEnabled bool
}

// NewPriorityScheduler creates a new scheduler that schedules tasks via priority.
func NewPriorityScheduler(config *SchedulerConfig) Scheduler {
	return &priorityScheduler{
		preemptionEnabled: config.Priority.Preemption,
	}
}

func (p *priorityScheduler) Schedule(rp *ResourcePool) ([]*sproto.AllocateRequest, []*actor.Ref) {
	return p.prioritySchedule(rp.taskList, rp.groups, rp.agents, rp.fittingMethod)
}

func (p *priorityScheduler) prioritySchedule(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	agents map[*actor.Ref]*agentState,
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
				taskList, groups, agentsWithLabel, fittingMethod, taskFilter(label, zeroSlots),
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
	agents map[*actor.Ref]*agentState,
	fittingMethod SoftConstraint,
	filter func(*sproto.AllocateRequest) bool,
) ([]*sproto.AllocateRequest, []*actor.Ref) {
	toAllocate := make([]*sproto.AllocateRequest, 0)
	toRelease := make(map[*actor.Ref]bool)

	// Sort tasks by priorities and timestamps. This sort determines the order in which
	// tasks are scheduled and preempted.
	priorityToPendingTasksMap, priorityToScheduledTaskMap := sortTasksByPriorityAndTimestamp(
		taskList, groups, filter)

	// Make a local copy of the agent state that we will modify.
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
					if allocatedTask.NonPreemptible {
						continue
					}
					log.Debugf("scheduled task via backfilling: %s", allocatedTask.Name)
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
				if fits := findFits(prioritizedAllocation, localAgentsState, fittingMethod); len(fits) > 0 {
					log.Debugf(
						"Not preempting tasks for task %s as it will be able to launch "+
							"once already scheduled preemptions complete", prioritizedAllocation.Name)
					addTaskToAgents(fits)
					continue
				}

				taskPlaced, updatedLocalAgentState, preemptedTasks := trySchedulingTaskViaPreemption(
					taskList, prioritizedAllocation, priority, fittingMethod, localAgentsState,
					priorityToScheduledTaskMap, toRelease, filter)

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
	fittingMethod SoftConstraint,
	agents map[*actor.Ref]*agentState,
	priorityToScheduledTaskMap map[int][]*sproto.AllocateRequest,
	tasksAlreadyPreempted map[*actor.Ref]bool,
	filter func(*sproto.AllocateRequest) bool,
) (bool, map[*actor.Ref]*agentState, map[*actor.Ref]bool) {
	localAgentsState := deepCopyAgents(agents)
	preemptedTasks := make(map[*actor.Ref]bool)
	log.Debugf("trying to schedule task %s by preempting other tasks", allocationRequest.Name)

	for priority := model.MaxUserSchedulingPriority; priority > allocationPriority; priority-- {
		for _, preemptionCandidate := range priorityToScheduledTaskMap[priority] {
			if preemptionCandidate.NonPreemptible || !filter(preemptionCandidate) {
				continue
			}

			if _, ok := tasksAlreadyPreempted[preemptionCandidate.TaskActor]; ok {
				continue
			}

			resourcesAllocated := taskList.GetAllocations(preemptionCandidate.TaskActor)
			removeTaskFromAgents(localAgentsState, resourcesAllocated)
			preemptedTasks[preemptionCandidate.TaskActor] = true

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
	agents map[*actor.Ref]*agentState,
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

// sortTasksByPriorityAndTimestamp sorts all pending and scheduled tasks
// separately by priority. Within each priority, tasks are ordered
// based on their creation time.
func sortTasksByPriorityAndTimestamp(
	taskList *taskList,
	groups map[*actor.Ref]*group,
	filter func(*sproto.AllocateRequest) bool,
) (map[int][]*sproto.AllocateRequest, map[int][]*sproto.AllocateRequest) {
	// Sort all non-zero slot tasks by priority.
	priorityToPendingTasksMap := make(map[int][]*sproto.AllocateRequest)
	priorityToScheduledTaskMap := make(map[int][]*sproto.AllocateRequest)

	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if !filter(req) {
			continue
		}

		priority := groups[req.Group].priority
		if priority == nil {
			panic(fmt.Sprintf("priority not set for task %s", req.Name))
		}

		assigned := taskList.GetAllocations(req.TaskActor)
		if assigned == nil || len(assigned.Reservations) == 0 {
			priorityToPendingTasksMap[*priority] = append(priorityToPendingTasksMap[*priority], req)
		} else {
			priorityToScheduledTaskMap[*priority] = append(priorityToScheduledTaskMap[*priority], req)
		}
	}

	// For each priority, independently sort pending and scheduled tasks by longest to shortest time of
	// existence.
	for _, tasksMap := range []map[int][]*sproto.AllocateRequest{
		priorityToPendingTasksMap, priorityToScheduledTaskMap,
	} {
		for _, tasks := range tasksMap {
			sort.Slice(tasks, func(i, j int) bool {
				return tasks[i].TaskActor.RegisteredTime().Before(tasks[j].TaskActor.RegisteredTime())
			})
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
		fit.Agent.allocateFreeDevices(fit.Slots, cproto.NewID())
	}
}

func removeTaskFromAgents(
	agents map[*actor.Ref]*agentState,
	resourcesAllocated *sproto.ResourcesAllocated,
) {
	for _, allocation := range resourcesAllocated.Reservations {
		allocation := allocation.(*containerReservation)
		if len(allocation.devices) == 0 {
			// Handle zero-slot containers.
			delete(agents[allocation.agent.handler].zeroSlotContainers, allocation.container.id)
		}

		for _, allocatedDevice := range allocation.devices {
			// Local devices are a deep copy of the originals so we loop over trying to find
			// the device that matches. If we assume that we have homogeneous devices we could
			// just search for the first used device.
			for localDevice, localContainer := range agents[allocation.agent.handler].devices {
				if allocatedDevice.ID == localDevice.ID && localContainer != nil {
					agents[allocation.agent.handler].devices[localDevice] = nil
				}
			}
		}
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

func taskFilter(label string, zeroSlots bool) func(*sproto.AllocateRequest) bool {
	return func(request *sproto.AllocateRequest) bool {
		return request.Label == label && (request.SlotsNeeded == 0) == zeroSlots
	}
}
