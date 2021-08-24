package model

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// Configuration constants for task name generator.
const (
	TaskNameGeneratorWords = 3
	TaskNameGeneratorSep   = "-"
)

const (
	// BestCheckpointPolicy will checkpoint trials after validation if the validation is the best
	// validation so far.
	BestCheckpointPolicy = "best"
	// AllCheckpointPolicy will always checkpoint trials after validation.
	AllCheckpointPolicy = "all"
	// NoneCheckpointPolicy will not checkpoint trials after validations.
	NoneCheckpointPolicy = "none"
)

// Default task environment docker image names.
const (
	CPUImage = "determinedai/environments:py-3.7-pytorch-1.9-lightning-1.3-tf-2.4-cpu-a173dcd"
	GPUImage = "determinedai/environments:cuda-11.1-pytorch-1.9-lightning-1.3-tf-2.4-gpu-a173dcd"
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
			CPU: CPUImage,
			GPU: GPUImage,
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

	config.AddCapabilities = taskContainerDefaults.AddCapabilities
	config.DropCapabilities = taskContainerDefaults.DropCapabilities
	return config
}

// DefaultExperimentConfig returns a new default experiment config.
func DefaultExperimentConfig(taskContainerDefaults *TaskContainerDefaultsConfig) ExperimentConfig {
	conf := schemas.WithDefaults(expconf.ExperimentConfig{}).(expconf.ExperimentConfig)

	defaultConfig := ExperimentConfig{
		Name: conf.RawName.String(),
		CheckpointStorage: CheckpointStorageConfig{
			SaveExperimentBest: 0,
			SaveTrialBest:      1,
			SaveTrialLatest:    1,
			SharedFSConfig:     &SharedFSConfig{},
		},
		CheckpointPolicy: BestCheckpointPolicy,
		DataLayer: DataLayerConfig{
			SharedFSConfig: &SharedFSDataLayerConfig{},
		},
		Hyperparameters: make(map[string]Hyperparameter),
		Searcher: SearcherConfig{
			SmallerIsBetter: true,
			RandomConfig: &RandomConfig{
				MaxConcurrentTrials: 0,
			},
			GridConfig: &GridConfig{
				MaxConcurrentTrials: 0,
			},
			AsyncHalvingConfig: &AsyncHalvingConfig{
				SmallerIsBetter:     true,
				Divisor:             4,
				MaxConcurrentTrials: 0,
				StopOnce:            false,
			},
			AdaptiveASHAConfig: &AdaptiveASHAConfig{
				SmallerIsBetter:     true,
				Divisor:             4,
				Mode:                StandardMode,
				MaxRungs:            5,
				MaxConcurrentTrials: 0,
				StopOnce:            false,
			},
			PBTConfig: &PBTConfig{
				SmallerIsBetter: true,
			},
		},
		Resources: DefaultResourcesConfig(taskContainerDefaults),
		Optimizations: OptimizationsConfig{
			AggregationFrequency:       1,
			AverageAggregatedGradients: true,
			AverageTrainingMetrics:     false,
			GradientCompression:        false,
			MixedPrecision:             "O0",
			TensorFusionThreshold:      64,
			TensorFusionCycleTime:      5,
			AutoTuneTensorFusion:       false,
		},
		RecordsPerEpoch: 0,
		SchedulingUnit:  100,
		Environment:     DefaultEnvConfig(taskContainerDefaults),
		Reproducibility: ReproducibilityConfig{
			ExperimentSeed: uint32(time.Now().Unix()),
		},
		MaxRestarts: 5,
		Profiling: ProfilingConfig{
			Enabled: false,
		},
	}

	if taskContainerDefaults == nil {
		return defaultConfig
	}

	defaultConfig.BindMounts = taskContainerDefaults.BindMounts

	return defaultConfig
}
