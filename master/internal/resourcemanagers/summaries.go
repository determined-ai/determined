package resourcemanagers

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/actor"

	"github.com/determined-ai/determined/master/internal/sproto"
)

// TaskSummary contains information about a task for external display.
type TaskSummary struct {
	ID             sproto.TaskID             `json:"id"`
	Name           string                    `json:"name"`
	RegisteredTime time.Time                 `json:"registered_time"`
	ResourcePool   string                    `json:"resource_pool"`
	SlotsNeeded    int                       `json:"slots_needed"`
	Containers     []sproto.ContainerSummary `json:"containers"`
	SchedulerType  string                    `json:"scheduler_type"`
	Priority       *int                      `json:"priority"`
}

func newTaskSummary(
	request *sproto.AllocateRequest,
	allocated *sproto.ResourcesAllocated,
	groups map[*actor.Ref]*group,
	schedulerType string,
) TaskSummary {
	// Summary returns a new immutable view of the task state.
	containerSummaries := make([]sproto.ContainerSummary, 0)
	if allocated != nil {
		for _, c := range allocated.Allocations {
			containerSummaries = append(containerSummaries, c.Summary())
		}
	}
	summary := TaskSummary{
		ID:             request.ID,
		Name:           request.Name,
		RegisteredTime: request.TaskActor.RegisteredTime(),
		ResourcePool:   request.ResourcePool,
		SlotsNeeded:    request.SlotsNeeded,
		Containers:     containerSummaries,
		SchedulerType:  schedulerType,
	}

	if group, ok := groups[request.Group]; ok {
		summary.Priority = group.priority
	}
	return summary
}

// newAgentSummary returns a new immutable view of the agent.
func newAgentSummary(state *agentState) sproto.AgentSummary {
	return sproto.AgentSummary{
		Name:   state.handler.Address().Local(),
		IsIdle: state.idle(),
	}
}

func getTaskSummary(
	reqList *taskList,
	id sproto.TaskID,
	groups map[*actor.Ref]*group,
	schedulerType string,
) *TaskSummary {
	if req, ok := reqList.GetTaskByID(id); ok {
		summary := newTaskSummary(req, reqList.GetAllocations(req.TaskActor), groups, schedulerType)
		return &summary
	}
	return nil
}

func getTaskSummaries(
	reqList *taskList,
	groups map[*actor.Ref]*group,
	schedulerType string,
) map[sproto.TaskID]TaskSummary {
	ret := make(map[sproto.TaskID]TaskSummary)
	for it := reqList.iterator(); it.next(); {
		req := it.value()
		ret[req.ID] = newTaskSummary(req, reqList.GetAllocations(req.TaskActor), groups, schedulerType)
	}
	return ret
}

// ResourceSummary is a summary of the resource available/used by a resource pool.
type ResourceSummary struct {
	numAgents              int
	numTotalSlots          int
	numActiveSlots         int
	maxNumCPUContainers    int
	numActiveCPUContainers int
}

func getResourceSummary(
	agentInfo map[*actor.Ref]*agentState,
) ResourceSummary {
	summary := ResourceSummary{
		numTotalSlots:          0,
		numActiveSlots:         0,
		maxNumCPUContainers:    0,
		numActiveCPUContainers: 0,
	}
	for _, agentState := range agentInfo {
		summary.numAgents++
		summary.numTotalSlots += agentState.numSlots()
		summary.numActiveSlots += agentState.numUsedSlots()
		summary.maxNumCPUContainers += agentState.maxZeroSlotContainers
		summary.numActiveCPUContainers += agentState.numZeroSlotContainers()
	}
	return summary
}
