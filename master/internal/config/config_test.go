//nolint:exhaustivestruct
package config

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/ghodss/yaml"
	"gotest.tools/assert"

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
		ResourceConfig: &ResourceConfig{
			ResourceManager: &ResourceManagerConfig{
				AgentRM: &AgentResourceManagerConfig{
					Scheduler: &SchedulerConfig{
						FairShare:     &FairShareSchedulerConfig{},
						FittingPolicy: "best",
					},
					DefaultComputeResourcePool: "default",
					DefaultAuxResourcePool:     "default",
				},
			},
			ResourcePools: []ResourcePoolConfig{
				{
					PoolName: "default",
					Provider: &provconfig.Config{
						AgentDockerRuntime:     "runc",
						AgentDockerNetwork:     "default",
						AgentDockerImage:       fmt.Sprintf("determinedai/determined-agent:%s", version.Version),
						AgentFluentImage:       aproto.FluentImage,
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
		ResourceConfig: &ResourceConfig{
			ResourceManager: &ResourceManagerConfig{
				AgentRM: &AgentResourceManagerConfig{
					Scheduler: &SchedulerConfig{
						FairShare:     &FairShareSchedulerConfig{},
						FittingPolicy: "best",
					},
					DefaultComputeResourcePool: "gpu-pool",
					DefaultAuxResourcePool:     "cpu-pool",
				},
			},
			ResourcePools: []ResourcePoolConfig{
				{
					PoolName: "cpu-pool",
					Provider: &provconfig.Config{
						AgentDockerRuntime:     "runc",
						AgentDockerNetwork:     "default",
						AgentDockerImage:       fmt.Sprintf("determinedai/determined-agent:%s", version.Version),
						AgentFluentImage:       aproto.FluentImage,
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
						AgentFluentImage:       aproto.FluentImage,
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

func TestPrintableConfig(t *testing.T) {
	s3Key := "my_access_key_secret"
	// nolint:gosec // These are not potential hardcoded credentials.
	s3Secret := "my_secret_key_secret"
	masterSecret := "my_master_secret"
	webuiSecret := "my_webui_secret"
	registryAuthSecret := "i_love_cellos"

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
`, s3Key, s3Secret, masterSecret, webuiSecret, registryAuthSecret)

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
			RegistryAuth: &types.AuthConfig{
				Username: "yo-yo-ma",
				Password: registryAuthSecret,
			},
			ShmSizeBytes: 4294967296,
			NetworkMode:  "bridge",
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

	// Ensure that the original was unmodified.
	assert.DeepEqual(t, unmarshaled, expected)
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
