package internal

import (
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/provisioner"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestUnmarshalConfigWithProvisioner(t *testing.T) {
	raw := `
log:
  level: info

db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

scheduler:
  fit: best

provisioner:
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
		Scheduler: &resourcemanagers.Config{Fit: "best"},
		Provisioner: &provisioner.Config{
			AgentDockerRuntime:     "runc",
			AgentDockerNetwork:     "default",
			AgentFluentImage:       "fluent/fluent-bit:1.6",
			MaxIdleAgentPeriod:     provisioner.Duration(30 * time.Second),
			MaxAgentStartingPeriod: provisioner.Duration(30 * time.Second),
			MaxInstances:           5,
		},
	}

	var unmarshaled Config
	err := yaml.Unmarshal([]byte(raw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	assert.DeepEqual(t, unmarshaled, expected)
}

func TestUnmarshalConfigWithoutProvisioner(t *testing.T) {
	raw := `
log:
  level: info

db:
  user: config_file_user
  password: password
  host: hostname
  port: "3000"

scheduler:
  fit: best
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
		Scheduler: &resourcemanagers.Config{Fit: "best"},
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

scheduler:
  fit: best

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
		Scheduler: &resourcemanagers.Config{Fit: "best"},
		CheckpointStorage: &expconf.CheckpointStorageConfig{
			S3Config: &expconf.S3Config{
				Bucket:    "my_bucket",
				AccessKey: ptrs.StringPtr("my_key"),
				SecretKey: ptrs.StringPtr("my_secret"),
			},
		},
	}

	var unmarshaled Config
	err := yaml.Unmarshal([]byte(raw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	assert.DeepEqual(t, unmarshaled, expected)
}
