package resourcemanagers

import (
	"time"

	"github.com/determined-ai/determined/master/internal/sproto"
	cproto "github.com/determined-ai/determined/master/pkg/container"
)

// TaskSummary contains information about a task for external display.
type TaskSummary struct {
	ID             TaskID             `json:"id"`
	Name           string             `json:"name"`
	RegisteredTime time.Time          `json:"registered_time"`
	SlotsNeeded    int                `json:"slots_needed"`
	Containers     []ContainerSummary `json:"containers"`
}

func newTaskSummary(request *AllocateRequest, allocated *ResourcesAllocated) TaskSummary {
	// Summary returns a new immutable view of the task state.
	containerSummaries := make([]ContainerSummary, 0)
	if allocated != nil {
		for _, c := range allocated.Allocations {
			containerSummaries = append(containerSummaries, c.Summary())
		}
	}
	return TaskSummary{
		ID:             request.ID,
		Name:           request.Name,
		RegisteredTime: request.TaskActor.RegisteredTime(),
		SlotsNeeded:    request.SlotsNeeded,
		Containers:     containerSummaries,
	}
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
		IsIdle: state.numUsedSlots() == 0 && len(state.zeroSlotContainers) == 0,
	}
}

func getTaskSummary(reqList *taskList, id TaskID) *TaskSummary {
	if req, ok := reqList.GetTaskByID(id); ok {
		summary := newTaskSummary(req, reqList.GetAllocations(req.TaskActor))
		return &summary
	}
	return nil
}

func getTaskSummaries(reqList *taskList) map[TaskID]TaskSummary {
	ret := make(map[TaskID]TaskSummary)
	for it := reqList.iterator(); it.next(); {
		req := it.value()
		ret[req.ID] = newTaskSummary(req, reqList.GetAllocations(req.TaskActor))
	}
	return ret
}
