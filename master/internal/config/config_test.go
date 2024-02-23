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
			ResourceManagerV0DontUse: &ResourceManagerConfigV0{
				AgentRM: &AgentResourceManagerConfigV0{
					Scheduler: &SchedulerConfig{
						FairShare:     &FairShareSchedulerConfig{},
						FittingPolicy: "best",
					},
					DefaultComputeResourcePool: "default",
					DefaultAuxResourcePool:     "default",
				},
			},
			ResourcePoolsDontUse: []ResourcePoolConfig{
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
			ResourceManagerV0DontUse: &ResourceManagerConfigV0{
				AgentRM: &AgentResourceManagerConfigV0{
					Scheduler: &SchedulerConfig{
						FairShare:     &FairShareSchedulerConfig{},
						FittingPolicy: "best",
					},
					DefaultComputeResourcePool: "gpu-pool",
					DefaultAuxResourcePool:     "cpu-pool",
				},
			},
			ResourcePoolsDontUse: []ResourcePoolConfig{
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
			*unmarshaled.ResourceConfig.ResourceManagers[0].KubernetesRM.MaxSlotsPerPod)
		require.Equal(t, c.expected,
			*unmarshaled.TaskContainerDefaults.Kubernetes.MaxSlotsPerPod)
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

resource_managers:
  - type: agent
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
		Logging: model.LoggingConfig{
			DefaultLoggingConfig: &model.DefaultLoggingConfig{},
		},
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
			ResourcePoolsDontUse: []ResourcePoolConfig{
				{
					Provider:                 provConfig,
					AgentReconnectWait:       25000000000,
					MaxAuxContainersPerAgent: 100,
				},
			},
			ResourceManagers: ResourceManagersConfig{
				{
					AgentRM: &AgentResourceManagerConfigV1{
						ResourcePools: []ResourcePoolConfig{
							{
								Provider:                 provConfig,
								AgentReconnectWait:       25000000000,
								MaxAuxContainersPerAgent: 100,
							},
						},
						DefaultAuxResourcePool:     defaultRPName,
						DefaultComputeResourcePool: defaultRPName,
					},
				},
			},
		},
	}

	unmarshaled := Config{
		Logging: model.LoggingConfig{
			DefaultLoggingConfig: &model.DefaultLoggingConfig{},
		},
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

func TestDeprecations(t *testing.T) {
	config := Config{
		ResourceConfig: ResourceConfig{
			ResourceManagers: ResourceManagersConfig{
				{
					AgentRM: &AgentResourceManagerConfigV1{
						ResourcePools: []ResourcePoolConfig{
							{
								PoolName:             "a",
								AgentReattachEnabled: true,
							},
							{
								PoolName:             "b",
								AgentReattachEnabled: false,
							},
							{
								PoolName:             "c",
								AgentReattachEnabled: true,
							},
						},
					},
				},
			},
		},
	}

	actual := config.Deprecations()
	expected := []error{agentReattachDeprecateError("a"), agentReattachDeprecateError("c")}
	require.Equal(t, expected, actual)
}

func TestRMPreemptionStatus(t *testing.T) {
	test := func(t *testing.T, configRaw string, rpName string, expected bool) {
		unmarshaled := DefaultConfig()
		err := yaml.Unmarshal([]byte(configRaw), unmarshaled, yaml.DisallowUnknownFields)
		assert.NilError(t, unmarshaled.Resolve())
		assert.NilError(t, err)
		assert.DeepEqual(t, readRMPreemptionStatus(unmarshaled, rpName), expected)
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
