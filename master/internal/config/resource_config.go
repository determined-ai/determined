package config

import (
	"fmt"
)

// DefaultResourceConfig returns the default resource configuration.
func DefaultResourceConfig() *ResourceConfig {
	return &ResourceConfig{}
}

// ResourceConfig hosts configuration fields of the resource manager and resource pools.
type ResourceConfig struct {
	ResourceManagers ResourceManagersConfig `json:"resource_managers"`
	// Deprecated: do not use this. If old config is specified it will be parsed into
	// ResourceManager so all access should happen through ResourceManager.
	ResourceManagerV0DontUse *ResourceManagerConfigV0 `json:"resource_manager,omitempty"`

	// Deprecated: do not use this. If old config is specified it will be parsed as ResourcePools
	// under each resource manager struct.
	ResourcePoolsDontUse []ResourcePoolConfig `json:"resource_pools"`
}

// ResourceManagersConfig is config for a list of resource managers.
type ResourceManagersConfig []*ResourceManagerConfigV1

// GetAgentRMConfig gets the agent rm config if it exists
// and returns a bool indiciating if it exists.
func (r ResourceManagersConfig) GetAgentRMConfig() (*AgentResourceManagerConfigV1, bool) {
	for _, c := range r {
		if c.AgentRM != nil {
			return c.AgentRM, true
		}
	}

	return nil, false
}

var (
	errBothRMAndRMsGiven = fmt.Errorf("both `resource_managers` and `resource_manager` specified, " +
		"please only specify `resource_managers`")

	errMoreThanOneRMAndRootPoolsGiven = fmt.Errorf("root level `resource_pools:` and more than one " +
		"resource manager specified, please specify resource_pools under each resource manager")

	errBothPoolsGiven = fmt.Errorf("root level `resource_pools:` and `resource_pools` under " +
		"resource_managers specified, please only specify resource_pools under resource_managers")

	errMultipleAgentRMsGiven = fmt.Errorf("only a single `resource_manager` of type agent " +
		"may be specified, please use multiple resource pools if you want to do something similar")
)

func defaultRMsConfig() ResourceManagersConfig {
	poolConfig := defaultRPConfig()
	poolConfig.PoolName = defaultRPName

	//nolint:exhaustruct
	return ResourceManagersConfig{
		{
			AgentRM: &AgentResourceManagerConfigV1{
				Name:                       defaultRMName,
				Scheduler:                  DefaultSchedulerConfig(),
				ResourcePools:              []ResourcePoolConfig{poolConfig},
				DefaultAuxResourcePool:     defaultRPName,
				DefaultComputeResourcePool: defaultRPName,
			},
		},
	}
}

// ResolveResource resolves the config.
func (r *ResourceConfig) ResolveResource(oldPools []ResourcePoolConfig) error {
	// Validate so config is either v0 or v1 and not both.
	if len(r.ResourceManagers) > 0 && r.ResourceManagerV0DontUse != nil {
		return errBothRMAndRMsGiven
	}
	if len(r.ResourceManagers) > 1 && len(oldPools) > 0 {
		return errMoreThanOneRMAndRootPoolsGiven
	}
	if len(r.ResourceManagers) == 1 && len(r.ResourceManagers[0].Pools()) > 0 && len(oldPools) > 0 {
		return errBothPoolsGiven
	}

	// Port v0 config to v1.
	if r.ResourceManagerV0DontUse != nil {
		r.ResourceManagers = ResourceManagersConfig{
			r.ResourceManagerV0DontUse.ToV1(),
		}
	}

	// Add defaults.
	if len(r.ResourceManagers) == 1 && r.ResourceManagers[0].Name() == "" {
		r.ResourceManagers[0].setName(defaultRMName)
	}
	if len(r.ResourceManagers) == 0 {
		r.ResourceManagers = defaultRMsConfig()
	}

	// Set v0 pools.
	if len(r.ResourceManagers) == 1 && len(oldPools) > 0 {
		r.ResourceManagers[0].setPools(oldPools)
	}

	agentRMCount := 0
	for _, rm := range r.ResourceManagers {
		if rm.AgentRM != nil {
			agentRMCount++
		}

		if rm.AgentRM != nil && rm.AgentRM.Scheduler == nil {
			rm.AgentRM.Scheduler = DefaultSchedulerConfig()
		}

		// Add a default pool.
		// (rm.AgentRM != nil || rm.KubernetesRM != nil) is always true for OSS,
		// however the logic is just that Dispatcher doesn't want default pools since it auto
		// discovers them based on slurm queues.
		if (rm.AgentRM != nil || rm.KubernetesRM != nil) && len(rm.Pools()) == 0 {
			defaultPool := defaultRPConfig()
			defaultPool.PoolName = defaultRPName
			rm.setPools([]ResourcePoolConfig{defaultPool})
		}
	}

	// We don't support multiple agent resource managers.
	if agentRMCount > 1 {
		return errMultipleAgentRMsGiven
	}

	return nil
}

// Validate implements the check.Validatable interface.
func (r ResourceConfig) Validate() []error {
	errs := make([]error, 0)

	rmNames := make(map[string]bool)
	for rmIndex, rm := range r.ResourceManagers {
		if _, ok := rmNames[rm.Name()]; ok {
			errs = append(errs, fmt.Errorf(
				"resource manager at index %d has a duplicate name: %s", rmIndex, rm.Name()))
		}
		rmNames[rm.Name()] = true

		poolNames := make(map[string]bool)
		for ix, rp := range rm.Pools() {
			if _, ok := poolNames[rp.PoolName]; ok {
				errs = append(errs, fmt.Errorf(
					"%d resource pool has a duplicate name: %s", ix, rp.PoolName))
			} else {
				poolNames[rp.PoolName] = true
			}
		}
	}

	return errs
}
