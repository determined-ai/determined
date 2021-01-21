package resourcemanagers

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/pkg/check"
)

// DefaultRPsConfig returns the default resources pools configuration.
func DefaultRPsConfig() *ResourcePoolsConfig {
	return &ResourcePoolsConfig{
		ResourcePools: []ResourcePoolConfig{{
			PoolName:                 defaultResourcePoolName,
			MaxCPUContainersPerAgent: 100,
			Scheduler:                defaultSchedulerConfig(),
		}},
	}
}

// DefaultRPConfig returns the default resources pool configuration.
func DefaultRPConfig() *ResourcePoolConfig {
	return &ResourcePoolConfig{
		MaxCPUContainersPerAgent: 100,
		Scheduler:                defaultSchedulerConfig(),
	}
}

// ResourcePoolConfig hosts the configuration for a resource pool.
type ResourcePoolConfig struct {
	PoolName                 string              `json:"pool_name"`
	Description              string              `json:"description"`
	Provider                 *provisioner.Config `json:"provider"`
	Scheduler                *SchedulerConfig    `json:"scheduler,omitempty"`
	MaxCPUContainersPerAgent int                 `json:"max_cpu_containers_per_agent"`
}

// Validate implements the check.Validatable interface.
func (r ResourcePoolConfig) Validate() []error {
	return []error{
		check.True(len(r.PoolName) != 0, "resource pool name cannot be empty"),
		check.True(r.MaxCPUContainersPerAgent >= 0,
			"resource pool max cpu containers per agent should be >= 0"),
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *ResourcePoolConfig) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, r); err != nil {
		return err
	}
	return errors.Wrap(json.Unmarshal(data, DefaultRPConfig()), "failed to parse resource pool")
}

// ResourcePoolsConfig hosts the configuration for resource pools.
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
