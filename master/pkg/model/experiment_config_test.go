package model

import (
	"encoding/json"
	"testing"

	"github.com/docker/docker/api/types"

	"github.com/determined-ai/determined/master/pkg/check"

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

// Helper function to setup and verify slurm option test cases.
func testEnvironmentSlurm(t *testing.T, slurm []string, expected ...string) {
	env := Environment{
		Slurm: slurm,
	}
	err := check.Validate(env)

	if len(expected) == 0 {
		assert.Equal(t, err, nil)
	} else {
		for _, msg := range expected {
			assert.ErrorContains(t, err, msg)
		}
	}
}

func TestValidateSlurmOptions(t *testing.T) {
	// No slurm args, not error
	testEnvironmentSlurm(t, []string{})
	// Forbidden -G option
	testEnvironmentSlurm(t, []string{"-G1"}, "slurm option -G is not configurable")
	// Forbidden --grpus=#
	testEnvironmentSlurm(t, []string{"--gpus=2"}, "slurm option --gpus= is not configurable")
	// OK --gpus-per-task=#
	testEnvironmentSlurm(t, []string{"--gpus-per-task=2"})
	// OK option containing letters of forbidden option (-n)
	testEnvironmentSlurm(t, []string{"--nice=3"})
	// Forbidden -n option intermixed with OK options
	testEnvironmentSlurm(t, []string{"--nice=7", "-n3", "-lname"},
		"slurm option -n is not configurable")
	// Multiple failures
	testEnvironmentSlurm(t, []string{"--nice=7", " -N2", "-Dmydir", "--partion=pname"},
		"slurm option -N is not configurable",
		"slurm option -D is not configurable")
}
