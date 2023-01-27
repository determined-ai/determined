package sproto

type (
	// ResourcePoolAvailabilityRequest checks for warnings trying to add a job to a resource pool.
	ResourcePoolAvailabilityRequest struct {
		PoolName string
		Slots    int
		Label    string
	}
	// CapacityCheck checks the potential available slots in a resource pool.
	CapacityCheck struct {
		Slots int
	}
	// CapacityCheckResponse is the response to a CapacityCheck message.
	CapacityCheckResponse struct {
		SlotsAvailable   int
		CapacityExceeded bool
	}
	// HasAgentWithLabel checks and returns true if a pool has an agent with a matching label.
	HasAgentWithLabel struct {
		Label string
	}
)
