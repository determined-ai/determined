package resourcemanagers

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/pkg/check"
)

// defaultRPConfig returns the default resources pool configuration.
func defaultRPConfig() *ResourcePoolConfig {
	return &ResourcePoolConfig{
		MaxCPUContainersPerAgent: 100,
	}
}

// ResourcePoolConfig hosts the configuration for a resource pool
type ResourcePoolConfig struct {
	PoolName                 string              `json:"pool_name"`
	Description              string              `json:"description"`
	Provider                 *provisioner.Config `json:"provider"`
	Scheduler                *SchedulerConfig    `json:"scheduler,omitempty"`
	MaxCPUContainersPerAgent int                 `json:"max_cpu_containers_per_agent"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *ResourcePoolConfig) UnmarshalJSON(data []byte) error {
	*r = *defaultRPConfig()
	type DefaultParser *ResourcePoolConfig
	return json.Unmarshal(data, DefaultParser(r))
}

// Validate implements the check.Validatable interface.
func (r ResourcePoolConfig) Validate() []error {
	return []error{
		check.True(len(r.PoolName) != 0, "resource pool name cannot be empty"),
		check.True(r.MaxCPUContainersPerAgent >= 0,
			"resource pool max cpu containers per agent should be >= 0"),
	}
}
