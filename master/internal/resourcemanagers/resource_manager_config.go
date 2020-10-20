package resourcemanagers

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/union"
)

// ResolveConfig applies backwards compatibility for the old scheduler
// and provisioner configuration.
func ResolveConfig(
	schedulerConf *Config,
	provisionerConf *provisioner.Config,
	resourceMangerConf *ResourceManagerConfig,
	resourcePoolsConf *ResourcePoolsConfig,
) (*ResourceManagerConfig, *ResourcePoolsConfig, error) {
	switch {
	case provisionerConf == nil && resourcePoolsConf == nil:
		resourcePoolsConf = DefaultRPsConfig()
	case provisionerConf != nil && resourcePoolsConf == nil:
		resourcePoolsConf = &ResourcePoolsConfig{
			ResourcePools: []ResourcePoolConfig{{PoolName: "default", Provider: provisionerConf}},
		}
	case provisionerConf != nil && resourcePoolsConf != nil:
		return nil, nil, errors.New("cannot specify both the provisioner and resource_pools fields")
	}

	switch {
	case schedulerConf == nil && resourceMangerConf == nil:
		resourceMangerConf = DefaultRMConfig()

	case schedulerConf != nil && resourceMangerConf == nil:
		switch {
		case schedulerConf.ResourceProvider == nil ||
			schedulerConf.ResourceProvider.DefaultRPConfig != nil:
			resourceMangerConf = &ResourceManagerConfig{
				AgentRM: &AgentResourceManagerConfig{
					SchedulingPolicy: schedulerConf.Type,
					FittingPolicy:    schedulerConf.Fit,
				},
			}
		case schedulerConf.ResourceProvider.KubernetesRPConfig != nil:
			resourceMangerConf = &ResourceManagerConfig{
				KubernetesRM: schedulerConf.ResourceProvider.KubernetesRPConfig,
			}
		}

	case schedulerConf != nil && resourceMangerConf != nil:
		return nil, nil, errors.New(
			"cannot specify both the scheduler and resource_manager fields")
	}

	if resourceMangerConf != nil && resourceMangerConf.AgentRM != nil {
		if resourceMangerConf.AgentRM.SchedulingPolicy == "" {
			resourceMangerConf.AgentRM.SchedulingPolicy = DefaultRMConfig().AgentRM.SchedulingPolicy
		}
		if resourceMangerConf.AgentRM.FittingPolicy == "" {
			resourceMangerConf.AgentRM.FittingPolicy = DefaultRMConfig().AgentRM.FittingPolicy
		}
	}
	return resourceMangerConf, resourcePoolsConf, nil
}

// DefaultRMConfig returns the default resource manager configuration.
func DefaultRMConfig() *ResourceManagerConfig {
	return &ResourceManagerConfig{
		AgentRM: DefaultAgentRMConfig(),
	}
}

// DefaultAgentRMConfig returns the default determined resource manager configuration.
func DefaultAgentRMConfig() *AgentResourceManagerConfig {
	return &AgentResourceManagerConfig{
		SchedulingPolicy: "fair_share",
		FittingPolicy:    "best",
	}
}

// ResourceManagerConfig hosts configuration fields for the resource manager.
type ResourceManagerConfig struct {
	AgentRM      *AgentResourceManagerConfig      `union:"type,agent" json:"-"`
	KubernetesRM *KubernetesResourceManagerConfig `union:"type,kubernetes" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (r ResourceManagerConfig) MarshalJSON() ([]byte, error) {
	return union.Marshal(r)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *ResourceManagerConfig) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, r); err != nil {
		return err
	}
	type DefaultParser *ResourceManagerConfig
	return errors.Wrap(json.Unmarshal(data, DefaultParser(r)), "failed to parse resource manager")
}

// AgentResourceManagerConfig hosts configuration fields for the determined resource manager.
type AgentResourceManagerConfig struct {
	SchedulingPolicy       string `json:"scheduling_policy"`
	FittingPolicy          string `json:"fitting_policy"`
	DefaultCPUResourcePool string `json:"default_cpu_resource_pool"`
	DefaultGPUResourcePool string `json:"default_gpu_resource_pool"`
}

// Validate implements the check.Validatable interface.
func (a AgentResourceManagerConfig) Validate() []error {
	return []error{
		check.Contains(
			a.SchedulingPolicy, []interface{}{"priority", "fair_share"}, "invalid scheduling policy",
		),
		check.Contains(
			a.FittingPolicy, []interface{}{"best", "worst"}, "invalid fitting policy",
		),
	}
}

// KubernetesResourceManagerConfig hosts configuration fields for the kubernetes resource manager.
type KubernetesResourceManagerConfig struct {
	Namespace                string `json:"namespace"`
	MaxSlotsPerPod           int    `json:"max_slots_per_pod"`
	MasterServiceName        string `json:"master_service_name"`
	LeaveKubernetesResources bool   `json:"leave_kubernetes_resources"`
}

// Validate implements the check.Validatable interface.
func (k KubernetesResourceManagerConfig) Validate() []error {
	return []error{
		check.GreaterThanOrEqualTo(k.MaxSlotsPerPod, 0, "max_slots_per_pod must be >= 0"),
	}
}
