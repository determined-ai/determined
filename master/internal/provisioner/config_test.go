package provisioner

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
)

func TestProvisionerConfigMissingFields(t *testing.T) {
	var config Config
	err := json.Unmarshal([]byte(`{}`), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.ErrorContains(t, err, "must configure aws or gcp cluster")
	expected := Config{
		MaxIdleAgentPeriod:     Duration(20 * time.Minute),
		MaxAgentStartingPeriod: Duration(20 * time.Minute),
		MaxInstances:           5,
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		AgentFluentImage:       "fluent/fluent-bit:1.6",
	}
	assert.DeepEqual(t, config, expected)
}

func TestUnmarshalProvisionerConfigMasterURL(t *testing.T) {
	configRaw := `{
"master_url": "http://test.master",
"provider": "aws",
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
	err = config.initMasterAddress()
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
		MaxIdleAgentPeriod:     Duration(30 * time.Second),
		MaxAgentStartingPeriod: Duration(30 * time.Second),
		MaxInstances:           5,
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
"provider": "aws",
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
	err = config.initMasterAddress()
	assert.NilError(t, err)
	awsConfig := defaultAWSClusterConfig
	awsConfig.Region = "test.region2"
	awsConfig.ImageID = "test.image2"
	awsConfig.SSHKeyName = "test-key2"
	unmarshaled := Config{
		MasterURL:              "http://test.master:8080",
		AWS:                    &awsConfig,
		AgentDockerImage:       "test_image",
		AgentFluentImage:       "fluent/fluent-bit:1.6",
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		MaxIdleAgentPeriod:     Duration(30 * time.Second),
		MaxAgentStartingPeriod: Duration(30 * time.Second),
		MaxInstances:           5,
	}
	assert.DeepEqual(t, config, unmarshaled)
}

func TestUnmarshalProvisionerConfigWithGCP(t *testing.T) {
	configRaw := `{
"master_url": "http://test.master",
"provider": "gcp",
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
	err = config.initMasterAddress()
	assert.NilError(t, err)
	expected := *DefaultGCPClusterConfig()
	expected.Project = "test_project2"
	expected.Zone = "test-zone2"
	expected.BootDiskSourceImage = "test-source_image2"
	unmarshaled := Config{
		MasterURL:              "http://test.master:8080",
		GCP:                    &expected,
		AgentDockerImage:       "test_image",
		AgentFluentImage:       "fluent/fluent-bit:1.6",
		AgentDockerRuntime:     "runc",
		AgentDockerNetwork:     "default",
		MaxIdleAgentPeriod:     Duration(20 * time.Minute),
		MaxAgentStartingPeriod: Duration(20 * time.Minute),
		MaxInstances:           5,
	}
	assert.DeepEqual(t, config, unmarshaled)
}
