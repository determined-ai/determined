package sproto

type (
	// CapacityCheck checks the potential available slots in a resource pool.
	CapacityCheck struct {
		Slots int
	}
	// CapacityCheckResponse is the response to a CapacityCheck message.
	CapacityCheckResponse struct {
		SlotsAvailable   int
		CapacityExceeded bool
	}
)
