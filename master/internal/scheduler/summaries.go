package scheduler

import (
	"time"
)

// TaskSummary contains information about a task for external display.
type TaskSummary struct {
	ID             RequestID          `json:"id"`
	Name           string             `json:"name"`
	RegisteredTime time.Time          `json:"registered_time"`
	SlotsNeeded    int                `json:"slots_needed"`
	Containers     []ContainerSummary `json:"containers"`
}

func (summary1 *TaskSummary) equals(summary2 *TaskSummary) bool {
	if summary1.ID != summary2.ID ||
		summary1.Name != summary2.Name ||
		summary1.RegisteredTime != summary2.RegisteredTime ||
		summary1.SlotsNeeded != summary2.SlotsNeeded {
		return false
	}

	if len(summary1.Containers) != len(summary2.Containers) {
		return false
	}

	containers := make(map[ContainerID]*ContainerSummary)
	for i := 0; i < len(summary1.Containers); i++ {
		c := summary1.Containers[i]
		containers[c.ID] = &c
	}

	for _, c2 := range summary2.Containers {
		if c, ok := containers[c2.ID]; !ok || *c != c2 {
			return false
		}
	}

	return true
}

func newTaskSummary(request *AssignRequest, assigned *ResourceAssigned) TaskSummary {
	// Summary returns a new immutable view of the task state.
	containerSummaries := make([]ContainerSummary, 0)
	if assigned != nil {
		for _, c := range assigned.Assignments {
			containerSummaries = append(containerSummaries, c.Summary())
		}
	}
	return TaskSummary{
		ID:             request.ID,
		Name:           request.Name,
		RegisteredTime: request.Handler.RegisteredTime(),
		SlotsNeeded:    request.SlotsNeeded,
		Containers:     containerSummaries,
	}
}

// AgentSummary contains information about an agent for external display.
type AgentSummary struct {
	Name string `json:"name"`
}

func (summary1 *AgentSummary) equals(summary2 *AgentSummary) bool {
	return summary1.Name == summary2.Name
}

// ContainerSummary contains information about a task container for external display.
type ContainerSummary struct {
	TaskID RequestID   `json:"task_id"`
	ID     ContainerID `json:"id"`
	Agent  string      `json:"agent"`
}

// newAgentSummary returns a new immutable view of the agent.
func newAgentSummary(state *agentState) AgentSummary {
	return AgentSummary{
		Name: state.handler.Address().Local(),
	}
}

func getTaskSummary(reqList *assignRequestList, id RequestID) *TaskSummary {
	if req, ok := reqList.GetByID(id); ok {
		summary := newTaskSummary(req, reqList.GetAssignments(req.Handler))
		return &summary
	}
	return nil
}

func getTaskSummaries(reqList *assignRequestList) map[RequestID]TaskSummary {
	ret := make(map[RequestID]TaskSummary)
	for it := reqList.iterator(); it.next(); {
		req := it.value()
		ret[req.ID] = newTaskSummary(req, reqList.GetAssignments(req.Handler))
	}
	return ret
}
