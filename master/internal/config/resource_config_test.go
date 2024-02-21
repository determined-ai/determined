//nolint:exhaustruct
package config

import (
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestGetAgentRMConfig(t *testing.T) {
	t.Run("no agent rm", func(t *testing.T) {
		noAgentRM := ResourceManagersConfig{
			{
				KubernetesRM: &KubernetesResourceManagerConfigV1{},
			},
			{
				KubernetesRM: &KubernetesResourceManagerConfigV1{},
			},
		}

		conf, ok := noAgentRM.GetAgentRMConfig()
		require.False(t, ok)
		require.Nil(t, conf)
	})

	t.Run("no agent rm", func(t *testing.T) {
		hasAgentRM := ResourceManagersConfig{
			{
				KubernetesRM: &KubernetesResourceManagerConfigV1{},
			},
			{
				AgentRM: &AgentResourceManagerConfigV1{},
			},
			{
				KubernetesRM: &KubernetesResourceManagerConfigV1{},
			},
		}

		conf, ok := hasAgentRM.GetAgentRMConfig()
		require.True(t, ok)
		require.Equal(t, hasAgentRM[1].AgentRM, conf)
	})
}

func TestResolveConfigErrors(t *testing.T) {
	cases := []struct {
		name                  string
		yaml                  string
		expectedError         error
		validationErrorString string
	}{
		{"both resource_manager and resource_managers", `
resource_manager:
  type: agent
resource_managers:
  - type: agent
`, errBothRMAndRMsGiven, ""},

		{"both resource_manager and resource_managers", `
resource_managers:
  - type: agent
    name: a
  - type: agent
    name: b
resource_pools:
  - name: test
`, errMoreThanOneRMAndRootPoolsGiven, ""},

		{"both resource_pools specified", `
resource_managers:
  - type: agent
    name: a
    resource_pools:
      - name: test
resource_pools:
  - name: test
`, errBothPoolsGiven, ""},

		{"multiple agent RM specified", `
resource_managers:
  - type: agent
    name: a
  - type: agent
    name: b
`, errMultipleAgentRMsGiven, ""},

		// TODO(RM-XXX) why is "Check Failed 2" errors.
		// I think it s because of check.Validate calling it twice somehow.
		{"dupe pools", `
resource_managers:
  - type: agent
    name: a
    resource_pools:
      - pool_name: a
      - pool_name: a
`, nil, "Check Failed! 2 errors found:\n\terror found at root.ResourceConfig: 1 resource pool " +
			"has a duplicate name: a\n\terror found at root: 1 resource pool has a duplicate name: a"},

		{"dupe rm names", `
resource_managers:
  - type: agent
    name: a
  - type: kubernetes
    max_slots_per_pod: 2
    name: a
`, nil, "Check Failed! 2 errors found:\n\terror found at root.ResourceConfig: " +
			"resource manager at index 1 has a duplicate name: a\n\terror found at root: " +
			"resource manager at index 1 has a duplicate name: a"},

		{"k8s name not specified", `
resource_managers:
  - type: agent
    name: a
  - type: kubernetes
    max_slots_per_pod: 2
`, nil, "Check Failed! 1 errors found:\n\terror found at root.ResourceConfig." +
			"ResourceManagers[1].KubernetesRM: name is required:  must be non-empty"},

		{"k8s rocm config", `
resource_managers:
  - type: kubernetes
    max_slots_per_pod: 2
    slot_type: rocm
`, nil, "Check Failed! 1 errors found:\n\terror found at root.ResourceConfig." +
			"ResourceManagers[0].KubernetesRM: rocm slot_type is not supported yet on k8s"},

		{"k8s negative cpu", `
resource_managers:
  - type: kubernetes
    max_slots_per_pod: 2
    slot_type: cpu
    slot_resource_requests:
        cpu: -10
`, nil, "Check Failed! 1 errors found:\n\terror found at root.ResourceConfig.ResourceManagers[0]." +
			"KubernetesRM: slot_resource_requests.cpu must be > 0: -10 is not greater than 0"},

		{"agent name not specified", `
resource_managers:
  - type: kubernetes
    max_slots_per_pod: 2
    name: a
  - type: agent
`, nil, "Check Failed! 1 errors found:\n\terror found at root.ResourceConfig." +
			"ResourceManagers[1].AgentRM: name is required:  must be non-empty"},
	}

	RegisterAuthZType("basic")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			unmarshaled := DefaultConfig()
			err := yaml.Unmarshal([]byte(c.yaml), &unmarshaled, yaml.DisallowUnknownFields)
			require.NoError(t, err)

			require.Equal(t, c.expectedError, unmarshaled.Resolve())
			if c.expectedError != nil {
				return
			}

			err = check.Validate(unmarshaled)
			require.Error(t, err, "expected validate to return error")
			require.Equal(t, c.validationErrorString, err.Error())
		})
	}
}

func TestResolveConfig(t *testing.T) {
	defaultRPConf := defaultRPConfig()
	defaultRPConf.PoolName = defaultRPName

	cases := []struct {
		name     string
		yaml     string
		expected Config
	}{
		{"no resource manager or pools specified", `{}`, Config{
			ResourceConfig: ResourceConfig{
				ResourceManagers: defaultRMsConfig(),
			},
		}},

		{"old resource manager specified with no pools / no scheduler", `
resource_manager:
  type: agent
`, Config{
			ResourceConfig: ResourceConfig{
				ResourceManagerV0DontUse: &ResourceManagerConfigV0{
					AgentRM: &AgentResourceManagerConfigV0{
						DefaultAuxResourcePool:     "default",
						DefaultComputeResourcePool: "default",
					},
				},
				ResourceManagers: defaultRMsConfig(),
			},
		}},

		{"old resource manager specified with no pools / scheduler given", `
resource_manager:
  type: agent
  scheduler:
    type: round_robin
`, Config{
			ResourceConfig: ResourceConfig{
				ResourceManagerV0DontUse: &ResourceManagerConfigV0{
					AgentRM: &AgentResourceManagerConfigV0{
						DefaultAuxResourcePool:     "default",
						DefaultComputeResourcePool: "default",
						Scheduler: &SchedulerConfig{
							RoundRobin:    &RoundRobinSchedulerConfig{},
							FittingPolicy: "best",
						},
					},
				},
				ResourceManagers: ResourceManagersConfig{
					{
						AgentRM: &AgentResourceManagerConfigV1{
							Name: defaultRMName,
							Scheduler: &SchedulerConfig{
								RoundRobin:    &RoundRobinSchedulerConfig{},
								FittingPolicy: "best",
							},
							ResourcePools:              []ResourcePoolConfig{defaultRPConf},
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
						},
					},
				},
			},
		}},

		{"old resource manager specified with pools given / scheduler given", `
resource_manager:
  type: agent
  scheduler:
    type: round_robin
resource_pools:
  - pool_name: test
  - pool_name: test2
`, Config{
			ResourceConfig: ResourceConfig{
				ResourceManagerV0DontUse: &ResourceManagerConfigV0{
					AgentRM: &AgentResourceManagerConfigV0{
						DefaultAuxResourcePool:     "default",
						DefaultComputeResourcePool: "default",
						Scheduler: &SchedulerConfig{
							RoundRobin:    &RoundRobinSchedulerConfig{},
							FittingPolicy: "best",
						},
					},
				},
				ResourcePoolsDontUse: []ResourcePoolConfig{
					{
						PoolName:                 "test",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
					{
						PoolName:                 "test2",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
				},
				ResourceManagers: ResourceManagersConfig{
					{
						AgentRM: &AgentResourceManagerConfigV1{
							Name: defaultRMName,
							Scheduler: &SchedulerConfig{
								RoundRobin:    &RoundRobinSchedulerConfig{},
								FittingPolicy: "best",
							},
							ResourcePools: []ResourcePoolConfig{
								{
									PoolName:                 "test",
									MaxAuxContainersPerAgent: 100,
									AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
								},
								{
									PoolName:                 "test2",
									MaxAuxContainersPerAgent: 100,
									AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
								},
							},
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
						},
					},
				},
			},
		}},

		{"new resource manager with old pools", `
resource_managers:
  - type: agent
resource_pools:
  - pool_name: test
  - pool_name: test2
`, Config{
			ResourceConfig: ResourceConfig{
				ResourcePoolsDontUse: []ResourcePoolConfig{
					{
						PoolName:                 "test",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
					{
						PoolName:                 "test2",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
				},
				ResourceManagers: ResourceManagersConfig{
					{
						AgentRM: &AgentResourceManagerConfigV1{
							Name:      defaultRMName,
							Scheduler: DefaultSchedulerConfig(),
							ResourcePools: []ResourcePoolConfig{
								{
									PoolName:                 "test",
									MaxAuxContainersPerAgent: 100,
									AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
								},
								{
									PoolName:                 "test2",
									MaxAuxContainersPerAgent: 100,
									AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
								},
							},
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
						},
					},
				},
			},
		}},

		{"no resource manager with pools given", `
resource_pools:
  - pool_name: test
  - pool_name: test2
`, Config{
			ResourceConfig: ResourceConfig{
				ResourceManagers: ResourceManagersConfig{
					{
						AgentRM: &AgentResourceManagerConfigV1{
							Name:                       defaultRMName,
							Scheduler:                  DefaultSchedulerConfig(),
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
							ResourcePools: []ResourcePoolConfig{
								{
									PoolName:                 "test",
									MaxAuxContainersPerAgent: 100,
									AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
								},
								{
									PoolName:                 "test2",
									MaxAuxContainersPerAgent: 100,
									AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
								},
							},
						},
					},
				},
				ResourcePoolsDontUse: []ResourcePoolConfig{
					{
						PoolName:                 "test",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
					{
						PoolName:                 "test2",
						MaxAuxContainersPerAgent: 100,
						AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
					},
				},
			},
		}},

		{"metadata for multiple resource managers", `
resource_managers:
  - type: agent
    name: dockeragents
    metadata:
      region: "nw"
      nest:
        into: "c"
  - type: kubernetes
    metadata:
      test: "y"
    name: k8s
    max_slots_per_pod: 65
`, Config{
			ResourceConfig: ResourceConfig{
				ResourceManagers: ResourceManagersConfig{
					{
						AgentRM: &AgentResourceManagerConfigV1{
							Name:                       "dockeragents",
							Scheduler:                  DefaultSchedulerConfig(),
							ResourcePools:              []ResourcePoolConfig{defaultRPConf},
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
							Metadata: map[string]any{
								"region": "nw",
								"nest":   map[string]any{"into": "c"},
							},
						},
					},
					{
						KubernetesRM: &KubernetesResourceManagerConfigV1{
							Name:                       "k8s",
							ResourcePools:              []ResourcePoolConfig{defaultRPConf},
							MaxSlotsPerPod:             ptrs.Ptr(65),
							SlotType:                   "cuda",
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
							Metadata: map[string]any{
								"test": "y",
							},
						},
					},
				},
			},
		}},

		{"multiple resource manager pools get defaulted", `
resource_managers:
  - type: agent
    name: dockeragents
  - type: kubernetes
    name: k8s
    max_slots_per_pod: 65
`, Config{
			ResourceConfig: ResourceConfig{
				ResourceManagers: ResourceManagersConfig{
					{
						AgentRM: &AgentResourceManagerConfigV1{
							Name:                       "dockeragents",
							Scheduler:                  DefaultSchedulerConfig(),
							ResourcePools:              []ResourcePoolConfig{defaultRPConf},
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
						},
					},
					{
						KubernetesRM: &KubernetesResourceManagerConfigV1{
							Name:                       "k8s",
							ResourcePools:              []ResourcePoolConfig{defaultRPConf},
							MaxSlotsPerPod:             ptrs.Ptr(65),
							SlotType:                   "cuda",
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
						},
					},
				},
			},
		}},

		{"new resource manager with new pools", `
resource_managers:
  - type: agent
    resource_pools:
      - pool_name: test
      - pool_name: test2
`, Config{
			ResourceConfig: ResourceConfig{
				ResourceManagers: ResourceManagersConfig{
					{
						AgentRM: &AgentResourceManagerConfigV1{
							Name:      defaultRMName,
							Scheduler: DefaultSchedulerConfig(),
							ResourcePools: []ResourcePoolConfig{
								{
									PoolName:                 "test",
									MaxAuxContainersPerAgent: 100,
									AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
								},
								{
									PoolName:                 "test2",
									MaxAuxContainersPerAgent: 100,
									AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
								},
							},
							DefaultAuxResourcePool:     defaultRPName,
							DefaultComputeResourcePool: defaultRPName,
						},
					},
				},
			},
		}},
	}

	RegisterAuthZType("basic")
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			unmarshaled := DefaultConfig()
			err := yaml.Unmarshal([]byte(c.yaml), &unmarshaled, yaml.DisallowUnknownFields)
			require.NoError(t, err)
			require.NoError(t, unmarshaled.Resolve())
			require.NoError(t, check.Validate(unmarshaled))

			require.Equal(t, c.expected.ResourceConfig, unmarshaled.ResourceConfig)
			require.Equal(t, c.expected.ResourcePoolsDontUse, unmarshaled.ResourcePoolsDontUse)
		})
	}
}
