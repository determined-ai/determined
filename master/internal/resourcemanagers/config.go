package resourcemanagers

import "github.com/pkg/errors"

// DefaultResourceConfig returns the default resource configuration.
func DefaultResourceConfig() *ResourceConfig {
	return &ResourceConfig{
		ResourceManager: &ResourceManagerConfig{},
	}
}

// ResourceConfig hosts configuration fields of the resource manager and resource pools.
type ResourceConfig struct {
	ResourceManager *ResourceManagerConfig `json:"resource_manager"`
	ResourcePools   []ResourcePoolConfig   `json:"resource_pools"`
}

// ResolveResource resolves the config.
func (r *ResourceConfig) ResolveResource() error {
	if r.ResourceManager == nil {
		r.ResourceManager = &ResourceManagerConfig{
			AgentRM: &AgentResourceManagerConfig{},
		}
	}
	if r.ResourceManager.AgentRM == nil {
		r.ResourceManager.AgentRM = &AgentResourceManagerConfig{}
	}
	if r.ResourcePools == nil {
		r.ResourcePools = []ResourcePoolConfig{
			{
				PoolName:                 defaultResourcePoolName,
				MaxCPUContainersPerAgent: 100,
			},
		}
	}
	return nil
}

// Validate implements the check.Validatable interface.
func (r ResourceConfig) Validate() []error {
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
