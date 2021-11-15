package resourcemanagers

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/internal/resourcemanagers/provisioner"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
)

// DefaultRPConfig returns the default resources pool configuration.
func defaultRPConfig() ResourcePoolConfig {
	return ResourcePoolConfig{
		MaxAuxContainersPerAgent: 100,
		MaxCPUContainersPerAgent: -1,
	}
}

// ResourcePoolConfig hosts the configuration for a resource pool.
type ResourcePoolConfig struct {
	PoolName                 string                             `json:"pool_name"`
	Description              string                             `json:"description"`
	Provider                 *provisioner.Config                `json:"provider"`
	Scheduler                *SchedulerConfig                   `json:"scheduler,omitempty"`
	MaxAuxContainersPerAgent int                                `json:"max_aux_containers_per_agent"`
	TaskContainerDefaults    *model.TaskContainerDefaultsConfig `json:"task_container_defaults"`
	// Deprecated: Use MaxAuxContainersPerAgent instead.
	MaxCPUContainersPerAgent int `json:"max_cpu_containers_per_agent,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *ResourcePoolConfig) UnmarshalJSON(data []byte) error {
	*r = defaultRPConfig()
	type DefaultParser *ResourcePoolConfig
	if err := json.Unmarshal(data, DefaultParser(r)); err != nil {
		return err
	}

	if r.MaxCPUContainersPerAgent != -1 {
		r.MaxAuxContainersPerAgent = r.MaxCPUContainersPerAgent
	}

	r.MaxCPUContainersPerAgent = 0

	return nil
}

// Validate implements the check.Validatable interface.
func (r ResourcePoolConfig) Validate() []error {
	return []error{
		check.True(len(r.PoolName) != 0, "resource pool name cannot be empty"),
		check.True(r.MaxAuxContainersPerAgent >= 0,
			"resource pool max cpu containers per agent should be >= 0"),
	}
}
