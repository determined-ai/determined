package config

import (
	"fmt"
)

// DefaultRMName is the default resource manager name when a user does not provide one.
const DefaultRMName = "default"

// DefaultRMIndex is the default resource manager index given a list of Resources().
const DefaultRMIndex = 0

// DefaultResourceConfig returns the default resource configuration.
func DefaultResourceConfig() *ResourceConfig {
	return &ResourceConfig{
		RootManagerInternal: &ResourceManagerConfig{},
	}
}

// ResourceManagerWithPoolsConfig is a resource manager pool config pair.
type ResourceManagerWithPoolsConfig struct {
	ResourceManager *ResourceManagerConfig `json:"resource_manager"`
	ResourcePools   []ResourcePoolConfig   `json:"resource_pools"`
}

// ResourceConfig hosts configuration fields of the resource manager and resource pools.
type ResourceConfig struct {
	// Deprecated: do not use this. All access should be through ResourceManagers().
	RootManagerInternal *ResourceManagerConfig `json:"resource_manager"`
	// Deprecated: do not use this. All access should be through ResourceManagers().
	RootPoolsInternal []ResourcePoolConfig `json:"resource_pools"`
	// Deprecated: do not use this. All access should be through ResourceManagers().
	AdditionalResourceManagersInternal []*ResourceManagerWithPoolsConfig `json:"additional_resource_managers"`
}

// ResourceManagers returns a list of resource managers and pools.
// All access should go through here and not struct items.
func (r *ResourceConfig) ResourceManagers() []*ResourceManagerWithPoolsConfig {
	return append([]*ResourceManagerWithPoolsConfig{
		{
			ResourceManager: r.RootManagerInternal,
			ResourcePools:   r.RootPoolsInternal,
		},
	}, r.AdditionalResourceManagersInternal...)
}

// GetAgentRMConfig gets the agent rm config if it exists
// and returns a bool indiciating if it exists.
func (r *ResourceConfig) GetAgentRMConfig() (*ResourceManagerWithPoolsConfig, bool) {
	for _, c := range r.ResourceManagers() {
		if c.ResourceManager.AgentRM != nil {
			return c, true
		}
	}

	return nil, false
}

func defaultAgentRM() *AgentResourceManagerConfig {
	return &AgentResourceManagerConfig{
		Name:                       DefaultRMName,
		DefaultComputeResourcePool: defaultResourcePoolName,
		DefaultAuxResourcePool:     defaultResourcePoolName,
	}
}

// ResolveResource resolves the config.
func (r *ResourceConfig) ResolveResource() error {
	if r.RootManagerInternal == nil {
		r.RootManagerInternal = &ResourceManagerConfig{
			AgentRM: defaultAgentRM(),
		}
	}

	// Add a default resource manager.
	// I'm not sure if this code could ever be true, since we default in UnmarshalJSON.
	// This feels risky to remove though.
	if r.RootManagerInternal.AgentRM == nil &&
		r.RootManagerInternal.KubernetesRM == nil &&
		r.RootManagerInternal.DispatcherRM == nil &&
		r.RootManagerInternal.PbsRM == nil {
		r.RootManagerInternal.AgentRM = defaultAgentRM()
	}
	for _, c := range r.AdditionalResourceManagersInternal {
		if c.ResourceManager.AgentRM == nil &&
			c.ResourceManager.KubernetesRM == nil &&
			c.ResourceManager.DispatcherRM == nil {
			// This error should be impossible to go off.
			return fmt.Errorf("please specify an resource manager type")
		}
	}

	// Default the name but only for the root level field.
	if r.RootManagerInternal.Name() == "" {
		r.RootManagerInternal.setName(DefaultRMName)
	}

	// Add a default resource pool for nonslurm default resource managers.
	// TODO(multirm-slurm) rethink pool discovery.
	if r.RootPoolsInternal == nil &&
		(r.RootManagerInternal.AgentRM != nil || r.RootManagerInternal.KubernetesRM != nil) {
		defaultPool := defaultRPConfig()

		defaultPool.PoolName = defaultResourcePoolName
		r.RootPoolsInternal = []ResourcePoolConfig{defaultPool}
	}

	return nil
}

// Validate implements the check.Validatable interface.
func (r ResourceConfig) Validate() []error {
	seenResourceManagerNames := make(map[string]bool)
	poolNames := make(map[string]bool)
	var errs []error
	for _, r := range r.ResourceManagers() {
		// All non slurm resource managers must have a resource pool.
		if len(r.ResourcePools) == 0 &&
			(r.ResourceManager.AgentRM != nil || r.ResourceManager.KubernetesRM != nil) {
			errs = append(errs, fmt.Errorf(
				"for additional_resource_managers, you must specify at least one resource pool"))
		}

		name := r.ResourceManager.Name()
		if _, ok := seenResourceManagerNames[name]; ok {
			errs = append(errs, fmt.Errorf("resource manager has a duplicate name: %s", name))
		}
		seenResourceManagerNames[name] = true

		rmPoolNames := make(map[string]bool)
		for _, rp := range r.ResourcePools {
			if _, ok := poolNames[rp.PoolName]; ok {
				if _, ok := rmPoolNames[rp.PoolName]; ok {
					errs = append(errs, fmt.Errorf(
						"resource pool has a duplicate name: %s", rp.PoolName))
				} else {
					errs = append(errs, fmt.Errorf("resource pool has a duplicate name: %s "+
						"They must be unique across even different resource managers", rp.PoolName))
				}
			}

			rmPoolNames[rp.PoolName] = true
			poolNames[rp.PoolName] = true
		}
	}

	for _, r := range r.AdditionalResourceManagersInternal {
		if r.ResourceManager.KubernetesRM == nil {
			errs = append(errs, fmt.Errorf(
				"additional_resource_managers only supports resource managers of type: kubernetes"))
		}
	}

	return errs
}
