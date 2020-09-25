package scheduler

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
	sConf *Config,
	pConf *provisioner.Config,
	rmConf *ResourceManagerConfig,
	rpsConf *ResourcePoolsConfig,
) (*ResourceManagerConfig, *ResourcePoolsConfig, error) {
	switch {
	case pConf == nil && rpsConf == nil:
		rpsConf = DefaultRPsConfig()
	case pConf != nil && rpsConf == nil:
		rpsConf = &ResourcePoolsConfig{
			ResourcePools: []ResourcePoolConfig{{PoolName: "default", Provider: pConf}},
		}
	case pConf != nil && rpsConf != nil:
		return nil, nil, errors.New("cannot specify both the provisioner and resource_pools fields")
	}

	switch {
	case sConf == nil && rmConf == nil:
		rmConf = DefaultRMConfig()

	case sConf != nil && rmConf == nil:
		switch {
		case sConf.ResourceProvider == nil ||
			sConf.ResourceProvider.DefaultRPConfig != nil:
			rmConf = &ResourceManagerConfig{
				DeterminedRM: &DeterminedResourceManagerConfig{
					SchedulingPolicy: sConf.Type,
					FittingPolicy:    sConf.Fit,
				},
			}
		case sConf.ResourceProvider.KubernetesRPConfig != nil:
			rmConf = &ResourceManagerConfig{
				KubernetesRM: &KubernetesResourceManagerConfig{
					Namespace:                sConf.ResourceProvider.KubernetesRPConfig.Namespace,
					MaxSlotsPerPod:           sConf.ResourceProvider.KubernetesRPConfig.MaxSlotsPerPod,
					MasterServiceName:        sConf.ResourceProvider.KubernetesRPConfig.MasterServiceName,
					LeaveKubernetesResources: sConf.ResourceProvider.KubernetesRPConfig.LeaveKubernetesResources,
				},
			}
		}

	case sConf != nil && rmConf != nil:
		return nil, nil, errors.New("cannot specify both the scheduler and resource_manager fields")
	}
	return rmConf, rpsConf, nil
}

// DefaultRMConfig returns the default resource manager configuration.
func DefaultRMConfig() *ResourceManagerConfig {
	return &ResourceManagerConfig{
		DeterminedRM: DefaultDetRMConfig(),
	}
}

// DefaultDetRMConfig returns the default determined resource manager configuration.
func DefaultDetRMConfig() *DeterminedResourceManagerConfig {
	return &DeterminedResourceManagerConfig{
		SchedulingPolicy: "fair_share",
		FittingPolicy:    "best",
	}
}

// ResourceManagerConfig hosts configuration fields for the resource manager.
type ResourceManagerConfig struct {
	DeterminedRM *DeterminedResourceManagerConfig `union:"type,default" json:"-"`
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

// DeterminedResourceManagerConfig hosts configuration fields for the determined resource manager.
type DeterminedResourceManagerConfig struct {
	SchedulingPolicy       string `json:"scheduling_policy"`
	FittingPolicy          string `json:"fitting_policy"`
	DefaultCPUResourcePool string `json:"default_cpu_resource_pool"`
	DefaultGPUResourcePool string `json:"default_gpu_resource_pool"`
}

// Validate implements the check.Validatable interface.
func (c DeterminedResourceManagerConfig) Validate() []error {
	return []error{
		check.Contains(
			c.SchedulingPolicy, []interface{}{"priority", "fair_share"}, "invalid scheduling policy",
		),
		check.Contains(
			c.FittingPolicy, []interface{}{"best", "worst"}, "invalid fitting policy",
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
