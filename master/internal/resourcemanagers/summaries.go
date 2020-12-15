package resourcemanagers

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/actor"

	"github.com/determined-ai/determined/master/internal/sproto"
	cproto "github.com/determined-ai/determined/master/pkg/container"
)

// TaskSummary contains information about a task for external display.
type TaskSummary struct {
	ID             TaskID             `json:"id"`
	Name           string             `json:"name"`
	RegisteredTime time.Time          `json:"registered_time"`
	ResourcePool   string             `json:"resource_pool"`
	SlotsNeeded    int                `json:"slots_needed"`
	Containers     []ContainerSummary `json:"containers"`
	SchedulerType  string             `json:"scheduler_type"`
	Priority       *int               `json:"priority"`
}

func newTaskSummary(
	request *AllocateRequest,
	allocated *ResourcesAllocated,
	groups map[*actor.Ref]*group,
	schedulerType string,
) TaskSummary {
	// Summary returns a new immutable view of the task state.
	containerSummaries := make([]ContainerSummary, 0)
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

// ContainerSummary contains information about a task container for external display.
type ContainerSummary struct {
	TaskID TaskID    `json:"task_id"`
	ID     cproto.ID `json:"id"`
	Agent  string    `json:"agent"`
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
	id TaskID,
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
) map[TaskID]TaskSummary {
	ret := make(map[TaskID]TaskSummary)
	for it := reqList.iterator(); it.next(); {
		req := it.value()
		ret[req.ID] = newTaskSummary(req, reqList.GetAllocations(req.TaskActor), groups, schedulerType)
	}
	return ret
}

type ResourceSummary struct {
	numAgents int
	numTotalSlots int
	numActiveSlots int
	maxNumCpuContainers int
	numActiveCpuContainers int
}

func getResourceSummary(
	agentInfo  map[*actor.Ref]*agentState,
) ResourceSummary {
	summary := ResourceSummary{
		numTotalSlots:          0,
		numActiveSlots:         0,
		maxNumCpuContainers:    0,
		numActiveCpuContainers: 0,
	}
	for _, agentState  := range agentInfo {
		summary.numAgents += 1
		summary.numTotalSlots += agentState.numSlots()
		summary.numActiveSlots += agentState.numUsedSlots()
		summary.maxNumCpuContainers += agentState.maxZeroSlotContainers
		summary.numActiveCpuContainers += agentState.numZeroSlotContainers()
	}
	return summary
}
