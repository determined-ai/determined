//nolint:exhaustruct
package config

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types/registry"
	"github.com/ghodss/yaml"
	"gotest.tools/assert"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/config/provconfig"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/config"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/version"
)

func TestDeprecations(t *testing.T) {
	c := Config{
		ResourceConfig: ResourceConfig{
			RootManagerInternal: &ResourceManagerConfig{
				KubernetesRM: &KubernetesResourceManagerConfig{},
			},
			RootPoolsInternal: []ResourcePoolConfig{
				{
					PoolName: "root",
				},
			},
			AdditionalResourceManagersInternal: []*ResourceManagerWithPoolsConfig{
				{
					ResourceManager: &ResourceManagerConfig{
						AgentRM: &AgentResourceManagerConfig{},
					},
					ResourcePools: []ResourcePoolConfig{
						{
							PoolName: "test",
						},
					},
				},
			},
		},
	}
	require.Empty(t, c.Deprecations())

	c.ResourceConfig.RootPoolsInternal[0].AgentReattachEnabled = true
	c.ResourceConfig.AdditionalResourceManagersInternal[0].ResourcePools[0].AgentReattachEnabled = true
	actual := c.Deprecations()
	require.Len(t, actual, 2)
	for i, n := range []string{"root", "test"} {
		require.ErrorContains(t, actual[i], "agent_reattach_enabled is set for resource pool "+n)
	}
}

func TestUnmarshalConfigWithAgentResourceManager(t *testing.T) {
	raw := `
log:
  level: info

db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: agent
  scheduler:
     fitting_policy: best

resource_pools:
  - pool_name: default
    provider:
      max_idle_agent_period: 30s
      max_agent_starting_period: 30s
    task_container_defaults:
      dtrain_network_interface: if0
integrations:
  pachyderm:
    address: foo
`
	defaultPriority := DefaultSchedulingPriority

	expected := Config{
		Log: logger.Config{
			Level: "info",
			Color: false,
		},
		DB: DBConfig{
			User:     "config_file_user",
			Password: "password",
			Host:     "hostname",
			Port:     "3000",
		},
		ResourceConfig: ResourceConfig{
			RootManagerInternal: &ResourceManagerConfig{
				AgentRM: &AgentResourceManagerConfig{
					Scheduler: &SchedulerConfig{
						Priority: &PrioritySchedulerConfig{
							DefaultPriority: &defaultPriority,
						},
						FittingPolicy: "best",
					},
					DefaultComputeResourcePool: "default",
					DefaultAuxResourcePool:     "default",
				},
			},
			RootPoolsInternal: []ResourcePoolConfig{
				{
					PoolName: "default",
					Provider: &provconfig.Config{
						AgentDockerRuntime:     "runc",
						AgentDockerNetwork:     "default",
						AgentDockerImage:       fmt.Sprintf("determinedai/determined-agent:%s", version.Version),
						AgentReconnectAttempts: aproto.AgentReconnectAttempts,
						AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
						MaxIdleAgentPeriod:     model.Duration(30 * time.Second),
						MaxAgentStartingPeriod: model.Duration(30 * time.Second),
						MaxInstances:           5,
					},
					MaxAuxContainersPerAgent: 100,
					TaskContainerDefaults: &model.TaskContainerDefaultsConfig{
						ShmSizeBytes:           4294967296,
						NetworkMode:            "bridge",
						DtrainNetworkInterface: "if0",
					},
					AgentReconnectWait: model.Duration(aproto.AgentReconnectWait),
				},
			},
		},
		Integrations: IntegrationsConfig{
			Pachyderm: PachydermConfig{
				Address: "foo",
			},
		},
	}

	unmarshaled := Config{}
	err := yaml.Unmarshal([]byte(raw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	assert.DeepEqual(t, unmarshaled, expected)
}

const (
	dbConfig = `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"`
)

func TestAgentRMSchedulerDeprecation(t *testing.T) {
	noScheduler := dbConfig + `

resource_manager:
  type: agent
`
	roundRobinScheduler := dbConfig + `
resource_manager:
  type: agent
  scheduler:
    type: round_robin
`

	fairShareScheduler := dbConfig + `
resource_manager:
  type: agent
  scheduler:
    type: fair_share
`

	type result struct {
		scheduler string
		allowed   bool
	}
	tests := map[string]result{
		noScheduler:         {PriorityScheduling, true},
		roundRobinScheduler: {RoundRobinScheduling, false},
		fairShareScheduler:  {FairShareScheduling, true},
	}

	for config, expected := range tests {
		unmarshaled := Config{}
		err := yaml.Unmarshal([]byte(config), &unmarshaled, yaml.DisallowUnknownFields)
		require.NoError(t, err)
		if expected.allowed {
			require.NoError(t, unmarshaled.Resolve())
		} else {
			require.Error(t, unmarshaled.Resolve())
		}
		rm := unmarshaled.ResourceManagers()
		require.Len(t, rm, 1)
		require.Equal(t, expected.scheduler, rm[0].ResourceManager.AgentRM.Scheduler.GetType())
	}
}

func TestUnmarshalConfigWithCPUGPUPools(t *testing.T) {
	raw := `
resource_manager:
  type: agent
  default_cpu_resource_pool: cpu-pool
  default_gpu_resource_pool: gpu-pool
  scheduler:
    type: fair_share
resource_pools:
  - pool_name: cpu-pool
    max_aux_containers_per_agent: 10
    provider:
      max_idle_agent_period: 10s
      max_agent_starting_period: 20s
  - pool_name: gpu-pool
    max_cpu_containers_per_agent: 0
    provider:
      max_idle_agent_period: 30s
      max_agent_starting_period: 40s
`
	expected := Config{
		ResourceConfig: ResourceConfig{
			RootManagerInternal: &ResourceManagerConfig{
				AgentRM: &AgentResourceManagerConfig{
					Scheduler: &SchedulerConfig{
						FairShare:     &FairShareSchedulerConfig{},
						FittingPolicy: "best",
					},
					DefaultComputeResourcePool: "gpu-pool",
					DefaultAuxResourcePool:     "cpu-pool",
				},
			},
			RootPoolsInternal: []ResourcePoolConfig{
				{
					PoolName: "cpu-pool",
					Provider: &provconfig.Config{
						AgentDockerRuntime:     "runc",
						AgentDockerNetwork:     "default",
						AgentDockerImage:       fmt.Sprintf("determinedai/determined-agent:%s", version.Version),
						AgentReconnectAttempts: aproto.AgentReconnectAttempts,
						AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
						MaxIdleAgentPeriod:     model.Duration(10 * time.Second),
						MaxAgentStartingPeriod: model.Duration(20 * time.Second),
						MaxInstances:           5,
					},
					MaxAuxContainersPerAgent: 10,
					MaxCPUContainersPerAgent: 0,
					AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
				},
				{
					PoolName: "gpu-pool",
					Provider: &provconfig.Config{
						AgentDockerRuntime:     "runc",
						AgentDockerNetwork:     "default",
						AgentDockerImage:       fmt.Sprintf("determinedai/determined-agent:%s", version.Version),
						AgentReconnectAttempts: aproto.AgentReconnectAttempts,
						AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
						MaxIdleAgentPeriod:     model.Duration(30 * time.Second),
						MaxAgentStartingPeriod: model.Duration(40 * time.Second),
						MaxInstances:           5,
					},
					MaxAuxContainersPerAgent: 0,
					MaxCPUContainersPerAgent: 0,
					AgentReconnectWait:       model.Duration(aproto.AgentReconnectWait),
				},
			},
		},
	}

	unmarshaled := Config{}
	err := yaml.Unmarshal([]byte(raw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	assert.DeepEqual(t, unmarshaled, expected)
}

func TestUnmarshalConfigWithoutResourceManager(t *testing.T) {
	raw := `
log:
  level: info

db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"
`
	expected := Config{
		Log: logger.Config{
			Level: "info",
			Color: false,
		},
		DB: DBConfig{
			User:     "config_file_user",
			Password: "password",
			Host:     "hostname",
			Port:     "3000",
		},
	}

	var unmarshaled Config
	err := yaml.Unmarshal([]byte(raw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	assert.DeepEqual(t, unmarshaled, expected)
}

func TestUnmarshalConfigWithExperiment(t *testing.T) {
	raw := `
log:
  level: info

db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

checkpoint_storage:
  type: s3
  access_key: my_key
  secret_key: my_secret
  bucket: my_bucket
`
	expected := Config{
		Log: logger.Config{
			Level: "info",
			Color: false,
		},
		DB: DBConfig{
			User:     "config_file_user",
			Password: "password",
			Host:     "hostname",
			Port:     "3000",
		},
		CheckpointStorage: expconf.CheckpointStorageConfig{
			RawS3Config: &expconf.S3Config{
				RawAccessKey: ptrs.Ptr("my_key"),
				RawBucket:    ptrs.Ptr("my_bucket"),
				RawSecretKey: ptrs.Ptr("my_secret"),
			},
		},
	}

	var unmarshaled Config
	err := yaml.Unmarshal([]byte(raw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	assert.DeepEqual(t, unmarshaled, expected)
}

func TestMaxSlotsPerPodConfig(t *testing.T) {
	{
		notK8s := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: agent
`
		var unmarshaled Config
		err := yaml.Unmarshal([]byte(notK8s), &unmarshaled, yaml.DisallowUnknownFields)
		require.NoError(t, err)
		require.NoError(t, unmarshaled.Resolve())
	}

	negativeRM := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
  max_slots_per_pod: -3
`
	negativeTask := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
task_container_defaults:
  kubernetes:
    max_slots_per_pod: -3
`
	for _, config := range []string{negativeRM, negativeTask} {
		var unmarshaled Config
		err := yaml.Unmarshal([]byte(config), &unmarshaled, yaml.DisallowUnknownFields)
		require.NoError(t, err)
		require.ErrorContains(t, unmarshaled.Resolve(), ">= 0")
	}

	noneOf := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
`

	both := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
  max_slots_per_pod: 10

task_container_defaults:
  kubernetes:
    max_slots_per_pod: 11
`
	for _, config := range []string{noneOf, both} {
		var unmarshaled Config
		err := yaml.Unmarshal([]byte(config), &unmarshaled, yaml.DisallowUnknownFields)
		require.NoError(t, err)
		require.ErrorContains(t, unmarshaled.Resolve(), "must provide exactly one")
	}

	rm := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
  max_slots_per_pod: 17
`
	rmZero := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
  max_slots_per_pod: 0
`

	taskDefaults := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
task_container_defaults:
  kubernetes:
    max_slots_per_pod: 17
`
	taskDefaultsZero := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
task_container_defaults:
  kubernetes:
    max_slots_per_pod: 0
`
	for _, c := range []struct {
		config   string
		expected int
	}{
		{rm, 17},
		{rmZero, 0},
		{taskDefaults, 17},
		{taskDefaultsZero, 0},
	} {
		var unmarshaled Config
		err := yaml.Unmarshal([]byte(c.config), &unmarshaled, yaml.DisallowUnknownFields)
		require.NoError(t, err)
		require.NoError(t, unmarshaled.Resolve())
		require.Equal(t, c.expected,
			*unmarshaled.RootManagerInternal.KubernetesRM.MaxSlotsPerPod)
		require.Equal(t, c.expected,
			*unmarshaled.TaskContainerDefaults.Kubernetes.MaxSlotsPerPod)
	}
}

func TestMaxSlotsPerPodConfigMultiRM(t *testing.T) {
	baseConfig := `
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

resource_manager:
  type: kubernetes
  max_slots_per_pod: 5

additional_resource_managers:
 - resource_manager:
     name: test
     type: kubernetes
`

	taskContainerDefaultsConfig := `
task_container_defaults:
  kubernetes:
    max_slots_per_pod: 0
`

	expectedMaxSlots := map[string]int{
		"default": 5,
		"test":    65,
	}

	testCases := map[string]struct {
		additionalMaxSlots string
		expectedError      string
	}{
		"valid config": {
			additionalMaxSlots: "     max_slots_per_pod: 65\n",
			expectedError:      "",
		},
		"negative max_slots": {
			additionalMaxSlots: "     max_slots_per_pod: -5\n",
			expectedError:      ">= 0",
		},
		"max_slots not defined": {
			additionalMaxSlots: "",
			expectedError:      "must provide resource_manager.max_slots_per_pod",
		},
		"global task max_slots also defined": {
			additionalMaxSlots: "     max_slots_per_pod: 65\n" + taskContainerDefaultsConfig,
			expectedError:      "",
		},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			var unmarshaled Config
			testConfig := baseConfig + test.additionalMaxSlots
			err := yaml.Unmarshal([]byte(testConfig), &unmarshaled, yaml.DisallowUnknownFields)
			require.NoError(t, err)
			if test.expectedError == "" { // no error is expected; this is a valid config
				require.NoError(t, unmarshaled.Resolve())
				actualRMs := unmarshaled.ResourceConfig.ResourceManagers()
				for _, r := range actualRMs {
					require.Equal(t, expectedMaxSlots[r.ResourceManager.Name()], *r.ResourceManager.KubernetesRM.MaxSlotsPerPod)
				}
			} else {
				require.Error(t, unmarshaled.Resolve(), test.expectedError)
			}
		})
	}
}

//nolint:gosec // These are not potential hardcoded credentials.
func TestPrintableConfig(t *testing.T) {
	s3Key := "my_access_key_secret"
	s3Secret := "my_secret_key_secret"
	masterSecret := "my_master_secret"
	webuiSecret := "my_webui_secret"
	registryAuthSecret := "i_love_cellos"
	startupScriptSecret := "my_startup_script_secret"
	containerStartupScriptSecret := "my_container_startup_secret"

	raw := fmt.Sprintf(`
db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

checkpoint_storage:
  type: s3
  access_key: %v
  secret_key: %v
  bucket: my_bucket

telemetry:
  enabled: true
  segment_master_key: %v
  segment_webui_key: %v

task_container_defaults:
  registry_auth:
    username: yo-yo-ma
    password: %v
    shm_size_bytes: 4294967296
    network_mode: bridge

resource_pools:
  - provider:
      type: gcp
      startup_script: %v
      container_startup_script: %v

additional_resource_managers:
  - resource_manager:
      type: agent
    resource_pools:
      - provider:
          type: gcp
          startup_script: %v
          container_startup_script: %v
`, s3Key, s3Secret, masterSecret, webuiSecret, registryAuthSecret, startupScriptSecret,
		containerStartupScriptSecret, startupScriptSecret, containerStartupScriptSecret)

	provConfig := provconfig.DefaultConfig()
	provConfig.StartupScript = startupScriptSecret
	provConfig.ContainerStartupScript = containerStartupScriptSecret
	provConfig.GCP = provconfig.DefaultGCPClusterConfig()
	expected := Config{
		Logging: model.LoggingConfig{DefaultLoggingConfig: &model.DefaultLoggingConfig{}},
		DB: DBConfig{
			User:     "config_file_user",
			Password: "password",
			Host:     "hostname",
			Port:     "3000",
		},
		CheckpointStorage: expconf.CheckpointStorageConfig{
			RawS3Config: &expconf.S3Config{
				RawAccessKey: ptrs.Ptr(s3Key),
				RawBucket:    ptrs.Ptr("my_bucket"),
				RawSecretKey: ptrs.Ptr(s3Secret),
			},
		},
		Telemetry: config.TelemetryConfig{
			Enabled:          true,
			SegmentMasterKey: masterSecret,
			SegmentWebUIKey:  webuiSecret,
		},
		TaskContainerDefaults: model.TaskContainerDefaultsConfig{
			RegistryAuth: &registry.AuthConfig{
				Username: "yo-yo-ma",
				Password: registryAuthSecret,
			},
			ShmSizeBytes: 4294967296,
			NetworkMode:  "bridge",
		},
		ResourceConfig: ResourceConfig{
			AdditionalResourceManagersInternal: []*ResourceManagerWithPoolsConfig{
				{
					ResourceManager: &ResourceManagerConfig{
						AgentRM: &AgentResourceManagerConfig{
							DefaultAuxResourcePool:     defaultResourcePoolName,
							DefaultComputeResourcePool: defaultResourcePoolName,
						},
					},
					ResourcePools: []ResourcePoolConfig{
						{
							Provider:                 provConfig,
							AgentReconnectWait:       25000000000,
							MaxAuxContainersPerAgent: 100,
						},
					},
				},
			},
			RootPoolsInternal: []ResourcePoolConfig{
				{
					Provider:                 provConfig,
					AgentReconnectWait:       25000000000,
					MaxAuxContainersPerAgent: 100,
				},
			},
		},
	}

	unmarshaled := Config{
		Logging: model.LoggingConfig{DefaultLoggingConfig: &model.DefaultLoggingConfig{}},
	}
	err := yaml.Unmarshal([]byte(raw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	assert.DeepEqual(t, unmarshaled, expected)

	printable, err := unmarshaled.Printable()
	assert.NilError(t, err)

	// No secrets are present.
	assert.Assert(t, !bytes.Contains(printable, []byte(s3Key)))
	assert.Assert(t, !bytes.Contains(printable, []byte(s3Secret)))
	assert.Assert(t, !bytes.Contains(printable, []byte(masterSecret)))
	assert.Assert(t, !bytes.Contains(printable, []byte(webuiSecret)))
	assert.Assert(t, !bytes.Contains(printable, []byte(registryAuthSecret)))
	assert.Assert(t, !bytes.Contains(printable, []byte(startupScriptSecret)))
	assert.Assert(t, !bytes.Contains(printable, []byte(containerStartupScriptSecret)))

	// Ensure that the original was unmodified.
	assert.DeepEqual(t, unmarshaled, expected)
}

func TestRMPreemptionStatus(t *testing.T) {
	test := func(t *testing.T, configRaw string, rpName string, expected bool) {
		unmarshaled := DefaultConfig()
		err := yaml.Unmarshal([]byte(configRaw), unmarshaled, yaml.DisallowUnknownFields)
		assert.NilError(t, unmarshaled.Resolve())
		assert.NilError(t, err)
		assert.DeepEqual(t, readRMPreemptionStatus(unmarshaled.ResourceManagers()[0], rpName), expected)
	}

	testCases := []struct {
		name              string
		configRaw         string
		rpName            string
		preemptionEnabled bool
	}{
		{
			name: "agent with scheduler.type=fair_share",
			configRaw: `
resource_manager:
  type: agent
  scheduler:
    type: fair_share
`,
			rpName:            "default",
			preemptionEnabled: true,
		},
		{
			name: "agent with scheduler.type=priority",
			configRaw: `
resource_manager:
  type: agent
  scheduler:
     type: priority
`,
			rpName:            "default",
			preemptionEnabled: false,
		},
		{
			name: "agent with scheduler.type=priority and preemption=true",
			configRaw: `
resource_manager:
  type: agent
  scheduler:
     type: priority
     preemption: true
`,
			rpName:            "default",
			preemptionEnabled: true,
		},
		{
			name: "agent with overridden preemption status by RP",
			configRaw: `
resource_manager:
  type: agent
  scheduler:
     preemption: true
     type: priority

resource_pools:
  - pool_name: default
    scheduler:
      preemption: false
      type: priority

`,
			rpName:            "default",
			preemptionEnabled: false,
		},
		{
			name: "agent with overridden preemption status by RP",
			configRaw: `
resource_manager:
  type: agent
  scheduler:
     preemption: false
     type: priority

resource_pools:
  - pool_name: default
    scheduler:
      preemption: true
      type: priority

`,
			rpName:            "default",
			preemptionEnabled: true,
		},
		{
			name: "agent with overridden preemption status by RP",
			configRaw: `
resource_manager:
  type: agent
  scheduler:
     preemption: true
     type: priority

resource_pools:
  - pool_name: default
    scheduler:
      preemption: false
      type: priority

`,
			rpName:            "non-existing",
			preemptionEnabled: true,
		},
		{
			name: "agent with overridden preemption status by RP",
			configRaw: `
resource_manager:
  type: agent
  scheduler:
     preemption: true
     type: priority

resource_pools:
  - pool_name: default
    scheduler:
      preemption: false
      type: priority
  - pool_name: preemtible
`,
			rpName:            "preemtible",
			preemptionEnabled: true,
		},
		{
			name: "k8 default",
			configRaw: `
resource_manager:
  type: kubernetes
  max_slots_per_pod: 2
`,
			rpName:            "",
			preemptionEnabled: false,
		},
		{
			name: "k8 with preemption plugin",
			configRaw: `
resource_manager:
  type: kubernetes
  default_scheduler: preemption
  max_slots_per_pod: 54
`,
			rpName:            "default",
			preemptionEnabled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test(t, tc.configRaw, tc.rpName, tc.preemptionEnabled)
		})
	}
}

func TestMultiRMPreemptionAndPriority(t *testing.T) {
	prio1 := 3
	prio2 := 30

	cfg := DefaultConfig()
	cfg.ResourceConfig = ResourceConfig{
		RootManagerInternal: &ResourceManagerConfig{AgentRM: &AgentResourceManagerConfig{
			Name: DefaultRMName, Scheduler: &SchedulerConfig{
				Priority: &PrioritySchedulerConfig{
					Preemption:      false,
					DefaultPriority: &prio1,
				},
			},
		}},
		RootPoolsInternal: []ResourcePoolConfig{{
			PoolName: "default123",
			Scheduler: &SchedulerConfig{Priority: &PrioritySchedulerConfig{
				Preemption:      true,
				DefaultPriority: &prio2,
			}},
		}},
		AdditionalResourceManagersInternal: []*ResourceManagerWithPoolsConfig{
			{ResourceManager: &ResourceManagerConfig{KubernetesRM: &KubernetesResourceManagerConfig{
				Name:             "test",
				DefaultScheduler: "not-preemption-scheduler",
			}}, ResourcePools: []ResourcePoolConfig{{
				PoolName: "default234",
				Scheduler: &SchedulerConfig{Priority: &PrioritySchedulerConfig{
					Preemption:      true,
					DefaultPriority: &prio1,
				}},
			}}},
			// nil preemption case
			{
				ResourceManager: &ResourceManagerConfig{KubernetesRM: &KubernetesResourceManagerConfig{
					Name: "nil-rm", DefaultScheduler: "not-preemption-scheduler",
				}},
				ResourcePools: []ResourcePoolConfig{{PoolName: "nil-rp"}},
			},
		},
	}

	SetMasterConfig(cfg)

	// 'default123' RP exists under 'default' RM, so the preemption will
	// be 'True' & priority to prio2, like the RP.
	status := ReadRMPreemptionStatus("default123")
	require.True(t, status)

	priority := ReadPriority("default123", model.CommandConfig{})
	require.Equal(t, prio2, priority)

	// 'test1' RP doesn't exist under any RM so the preemption and
	// priority will default.
	status = ReadRMPreemptionStatus("test1")
	require.False(t, status)

	priority = ReadPriority("test1", model.CommandConfig{})
	require.Equal(t, DefaultSchedulingPriority, priority)

	// 'default234' RP exists under 'test' RM, so the preemption
	// & priority will default to the RP's.
	status = ReadRMPreemptionStatus("default234")
	require.True(t, status)

	priority = ReadPriority("default234", model.CommandConfig{})
	require.Equal(t, prio1, priority)

	// 'nil-rp' RP exists under 'nil-rm' RM, so the preemption
	// & priority default to the RMs.
	status = ReadRMPreemptionStatus("nil-rp")
	require.False(t, status)

	priority = ReadPriority("nil-rp", model.CommandConfig{})
	require.Equal(t, KubernetesDefaultPriority, priority)
}

func TestPickVariation(t *testing.T) {
	tests := []struct {
		name        string
		variations  MediaAssetVariations
		mode        string
		orientation string
		expected    string
	}{
		{
			name: "Light Horizontal prioritized",
			variations: MediaAssetVariations{
				LightHorizontal: "light-horizontal",
				LightVeritical:  "light-vertical",
				DarkHorizontal:  "dark-horizontal",
				DarkVeritical:   "dark-vertical",
			},
			mode:        "",
			orientation: "",
			expected:    "light-horizontal",
		},
		{
			name: "Light Vertical when Light Horizontal is empty",
			variations: MediaAssetVariations{
				LightHorizontal: "",
				LightVeritical:  "light-vertical",
				DarkHorizontal:  "dark-horizontal",
				DarkVeritical:   "dark-vertical",
			},
			mode:        "",
			orientation: "vertical",
			expected:    "light-vertical",
		},
		{
			name: "Dark Horizontal when mode is dark",
			variations: MediaAssetVariations{
				LightHorizontal: "light-horizontal",
				LightVeritical:  "light-vertical",
				DarkHorizontal:  "dark-horizontal",
				DarkVeritical:   "dark-vertical",
			},
			mode:        "dark",
			orientation: "",
			expected:    "dark-horizontal",
		},
		{
			name: "Dark Vertical when mode is dark and orientation is vertical",
			variations: MediaAssetVariations{
				LightHorizontal: "light-horizontal",
				LightVeritical:  "light-vertical",
				DarkHorizontal:  "dark-horizontal",
				DarkVeritical:   "dark-vertical",
			},
			mode:        "dark",
			orientation: "vertical",
			expected:    "dark-vertical",
		},
		{
			name: "Fallback to Light Horizontal if no matches",
			variations: MediaAssetVariations{
				LightHorizontal: "light-horizontal",
				LightVeritical:  "",
				DarkHorizontal:  "",
				DarkVeritical:   "",
			},
			mode:        "dark",
			orientation: "vertical",
			expected:    "light-horizontal",
		},
		{
			name: "Empty variations fallback to empty string",
			variations: MediaAssetVariations{
				LightHorizontal: "",
				LightVeritical:  "",
				DarkHorizontal:  "",
				DarkVeritical:   "",
			},
			mode:        "",
			orientation: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.variations.PickVariation(tt.mode, tt.orientation)
			if result != tt.expected {
				t.Errorf("PickVariation(%v, %v) = %v; want %v", tt.mode, tt.orientation, result, tt.expected)
			}
		})
	}
}
