package config

import (
	"fmt"
)

// DefaultRMName is the default resource manager name when a user does not provide one.
const DefaultRMName = "default"

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
	if r.RootManagerInternal.AgentRM == nil && r.RootManagerInternal.KubernetesRM == nil {
		r.RootManagerInternal.AgentRM = defaultAgentRM()
	}
	for _, c := range r.AdditionalResourceManagersInternal {
		if c.ResourceManager.AgentRM == nil && c.ResourceManager.KubernetesRM == nil {
			c.ResourceManager.AgentRM = defaultAgentRM()
		}
	}

	// Default the name but only for the root level field.
	if r.RootManagerInternal.Name() == "" {
		r.RootManagerInternal.setName(DefaultRMName)
	}

	// Add a default resource pool for all non resource managers.
	if r.RootPoolsInternal == nil &&
		(r.RootManagerInternal.AgentRM != nil || r.RootManagerInternal.KubernetesRM != nil) {
		defaultPool := defaultRPConfig()

		defaultPool.PoolName = defaultResourcePoolName
		r.RootPoolsInternal = []ResourcePoolConfig{defaultPool}
	}
	for _, c := range r.AdditionalResourceManagersInternal {
		if c.ResourcePools == nil &&
			(c.ResourceManager.AgentRM != nil || c.ResourceManager.KubernetesRM != nil) {
			defaultPool := defaultRPConfig()
			defaultPool.PoolName = defaultResourcePoolName
			c.ResourcePools = []ResourcePoolConfig{defaultPool}
		}
	}

	return nil
}

// Validate implements the check.Validatable interface.
func (r ResourceConfig) Validate() []error {
	agentRMCount := 0
	seenResourceManagerNames := map[string]bool{}

	var errs []error
	for _, r := range r.ResourceManagers() {
		name := r.ResourceManager.Name()
		if _, ok := seenResourceManagerNames[name]; ok {
			errs = append(errs, fmt.Errorf("resource manager has a duplicate name: %s", name))
		}
		seenResourceManagerNames[name] = true

		if r.ResourceManager.AgentRM != nil {
			agentRMCount++
		}

		poolNames := make(map[string]bool)
		for _, rp := range r.ResourcePools {
			if _, ok := poolNames[rp.PoolName]; ok {
				errs = append(errs, fmt.Errorf(
					"resource pool has a duplicate name: %s", rp.PoolName))
			} else {
				poolNames[rp.PoolName] = true
			}
		}
	}

	if agentRMCount > 1 {
		errs = append(errs, fmt.Errorf("got %d total agent resource managers, "+
			"only a single agent resource manager is supported. Please use multiple "+
			"resource pools if you want to do something similar", agentRMCount))
	}

	return errs
}
