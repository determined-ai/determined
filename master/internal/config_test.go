package internal

import (
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/ghodss/yaml"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/logger"
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
						AgentFluentImage:       "fluent/fluent-bit:1.6",
						MaxIdleAgentPeriod:     provisioner.Duration(30 * time.Second),
						MaxAgentStartingPeriod: provisioner.Duration(30 * time.Second),
						MaxInstances:           5,
					},
					MaxCPUContainersPerAgent: 100,
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

func removeAllWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
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
		CheckpointStorage: CheckpointStorageConfig(removeAllWhitespace(`
{
  "access_key": "my_key",
  "bucket": "my_bucket",
  "save_experiment_best": 0,
  "save_trial_best": 0,
  "save_trial_latest": 0,
  "secret_key": "my_secret",
  "type":"s3"
}`)),
	}

	var unmarshaled Config
	err := yaml.Unmarshal([]byte(raw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	assert.DeepEqual(t, unmarshaled, expected)
}
