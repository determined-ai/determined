package sproto

import "github.com/determined-ai/determined/master/pkg/model"

type (
	// CapacityCheck checks the potential available slots in a resource pool.
	CapacityCheck struct {
		Slots  int
		TaskID *model.TaskID
	}
	// CapacityCheckResponse is the response to a CapacityCheck message.
	CapacityCheckResponse struct {
		SlotsAvailable   int
		CapacityExceeded bool
	}
)
