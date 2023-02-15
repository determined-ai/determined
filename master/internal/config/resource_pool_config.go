package config

import (
	"encoding/json"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
)

// DefaultRPConfig returns the default resources pool configuration.
func defaultRPConfig() ResourcePoolConfig {
	return ResourcePoolConfig{
		MaxAuxContainersPerAgent: 100,
		MaxCPUContainersPerAgent: -1,
		AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
	}
}

// ResourcePoolConfig hosts the configuration for a resource pool.
type ResourcePoolConfig struct {
	PoolName                 string                             `json:"pool_name"`
	Description              string                             `json:"description"`
	Provider                 *provconfig.Config                 `json:"provider"`
	Scheduler                *SchedulerConfig                   `json:"scheduler,omitempty"`
	MaxAuxContainersPerAgent int                                `json:"max_aux_containers_per_agent"`
	TaskContainerDefaults    *model.TaskContainerDefaultsConfig `json:"task_container_defaults"`
	// AgentReattachEnabled defines if master & agent try to recover
	// running containers after a clean restart.
	AgentReattachEnabled bool `json:"agent_reattach_enabled"`
	// AgentReconnectWait define the time master will wait for agent
	// before abandoning it.
	AgentReconnectWait model.Duration `json:"agent_reconnect_wait"`

	// If empty, will behave as if the value is resource_manager.namespace,
	// which in most cases will be the namespace the helm deployment is in.
	KubernetesNamespace string `json:"kubernetes_namespace"`

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
