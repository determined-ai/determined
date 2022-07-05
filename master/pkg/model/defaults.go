package model

import (
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// DefaultResourcesConfig returns the default resources configuration.
func DefaultResourcesConfig(taskContainerDefaults *TaskContainerDefaultsConfig) ResourcesConfig {
	config := ResourcesConfig{
		Weight:         1,
		NativeParallel: false,
	}
	if taskContainerDefaults == nil {
		return config
	}

	config.Devices = taskContainerDefaults.Devices
	return config
}

// DefaultEnvConfig returns the default environment configuration.
func DefaultEnvConfig(taskContainerDefaults *TaskContainerDefaultsConfig) Environment {
	config := Environment{
		Image: RuntimeItem{
			CPU:  expconf.CPUImage,
			CUDA: expconf.CUDAImage,
			ROCM: expconf.ROCMImage,
		},
	}

	if taskContainerDefaults == nil {
		return config
	}

	config.RegistryAuth = taskContainerDefaults.RegistryAuth
	config.ForcePullImage = taskContainerDefaults.ForcePullImage

	if taskContainerDefaults.Image != nil {
		config.Image = *taskContainerDefaults.Image
	}
	if taskContainerDefaults.EnvironmentVariables != nil {
		config.EnvironmentVariables = *taskContainerDefaults.EnvironmentVariables
	}

	config.AddCapabilities = taskContainerDefaults.AddCapabilities
	config.DropCapabilities = taskContainerDefaults.DropCapabilities
	return config
}
