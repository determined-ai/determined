package internal

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
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
		DB: db.Config{
			User:     "config_file_user",
			Password: "password",
			Host:     "hostname",
			Port:     "3000",
		},
		ResourceConfig: &resourcemanagers.ResourceConfig{
			ResourceManager: &resourcemanagers.ResourceManagerConfig{
				AgentRM: &resourcemanagers.AgentResourceManagerConfig{
					Scheduler: &resourcemanagers.SchedulerConfig{
						FairShare:     &resourcemanagers.FairShareSchedulerConfig{},
						FittingPolicy: "best",
					},
					DefaultGPUResourcePool: "default",
					DefaultCPUResourcePool: "default",
				},
			},
			ResourcePools: []resourcemanagers.ResourcePoolConfig{
				{
					PoolName: "default",
					Provider: &provisioner.Config{
						AgentDockerRuntime:     "runc",
						AgentDockerNetwork:     "default",
						AgentDockerImage:       fmt.Sprintf("determinedai/determined-agent:%s", version.Version),
						AgentFluentImage:       "fluent/fluent-bit:1.6",
						MaxIdleAgentPeriod:     provisioner.Duration(30 * time.Second),
						MaxAgentStartingPeriod: provisioner.Duration(30 * time.Second),
						MaxInstances:           5,
					},
					MaxCPUContainersPerAgent: 100,
					TaskContainerDefaults: &model.TaskContainerDefaultsConfig{
						ShmSizeBytes:           4294967296,
						NetworkMode:            "bridge",
						DtrainNetworkInterface: "if0",
					},
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
		DB: db.Config{
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
		DB: db.Config{
			User:     "config_file_user",
			Password: "password",
			Host:     "hostname",
			Port:     "3000",
		},
		CheckpointStorage: expconf.CheckpointStorageConfig{
			RawS3Config: &expconf.S3Config{
				RawAccessKey: ptrs.StringPtr("my_key"),
				RawBucket:    ptrs.StringPtr("my_bucket"),
				RawSecretKey: ptrs.StringPtr("my_secret"),
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
`, s3Key, s3Secret, masterSecret, webuiSecret)

	expected := Config{
		Logging: model.LoggingConfig{
			DefaultLoggingConfig: &model.DefaultLoggingConfig{},
		},
		DB: db.Config{
			User:     "config_file_user",
			Password: "password",
			Host:     "hostname",
			Port:     "3000",
		},
		CheckpointStorage: expconf.CheckpointStorageConfig{
			RawS3Config: &expconf.S3Config{
				RawAccessKey: ptrs.StringPtr(s3Key),
				RawBucket:    ptrs.StringPtr("my_bucket"),
				RawSecretKey: ptrs.StringPtr(s3Secret),
			},
		},
		Telemetry: TelemetryConfig{
			Enabled:          true,
			SegmentMasterKey: masterSecret,
			SegmentWebUIKey:  webuiSecret,
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

	// Ensure that the original was unmodified.
	assert.DeepEqual(t, unmarshaled, expected)
}
