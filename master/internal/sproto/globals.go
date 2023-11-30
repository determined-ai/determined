package sproto

type (
	// GetDefaultComputeResourcePoolRequest is a message asking for the name of the default
	// GPU resource pool.
	GetDefaultComputeResourcePoolRequest struct{}

	// GetDefaultComputeResourcePoolResponse is the response to
	// GetDefaultComputeResourcePoolRequest.
	GetDefaultComputeResourcePoolResponse struct {
		PoolName string
	}

	// GetDefaultAuxResourcePoolRequest is a message asking for the name of the default
	// CPU resource pool.
	GetDefaultAuxResourcePoolRequest struct{}

	// GetDefaultAuxResourcePoolResponse is the response to GetDefaultAuxResourcePoolRequest.
	GetDefaultAuxResourcePoolResponse struct {
		PoolName string
	}
)
