package resourcemanagers

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/pkg/check"
)

// DefaultRPsConfig returns the default resources pools configuration.
func DefaultRPsConfig() *ResourcePoolsConfig {
	return &ResourcePoolsConfig{
		ResourcePools: []ResourcePoolConfig{{PoolName: defaultResourcePoolName}},
	}
}

// ResourcePoolConfig hosts the configuration for a resource pool
type ResourcePoolConfig struct {
	PoolName    string              `json:"pool_name"`
	Description string              `json:"description"`
	Provider    *provisioner.Config `json:"provider"`
}

// Validate implements the check.Validatable interface.
func (r ResourcePoolConfig) Validate() []error {
	return []error{
		check.True(len(r.PoolName) != 0, "resource pool name cannot be empty"),
	}
}

// ResourcePoolsConfig hosts the configuration for resource pools
type ResourcePoolsConfig struct {
	ResourcePools []ResourcePoolConfig `json:"resource_pools"`
}

// Validate implements the check.Validatable interface.
func (r ResourcePoolsConfig) Validate() []error {
	errs := make([]error, 0)
	poolNames := make(map[string]bool)
	for ix, rp := range r.ResourcePools {
		if _, ok := poolNames[rp.PoolName]; ok {
			errs = append(errs, errors.Errorf("%d resource pool has a duplicate name: %s", ix, rp.PoolName))
		} else {
			poolNames[rp.PoolName] = true
		}
	}
	return errs
}
