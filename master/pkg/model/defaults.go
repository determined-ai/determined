package model

import (
	"fmt"
	"time"

	petname "github.com/dustinkirkland/golang-petname"
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
	defaultCPUImage = "determinedai/environments:py-3.7-pytorch-1.7-tf-1.15-cpu-1c26118"
	defaultGPUImage = "determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-1c26118"
)

// DefaultExperimentConfig returns a new default experiment config.
func DefaultExperimentConfig(taskContainerDefaults *TaskContainerDefaultsConfig) ExperimentConfig {
	defaultDescription := fmt.Sprintf(
		"Experiment (%s)",
		petname.Generate(TaskNameGeneratorWords, TaskNameGeneratorSep))

	defaultConfig := ExperimentConfig{
		Description: defaultDescription,
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
		Resources: ResourcesConfig{
			SlotsPerTrial:  1,
			Weight:         1,
			NativeParallel: false,
		},
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
		Environment: Environment{
			Image: RuntimeItem{
				CPU: defaultCPUImage,
				GPU: defaultGPUImage,
			},
		},
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

	defaultConfig.Environment.RegistryAuth = taskContainerDefaults.RegistryAuth
	defaultConfig.Environment.ForcePullImage = taskContainerDefaults.ForcePullImage

	if taskContainerDefaults.Image != nil {
		defaultConfig.Environment.Image = *taskContainerDefaults.Image
	}

	defaultConfig.Resources.Devices = taskContainerDefaults.Devices
	defaultConfig.Environment.AddCapabilities = taskContainerDefaults.AddCapabilities
	defaultConfig.Environment.DropCapabilities = taskContainerDefaults.DropCapabilities

	return defaultConfig
}
