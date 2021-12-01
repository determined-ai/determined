package sproto

import (
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ContainerSummary contains information about a task container for external display.
type ContainerSummary struct {
	AllocationID model.AllocationID `json:"allocation_id"`
	ID           cproto.ID          `json:"id"`
	Agent        string             `json:"agent"`
	Devices      []device.Device    `json:"devices"`
}
