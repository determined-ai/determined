package resourcemanagers

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/union"
)

const defaultResourcePoolName = "default"

// ResolveConfig applies backwards compatibility for the old scheduler
// and provisioner configuration.
func ResolveConfig(
	schedulerConf *Config,
	provisionerConf *provisioner.Config,
	resourceManagerConf *ResourceManagerConfig,
	resourcePoolsConf *ResourcePoolsConfig,
) (*ResourceManagerConfig, *ResourcePoolsConfig, error) {
	switch {
	case provisionerConf == nil && resourcePoolsConf == nil:
		resourcePoolsConf = DefaultRPsConfig()
	case provisionerConf != nil && resourcePoolsConf == nil:
		resourcePoolsConf = &ResourcePoolsConfig{
			ResourcePools: []ResourcePoolConfig{
				{PoolName: defaultResourcePoolName, Provider: provisionerConf},
			},
		}
	case provisionerConf != nil && resourcePoolsConf != nil:
		return nil, nil, errors.New("cannot specify both the provisioner and resource_pools fields")
	}

	switch {
	case schedulerConf == nil && resourceManagerConf == nil:
		resourceManagerConf = DefaultRMConfig()

	case schedulerConf != nil && resourceManagerConf == nil:
		switch {
		case schedulerConf.ResourceProvider == nil ||
			schedulerConf.ResourceProvider.DefaultRPConfig != nil:

			// Fill in defaults for the old scheduler format.
			if schedulerConf.Type == "" {
				schedulerConf.Type = fairShareScheduling
			}
			if schedulerConf.Fit == "" {
				schedulerConf.Fit = defaultFitPolicy
			}
			schedulerPolicyConf := &SchedulerConfig{
				FittingPolicy: schedulerConf.Fit,
			}

			switch schedulerConf.Type {
			case fairShareScheduling:
				schedulerPolicyConf.FairShare = &FairShareSchedulerConfig{}
			case priorityScheduling:
				schedulerPolicyConf.Priority = &PrioritySchedulerConfig{
					Preemption: schedulerConf.Preemption,
				}

				if schedulerConf.DefaultPriority != nil {
					schedulerPolicyConf.Priority.DefaultPriority = schedulerConf.DefaultPriority
				}
			default:
				return nil, nil, errors.Errorf(
					"unsupported scheduler type %s; "+
						"scheduler type must be `fair_share` or `priority`", schedulerConf.Type)
			}

			resourceManagerConf = &ResourceManagerConfig{
				AgentRM: &AgentResourceManagerConfig{
					Scheduler:              schedulerPolicyConf,
					DefaultCPUResourcePool: defaultResourcePoolName,
					DefaultGPUResourcePool: defaultResourcePoolName,
				},
			}
		case schedulerConf.ResourceProvider.KubernetesRPConfig != nil:
			resourceManagerConf = &ResourceManagerConfig{
				KubernetesRM: schedulerConf.ResourceProvider.KubernetesRPConfig,
			}
		}

	case schedulerConf != nil && resourceManagerConf != nil:
		return nil, nil, errors.New(
			"cannot specify both the scheduler and resource_manager fields")
	}

	if resourceManagerConf != nil && resourceManagerConf.AgentRM != nil {
		if resourceManagerConf.AgentRM.Scheduler == nil {
			resourceManagerConf.AgentRM.Scheduler = defaultSchedulerConfig()
		}

		// Fill in default fitting policy if unspecified.
		if resourceManagerConf.AgentRM.Scheduler.FittingPolicy == "" {
			resourceManagerConf.AgentRM.Scheduler.FittingPolicy = defaultFitPolicy
		}

		// Set default scheduling priority if it is not specified and priority scheduler
		// is being used.
		prioritySchedulerConf := resourceManagerConf.AgentRM.Scheduler.Priority
		if prioritySchedulerConf != nil && prioritySchedulerConf.DefaultPriority == nil {
			defaultPriority := DefaultSchedulingPriority
			prioritySchedulerConf.DefaultPriority = &defaultPriority
		}

		// If a pool specifies a scheduler unique for that pool we overwrite the
		// scheduler specified as part of the ResourceManager config and replace
		// it completely. We go through the pools if they specify a scheduler and
		// fill in defaults the same way we do for the ResourceManager.
		if resourcePoolsConf != nil {
			for _, resourcePool := range resourcePoolsConf.ResourcePools {
				if resourcePool.Scheduler == nil {
					continue
				}

				if resourcePool.Scheduler.FittingPolicy == "" {
					resourcePool.Scheduler.FittingPolicy = defaultFitPolicy
				}

				if resourcePool.Scheduler.Priority == nil {
					continue
				}

				if resourcePool.Scheduler.Priority.DefaultPriority == nil {
					defaultPriority := DefaultSchedulingPriority
					resourcePool.Scheduler.Priority.DefaultPriority = &defaultPriority
				}
			}
		}
	}

	return resourceManagerConf, resourcePoolsConf, nil
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
		Scheduler:              defaultSchedulerConfig(),
		DefaultGPUResourcePool: defaultResourcePoolName,
		DefaultCPUResourcePool: defaultResourcePoolName,
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
	Scheduler              *SchedulerConfig `json:"scheduler"`
	DefaultCPUResourcePool string           `json:"default_cpu_resource_pool"`
	DefaultGPUResourcePool string           `json:"default_gpu_resource_pool"`
}

// Validate implements the check.Validatable interface.
func (a AgentResourceManagerConfig) Validate() []error {
	return []error{
		check.NotEmpty(a.DefaultCPUResourcePool, "default_cpu_resource_pool should be non-empty"),
		check.NotEmpty(a.DefaultGPUResourcePool, "default_gpu_resource_pool should be non-empty"),
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
