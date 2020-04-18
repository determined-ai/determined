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

// DefaultExperimentConfig returns a new default experiment config.
func DefaultExperimentConfig() ExperimentConfig {
	defaultDescription := fmt.Sprintf(
		"Experiment (%s)",
		petname.Generate(TaskNameGeneratorWords, TaskNameGeneratorSep))

	return ExperimentConfig{
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
			AsyncHalvingConfig: &AsyncHalvingConfig{
				SmallerIsBetter: true,
				Divisor:         4,
				TrainStragglers: true,
			},
			AdaptiveConfig: &AdaptiveConfig{
				SmallerIsBetter: true,
				Divisor:         4,
				TrainStragglers: true,
				Mode:            StandardMode,
				MaxRungs:        5,
			},
			AdaptiveSimpleConfig: &AdaptiveSimpleConfig{
				SmallerIsBetter: true,
				Divisor:         4,
				Mode:            StandardMode,
				MaxRungs:        5,
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
		BatchesPerStep: 100,
		Environment: Environment{
			Image: RuntimeItem{
				CPU: "determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.14-cpu-90bf50b",
				GPU: "determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.14-gpu-90bf50b",
			},
		},
		Reproducibility: ReproducibilityConfig{
			ExperimentSeed: uint32(time.Now().Unix()),
		},
		MaxRestarts: 5,
	}
}
