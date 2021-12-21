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
