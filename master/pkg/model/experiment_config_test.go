package model

import (
	"encoding/json"
	"testing"

	"github.com/docker/docker/api/types"

	"gotest.tools/assert"
)

func TestMasterConfigImage(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		Image: &RuntimeItem{
			CPU:  "test/cpu",
			CUDA: "test/gpu",
			ROCM: "test/rocm",
		},
	}
	actual := DefaultEnvConfig(masterDefault)

	expected := DefaultEnvConfig(nil)
	expected.Image.CPU = "test/cpu"
	expected.Image.CUDA = "test/gpu"
	expected.Image.ROCM = "test/rocm"

	assert.DeepEqual(t, actual, expected)
}

func TestOverrideMasterConfigImage(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		Image: &RuntimeItem{
			CPU:  "test/cpu",
			CUDA: "test/gpu",
			ROCM: "test/rocm",
		},
	}
	actual := DefaultEnvConfig(masterDefault)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "image":  "my-test-image"
}`), &actual))

	expected := DefaultEnvConfig(nil)
	myTestImage := "my-test-image"
	expected.Image = RuntimeItem{
		CPU:  myTestImage,
		CUDA: myTestImage,
		ROCM: myTestImage,
	}

	assert.DeepEqual(t, actual, expected)
}

func TestMasterConfigPullPolicy(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		ForcePullImage: true,
	}
	actual := DefaultEnvConfig(masterDefault)

	expected := DefaultEnvConfig(nil)
	expected.ForcePullImage = true

	assert.DeepEqual(t, actual, expected)
}

func TestOverrideMasterConfigPullPolicy(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		ForcePullImage: true,
	}
	actual := DefaultEnvConfig(masterDefault)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "force_pull_image": false
}`), &actual))

	expected := DefaultEnvConfig(nil)

	assert.DeepEqual(t, actual, expected)
}

func TestMasterConfigRegistryAuth(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		RegistryAuth: &types.AuthConfig{
			Username: "best-user",
			Password: "secret-password",
		},
	}
	actual := DefaultEnvConfig(masterDefault)

	expected := DefaultEnvConfig(nil)
	expected.RegistryAuth = &types.AuthConfig{
		Username: "best-user",
		Password: "secret-password",
	}

	assert.DeepEqual(t, actual, expected)
}

func TestOverrideMasterConfigRegistryAuth(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		RegistryAuth: &types.AuthConfig{
			Username: "best-user",
		},
	}
	actual := DefaultEnvConfig(masterDefault)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "registry_auth": {"username": "worst-user"}
}`), &actual))

	expected := DefaultEnvConfig(nil)
	expected.RegistryAuth = &types.AuthConfig{
		Username: "worst-user",
	}

	assert.DeepEqual(t, actual, expected)
}

func TestOverrideMasterEnvironmentVariables(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		EnvironmentVariables: &RuntimeItems{
			CPU: []string{"a=from_master", "b=from_master"},
		},
	}
	actual := DefaultEnvConfig(masterDefault)
	assert.NilError(t, json.Unmarshal([]byte(`{
    "environment_variables": ["a=from_exp", "c=from_master"]
}`), &actual))
	assert.DeepEqual(t, actual.EnvironmentVariables.CPU, []string{
		"a=from_master", "b=from_master",
		"a=from_exp", "c=from_master", // Exp config overwriters master config by being later.
	})
}

func TestDeviceConfig(t *testing.T) {
	// Devices can be strings or maps, and merging device lists is additive.
	var actual DevicesConfig

	assert.NilError(t, json.Unmarshal([]byte(`[
    {"host_path": "/not_asdf", "container_path": "/asdf"},
    {"host_path": "/zxcv", "container_path": "/zxcv"}
]`), &actual))

	assert.NilError(t, json.Unmarshal([]byte(`[
    {"host_path": "/asdf", "container_path": "/asdf"},
    "/qwer:/qwer"
]`), &actual))

	var expected DevicesConfig
	expected = append(expected, DeviceConfig{"/asdf", "/asdf", "mrw"})
	expected = append(expected, DeviceConfig{"/qwer", "/qwer", "mrw"})
	expected = append(expected, DeviceConfig{"/zxcv", "/zxcv", "mrw"})

	assert.DeepEqual(t, actual, expected)
}
