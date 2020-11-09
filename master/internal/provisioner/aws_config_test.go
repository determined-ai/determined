package provisioner

import (
	"encoding/json"
	"testing"

	"github.com/ghodss/yaml"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
)

func TestDefaultAWSClusterConfig(t *testing.T) {
	var config AWSClusterConfig
	err := json.Unmarshal([]byte(`
{
	"region": "test.region",
	"image_id": "test.image",
	"ssh_key_name": "test-key"
}`), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.NilError(t, err)
	expected := defaultAWSClusterConfig
	expected.Region = "test.region"
	expected.ImageID = "test.image"
	expected.SSHKeyName = "test-key"
	assert.DeepEqual(t, config, expected)
}

func TestUnmarshalAWSClusterConfig(t *testing.T) {
	type testcase struct {
		json        string
		Unmarshaled AWSClusterConfig
	}
	tc := testcase{`
{
	"region": "test.region",
	"image_id": "test.image",
	"instance_name": "test.instance_name",
	"ssh_key_name": "test.key",
	"network_interface": {
		"public_ip": false,
		"subnet_id": "test.subnet",
		"security_group_id": "test.security"
	},
	"tag_key": "dai",
	"tag_value": "agent",
	"root_volume_size": 120,
	"instance_type": "p2.xlarge",
	"iam_instance_profile_arn": "test_instance_profile",
	"custom_tags": [
		{
			"key": "key1",
			"value": "value1",
		}
	]
}`,
		AWSClusterConfig{
			Region:       "test.region",
			ImageID:      "test.image",
			InstanceName: "test.instance_name",
			SSHKeyName:   "test.key",
			NetworkInterface: ec2NetworkInterface{
				PublicIP:        false,
				SubnetID:        "test.subnet",
				SecurityGroupID: "test.security",
			},
			TagKey:                "dai",
			TagValue:              "agent",
			RootVolumeSize:        120,
			InstanceType:          "p2.xlarge",
			IamInstanceProfileArn: "test_instance_profile",
			CustomTags: []*ec2Tag{
				{
					Key: "key1",
					Value: "value1",
				},
			},
		},
	}

	config := AWSClusterConfig{}
	err := yaml.Unmarshal([]byte(tc.json), &config, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.NilError(t, err)
	assert.DeepEqual(t, config, tc.Unmarshaled)
}

func TestAWSClusterConfigMissingFields(t *testing.T) {
	var config AWSClusterConfig
	err := yaml.Unmarshal([]byte(`{}`), &config, yaml.DisallowUnknownFields)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.ErrorContains(t, err, "non-empty")
}
