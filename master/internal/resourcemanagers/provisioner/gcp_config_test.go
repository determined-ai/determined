package provisioner

import (
	"encoding/json"
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestDefaultGCPClusterConfig(t *testing.T) {
	var config GCPClusterConfig
	err := json.Unmarshal([]byte(`
{
	"boot_disk_source_image": "test-source_image"
}`), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.NilError(t, err)
	expected := *DefaultGCPClusterConfig()
	expected.BootDiskSourceImage = "test-source_image"
	assert.DeepEqual(t, config, expected)
}

func TestUnmarshalGCPClusterConfig(t *testing.T) {
	type testcase struct {
		json        string
		unmarshaled GCPClusterConfig
	}
	tc := testcase{
		json: `
{
	"project": "test-project",
	"zone": "test-zone",
	"boot_disk_size": 100,
	"boot_disk_source_image": "test-source_image",
	"label_key": "test-label-key",
	"label_value": "test-label-value",
	"name_prefix": "test-name",
	"network_interface": {
		"network": "test-network",
		"subnetwork": "test-subnetwork",
		"external_ip": true
	},
	"network_tags": ["test-tag1", "test-tag2"],
	"service_account": {
		"email": "my-project-123@service.account.com",
		"scopes": ["a", "b"]
	},
	"instance_type": {
		"machine_type": "custom-1-1",
		"gpu_type": "nvidia-tesla-v100",
		"gpu_num": 2
	},
	"operation_timeout_period": "5m"
}`,
		unmarshaled: GCPClusterConfig{
			Project:             "test-project",
			Zone:                "test-zone",
			BootDiskSize:        100,
			BootDiskSourceImage: "test-source_image",
			LabelKey:            "test-label-key",
			LabelValue:          "test-label-value",
			NamePrefix:          "test-name",
			NetworkInterface: gceNetworkInterface{
				Network:    "test-network",
				Subnetwork: "test-subnetwork",
				ExternalIP: true,
			},
			NetworkTags: []string{"test-tag1", "test-tag2"},
			ServiceAccount: gceServiceAccount{
				Email:  "my-project-123@service.account.com",
				Scopes: []string{"a", "b"},
			},
			InstanceType: gceInstanceType{
				MachineType: "custom-1-1",
				GPUType:     "nvidia-tesla-v100",
				GPUNum:      2,
			},
			OperationTimeoutPeriod: model.Duration(5 * time.Minute),
		},
	}

	config := GCPClusterConfig{}
	err := json.Unmarshal([]byte(tc.json), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.NilError(t, err)
	assert.DeepEqual(t, config, tc.unmarshaled)
}

func TestGCPClusterConfigMissingFields(t *testing.T) {
	var config GCPClusterConfig
	err := json.Unmarshal([]byte(`{}`), &config)
	assert.NilError(t, err)
	err = check.Validate(&config)
	assert.NilError(t, err)
}

func TestGCEServiceAccount(t *testing.T) {
	type testcase struct {
		name        string
		raw         string
		unmarshaled gceServiceAccount
		errContains string
	}
	tcs := []testcase{
		{
			name: "unmarshal",
			raw: `{
"email": "22222-compute@developer.gserviceaccount.com",
"scopes": ["a", "b"]
}`,
			unmarshaled: gceServiceAccount{
				Email:  "22222-compute@developer.gserviceaccount.com",
				Scopes: []string{"a", "b"},
			},
			errContains: "",
		},
	}
	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			var unmarshaled gceServiceAccount
			err := json.Unmarshal([]byte(tc.raw), &unmarshaled)
			assert.NilError(t, err)
			assert.DeepEqual(t, unmarshaled, tc.unmarshaled)
			if err = check.Validate(&unmarshaled); tc.errContains == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}

func TestGCEInstanceType(t *testing.T) {
	type testcase struct {
		name        string
		raw         string
		unmarshaled gceInstanceType
		errContains string
	}
	tcs := []testcase{
		{
			name: "unmarshal",
			raw: `{
"machine_type": "n1-standard-1",
"gpu_type": "nvidia-tesla-v100",
"gpu_num": 8
}`,
			unmarshaled: gceInstanceType{
				MachineType: "n1-standard-1",
				GPUType:     "nvidia-tesla-v100",
				GPUNum:      8,
			},
		},
		{
			name: "invalid machine type",
			raw: `{
"machine_type": "bunny-1-1",
"gpu_type": "nvidia-tesla-v100",
"gpu_num": 8
}`,
			unmarshaled: gceInstanceType{
				MachineType: "bunny-1-1",
				GPUType:     "nvidia-tesla-v100",
				GPUNum:      8,
			},
			errContains: "machine type must be within",
		},
		{
			name: "invalid gpu type",
			raw: `{
"machine_type": "n1-standard-1",
"gpu_type": "nvidia-tesla-v999",
"gpu_num": 8
}`,
			unmarshaled: gceInstanceType{
				MachineType: "n1-standard-1",
				GPUType:     "nvidia-tesla-v999",
				GPUNum:      8,
			},
			errContains: "gpu type must be within",
		},
		{
			name: "invalid gpu num",
			raw: `{
"machine_type": "n1-standard-1",
"gpu_type": "nvidia-tesla-v100",
"gpu_num": 10
}`,
			unmarshaled: gceInstanceType{
				MachineType: "n1-standard-1",
				GPUType:     "nvidia-tesla-v100",
				GPUNum:      10,
			},
			errContains: "num must be within",
		},
	}
	for idx := range tcs {
		tc := tcs[idx]
		t.Run(tc.name, func(t *testing.T) {
			var unmarshaled gceInstanceType
			err := json.Unmarshal([]byte(tc.raw), &unmarshaled)
			assert.NilError(t, err)
			assert.DeepEqual(t, unmarshaled, tc.unmarshaled)
			if err = check.Validate(&unmarshaled); tc.errContains == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.errContains)
			}
		})
	}
}
