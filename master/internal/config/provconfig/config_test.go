package provconfig

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"google.golang.org/api/compute/v1"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/version"
)

func TestProvisionerConfigMissingFields(t *testing.T) {
	var config Config
	err := json.Unmarshal([]byte(`{}`), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.ErrorContains(t, err, "must configure aws or gcp cluster")
	expected := Config{
		MaxIdleAgentPeriod:     model.Duration(20 * time.Minute),
		MaxAgentStartingPeriod: model.Duration(20 * time.Minute),
		MaxInstances:           5,
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		AgentDockerImage:       fmt.Sprintf("determinedai/determined-agent:%s", version.Version),
		AgentFluentImage:       aproto.FluentImage,
		AgentReconnectAttempts: aproto.AgentReconnectAttempts,
		AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
	}
	assert.DeepEqual(t, config, expected)
}

func TestUnmarshalProvisionerConfigMasterURL(t *testing.T) {
	configRaw := `{
"master_url": "http://test.master",
"type": "aws",
"agent_docker_image": "test_image",
"agent_fluent_image": "fluent_image",
"region": "test.region3",
"image_id": "test.image3",
"ssh_key_name": "test-key3",
"max_idle_agent_period": "30s",
"max_agent_starting_period": "30s"
}`
	config := Config{}
	err := json.Unmarshal([]byte(configRaw), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.NilError(t, err)
	err = config.InitMasterAddress()
	assert.NilError(t, err)
	awsConfig := defaultAWSClusterConfig
	awsConfig.Region = "test.region3"
	awsConfig.ImageID = "test.image3"
	awsConfig.SSHKeyName = "test-key3"
	unmarshaled := Config{
		MasterURL:              "http://test.master:8080",
		AgentDockerImage:       "test_image",
		AgentFluentImage:       "fluent_image",
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		AWS:                    &awsConfig,
		MaxIdleAgentPeriod:     model.Duration(30 * time.Second),
		MaxAgentStartingPeriod: model.Duration(30 * time.Second),
		MaxInstances:           5,
		AgentReconnectAttempts: aproto.AgentReconnectAttempts,
		AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
	}
	assert.DeepEqual(t, config, unmarshaled)
}

func TestUnmarshalProvisionerConfigStartupScript(t *testing.T) {
	configRaw := `
startup_script: |
                echo "hello world"
                sleep 5
`
	unmarshaled := DefaultConfig()
	unmarshaled.StartupScript = "echo \"hello world\"\nsleep 5\n"

	var config Config
	err := yaml.Unmarshal([]byte(configRaw), &config)
	assert.NilError(t, err)
	assert.DeepEqual(t, &config, unmarshaled)
}

func TestUnmarshalProvisionerConfigWithAWS(t *testing.T) {
	configRaw := `{
"master_url": "http://test.master",
"type": "aws",
"agent_docker_image": "test_image",
"region": "test.region2",
"image_id": "test.image2",
"ssh_key_name": "test-key2",
"max_idle_agent_period": "30s",
"max_agent_starting_period": "30s"
}`
	config := Config{}
	err := json.Unmarshal([]byte(configRaw), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.NilError(t, err)
	err = config.InitMasterAddress()
	assert.NilError(t, err)
	awsConfig := defaultAWSClusterConfig
	awsConfig.Region = "test.region2"
	awsConfig.ImageID = "test.image2"
	awsConfig.SSHKeyName = "test-key2"
	unmarshaled := Config{
		MasterURL:              "http://test.master:8080",
		AWS:                    &awsConfig,
		AgentDockerImage:       "test_image",
		AgentFluentImage:       aproto.FluentImage,
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		MaxIdleAgentPeriod:     model.Duration(30 * time.Second),
		MaxAgentStartingPeriod: model.Duration(30 * time.Second),
		MaxInstances:           5,
		AgentReconnectAttempts: aproto.AgentReconnectAttempts,
		AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
	}
	assert.DeepEqual(t, config, unmarshaled)
}

func TestUnmarshalProvisionerConfigWithGCP(t *testing.T) {
	configRaw := `{
"master_url": "http://test.master",
"type": "gcp",
"agent_docker_image": "test_image",
"project": "test_project2",
"zone": "test-zone2",
"boot_disk_source_image": "test-source_image2"
}`
	config := Config{}
	err := json.Unmarshal([]byte(configRaw), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.NilError(t, err)
	err = config.InitMasterAddress()
	assert.NilError(t, err)
	expected := *DefaultGCPClusterConfig()
	expected.Project = "test_project2"
	expected.Zone = "test-zone2"
	expected.BootDiskSourceImage = "test-source_image2"
	unmarshaled := Config{
		MasterURL:              "http://test.master:8080",
		GCP:                    &expected,
		AgentDockerImage:       "test_image",
		AgentFluentImage:       aproto.FluentImage,
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		MaxIdleAgentPeriod:     model.Duration(20 * time.Minute),
		MaxAgentStartingPeriod: model.Duration(20 * time.Minute),
		MaxInstances:           5,
		AgentReconnectAttempts: aproto.AgentReconnectAttempts,
		AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
	}
	assert.DeepEqual(t, config, unmarshaled)
}

func TestUnmarshalProvisionerConfigWithGCPBase(t *testing.T) {
	configRaw := `
master_url: http://test.master
agent_docker_image: test_image

type: gcp
base_config:
  disks:
    - mode: READ_ONLY
      boot: false
      initializeParams:
        sourceImage: projects/determined-ai/global/images/determined-agent
        diskSizeGb: "200"
        diskType: projects/determined-ai/zones/us-central1-a/diskTypes/pd-ssd
      autoDelete: true
project: test_project3
zone: test-zone3
boot_disk_source_image: test-source_image3
`
	unmarshaled := Config{}
	err := yaml.Unmarshal([]byte(configRaw), &unmarshaled, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	err = check.Validate(&unmarshaled)
	assert.NilError(t, err)
	err = unmarshaled.InitMasterAddress()
	assert.NilError(t, err)

	expectedGCP := *DefaultGCPClusterConfig()
	expectedGCP.BaseConfig = &compute.Instance{
		Disks: []*compute.AttachedDisk{
			{
				Mode: "READ_ONLY",
				Boot: false,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "projects/determined-ai/global/images/determined-agent",
					DiskSizeGb:  200,
					DiskType:    "projects/determined-ai/zones/us-central1-a/diskTypes/pd-ssd",
				},
				AutoDelete: true,
			},
		},
	}
	expectedGCP.Project = "test_project3"
	expectedGCP.Zone = "test-zone3"
	expectedGCP.BootDiskSourceImage = "test-source_image3"

	expected := Config{
		MasterURL:              "http://test.master:8080",
		GCP:                    &expectedGCP,
		AgentDockerImage:       "test_image",
		AgentFluentImage:       aproto.FluentImage,
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		MaxIdleAgentPeriod:     model.Duration(20 * time.Minute),
		MaxAgentStartingPeriod: model.Duration(20 * time.Minute),
		MaxInstances:           5,
		AgentReconnectAttempts: aproto.AgentReconnectAttempts,
		AgentReconnectBackoff:  aproto.AgentReconnectBackoffValue,
	}
	assert.DeepEqual(t, expected, unmarshaled)
}
