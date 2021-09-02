package sproto

import (
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ContainerSummary contains information about a task container for external display.
type ContainerSummary struct {
	AllocationID model.AllocationID `json:"allocation_id"`
	ID           cproto.ID          `json:"id"`
	Agent        string             `json:"agent"`
}
