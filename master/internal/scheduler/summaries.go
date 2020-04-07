package scheduler

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/agent"
)

// TaskSummary contains information about a task for external display.
type TaskSummary struct {
	ID             TaskID             `json:"id"`
	State          string             `json:"state"`
	Name           string             `json:"name"`
	RegisteredTime time.Time          `json:"registered_time"`
	SlotsNeeded    int                `json:"slots_needed"`
	Containers     []ContainerSummary `json:"containers"`
}

func (summary1 *TaskSummary) equals(summary2 *TaskSummary) bool {
	if summary1.ID != summary2.ID ||
		summary1.State != summary2.State ||
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

// AgentSummary contains information about an agent for external display.
type AgentSummary struct {
	Name string `json:"name"`
}

// ContainerSummary contains information about a task container for external display.
type ContainerSummary struct {
	TaskID     TaskID                  `json:"task_id"`
	ID         ContainerID             `json:"id"`
	State      string                  `json:"state"`
	Agent      string                  `json:"agent"`
	ExitStatus *agent.ContainerStopped `json:"exit_status"`
	IsLeader   bool                    `json:"is_leader"`
}

// newAgentSummary returns a new immutable view of the agent.
func newAgentSummary(state *agentState) AgentSummary {
	return AgentSummary{
		Name: state.handler.Address().Local(),
	}
}

// newTaskSummary returns a new immutable view of the task state.
func newTaskSummary(task *Task) TaskSummary {
	containerSummaries := make([]ContainerSummary, 0, len(task.containers))
	for _, c := range task.containers {
		containerSummaries = append(containerSummaries, newContainerSummary(c))
	}
	return TaskSummary{
		ID:             task.ID,
		Name:           task.name,
		RegisteredTime: task.handler.RegisteredTime(),
		State:          string(task.state),
		SlotsNeeded:    task.SlotsNeeded(),
		Containers:     containerSummaries,
	}
}

// newContainerSummary returns a snapshot view of the container state.
func newContainerSummary(c *container) ContainerSummary {
	return ContainerSummary{
		TaskID:     c.task.ID,
		ID:         c.id,
		State:      string(c.state),
		Agent:      c.agent.handler.Address().Local(),
		ExitStatus: c.exitStatus,
		IsLeader:   c.IsLeader(),
	}
}
