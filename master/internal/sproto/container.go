package sproto

import cproto "github.com/determined-ai/determined/master/pkg/container"

// ContainerSummary contains information about a task container for external display.
type ContainerSummary struct {
	TaskID TaskID    `json:"task_id"`
	ID     cproto.ID `json:"id"`
	Agent  string    `json:"agent"`
}
