package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
)

const description = "provided"

func intP(x int) *int {
	return &x
}

func TestLabelsMap(t *testing.T) {
	actual := DefaultExperimentConfig()
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "labels": {"l1": true, "l2": true}
}`), &actual))

	expected := DefaultExperimentConfig()
	expected.Description = description
	expected.Labels = map[string]bool{
		"l1": true, "l2": true,
	}
	assert.DeepEqual(t, actual, expected)
}

func TestLabelsList(t *testing.T) {
	actual := DefaultExperimentConfig()
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "labels": ["l1", "l2"]
}`), &actual))

	expected := DefaultExperimentConfig()
	expected.Description = description
	expected.Labels = map[string]bool{
		"l1": true, "l2": true,
	}
	assert.DeepEqual(t, actual, expected)
}

func TestLabelsJoin(t *testing.T) {
	actual := DefaultExperimentConfig()
	actual.Labels = map[string]bool{"l3": true}
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "labels": {"l1": true, "l2": true}
}`), &actual))

	expected := DefaultExperimentConfig()
	expected.Description = description
	expected.Labels = map[string]bool{
		"l1": true, "l2": true, "l3": true,
	}
	assert.DeepEqual(t, actual, expected)
}

func TestDefaultDescription(t *testing.T) {
	json1 := []byte(`{
  "description": "test"
}`)

	json2 := []byte(`{
  "description": ""
}`)

	json3 := []byte(`{
}`)

	// Check that user provided description persists.
	config1 := DefaultExperimentConfig()
	assert.NilError(t, json.Unmarshal(json1, &config1))
	assert.DeepEqual(t, config1.Description, "test")

	// Check that user provided null string persists.
	config2 := DefaultExperimentConfig()
	assert.NilError(t, json.Unmarshal(json2, &config2))
	assert.DeepEqual(t, config2.Description, "")

	// Check that unprovided description field will get filled.
	config3 := DefaultExperimentConfig()
	assert.NilError(t, json.Unmarshal(json3, &config3))
	assert.Assert(t, strings.HasPrefix(config3.Description, "Experiment"))
}

func validGridSearchConfig() ExperimentConfig {
	config := DefaultExperimentConfig()

	// This is unrelated to grid search but must be set to make the default config pass validation.
	config.CheckpointStorage.SharedFSConfig.HostPath = "/"
	config.Entrypoint = "model_def:TrialClass"
	config.CheckpointPeriod = NewLengthInBatches(1000)
	config.ValidationPeriod = NewLengthInBatches(2000)

	// Construct a valid grid search config and hyperparameters.
	config.Searcher = SearcherConfig{
		GridConfig: &GridConfig{
			MaxLength: NewLengthInBatches(1000),
		},
	}
	config.Hyperparameters = map[string]Hyperparameter{
		"const": {
			ConstHyperparameter: &ConstHyperparameter{
				Val: map[string]interface{}{
					"test": []interface{}{1., 2., 3.},
				},
			},
		},
		"cat": {
			CategoricalHyperparameter: &CategoricalHyperparameter{
				Vals: []interface{}{"a", 1.0},
			},
		},
		"int": {
			IntHyperparameter: &IntHyperparameter{
				Minval: 50,
				Maxval: 60,
				Count:  intP(5),
			},
		},
		"log": {
			LogHyperparameter: &LogHyperparameter{
				Minval: -6,
				Maxval: -2,
				Base:   10,
				Count:  intP(100),
			},
		},
	}
	return config
}

func validResourcesConfig() ExperimentConfig {
	config := validGridSearchConfig()
	config.Resources.SlotsPerTrial = 4
	return config
}

// TestGridValidation tests that invalid grid search configurations produce validation errors and
// valid ones don't.
func TestGridValidation(t *testing.T) {
	// Check that a config that should be good produces no errors.
	{
		config := validGridSearchConfig()
		assert.NilError(t, check.Validate(config))
	}

	// Check that too many trials triggers an error.
	{
		config := validGridSearchConfig()
		config.Hyperparameters["log"].LogHyperparameter.Count = intP(MaxAllowedTrials)
		config.Hyperparameters["int"].IntHyperparameter.Count = intP(2)
		assert.ErrorContains(t, check.Validate(config), "number of trials")
	}

	// Check that counts for int hyperparameters are clamped properly.
	{
		config := validGridSearchConfig()
		config.Hyperparameters["log"].LogHyperparameter.Count = intP(1)
		config.Hyperparameters["int"].IntHyperparameter.Count = intP(100000)
		assert.NilError(t, check.Validate(config))
	}

	// Check that a missing count triggers an error.
	{
		config := validGridSearchConfig()
		config.Hyperparameters["log"].LogHyperparameter.Count = nil
		assert.ErrorContains(t, check.Validate(config), "must specify counts for grid search: log")
	}
}

// TestResourcesValidation tests that invalid resources configurations produce validation errors and
// valid ones don't.
func TestResourcesValidation(t *testing.T) {
	// Check that a config that should be good produces no errors.
	{
		config := validResourcesConfig()
		assert.NilError(t, check.Validate(config))
	}
}

func TestExperiment(t *testing.T) {
	json1 := []byte(`{
  "description": "test",
  "data": {
    "foo": -1.2
  },
  "labels": {"l1": true, "l2": true},
  "checkpoint_storage": {
    "type": "s3",
    "save_experiment_best": 2,
    "bucket": "my bucket",
    "access_key": "my key",
    "secret_key": "my secret"
  },
  "tensorboard_storage": {
    "type": "s3",
    "bucket": "my bucket",
    "access_key": "my key",
    "secret_key": "my secret"
  },
  "min_validation_period": null,
  "validation_period": {
	"batches": 1000
  },
  "checkpoint_period": {
    "batches": 2000
  },
  "hyperparameters": {
    "const1": {
      "type": "const",
      "val": { "test": [1, 2, 3] }
    },
    "const2": 10,
    "log1": {
      "type": "log",
      "minval": -6,
      "maxval": -2,
      "base": 10
    },
    "cat1": {
      "type": "categorical",
      "vals": ["a", 1]
    },
    "int1": {
      "type": "int",
      "minval": 2,
      "maxval": 8,
      "count": 5
    }
  },
  "searcher": {
    "name": "single",
    "metric": "loss",
    "smaller_is_better": false,
    "max_length": {
		"batches": 1000
	}
  },
  "batches_per_step": 32,
  "bind_mounts": [
    { "host_path": "/host/path",
      "container_path": "/container/path",
      "read_only": false
    }
  ],
  "environment": {
    "image": "my_image"
  },
  "reproducibility": {
    "experiment_seed": 42
  },
  "security": {
	"kerberos": {
	  "config_file": "/etc/kerberos.conf"
	}
  }
}`)
	config1 := DefaultExperimentConfig()
	assert.NilError(t, json.Unmarshal(json1, &config1))

	accessKey := "my key"
	secretKey := "my secret"
	config2 := ExperimentConfig{
		Description: "test",
		Data:        map[string]interface{}{"foo": -1.2},
		Labels:      map[string]bool{"l1": true, "l2": true},
		CheckpointStorage: CheckpointStorageConfig{
			SaveExperimentBest: 2,
			SaveTrialBest:      1,
			SaveTrialLatest:    1,
			S3Config: &S3Config{
				Bucket:    "my bucket",
				AccessKey: &accessKey,
				SecretKey: &secretKey,
			},
		},
		DataLayer: DataLayerConfig{
			SharedFSConfig: &SharedFSDataLayerConfig{},
		},
		TensorboardStorage: &TensorboardStorageConfig{
			S3Config: &S3Config{
				Bucket:    "my bucket",
				AccessKey: &accessKey,
				SecretKey: &secretKey,
			},
		},
		MinCheckpointPeriod: nil,
		MinValidationPeriod: nil,
		CheckpointPeriod:    NewLength(Batches, 2000),
		ValidationPeriod:    NewLength(Batches, 1000),
		CheckpointPolicy:    "best",
		Hyperparameters: map[string]Hyperparameter{
			"const1": {
				ConstHyperparameter: &ConstHyperparameter{
					Val: map[string]interface{}{
						"test": []interface{}{1., 2., 3.},
					},
				},
			},
			"const2": {
				ConstHyperparameter: &ConstHyperparameter{
					Val: 10.0,
				},
			},
			"log1": {
				LogHyperparameter: &LogHyperparameter{
					Minval: -6,
					Maxval: -2,
					Base:   10,
				},
			},
			"cat1": {
				CategoricalHyperparameter: &CategoricalHyperparameter{
					Vals: []interface{}{"a", 1.0},
				},
			},
			"int1": {
				IntHyperparameter: &IntHyperparameter{
					Minval: 2,
					Maxval: 8,
					Count:  intP(5),
				},
			},
		},
		Searcher: SearcherConfig{
			Metric:          "loss",
			SmallerIsBetter: false,
			SingleConfig:    &SingleConfig{MaxLength: NewLengthInBatches(1000)},
		},
		Resources: ResourcesConfig{SlotsPerTrial: 1, Weight: 1},
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
		BatchesPerStep: 32,
		BindMounts: []BindMount{
			{
				HostPath:      "/host/path",
				ContainerPath: "/container/path",
				ReadOnly:      false,
				Propagation:   "rprivate",
			},
		},
		Environment: Environment{
			Image: RuntimeItem{
				CPU: "my_image",
				GPU: "my_image",
			},
		},
		Reproducibility: ReproducibilityConfig{
			ExperimentSeed: 42,
		},
		Security: &SecurityConfig{
			Kerberos: &KerberosConfig{
				ConfigFile: "/etc/kerberos.conf",
			},
		},
		MaxRestarts: 5,
	}

	// Unmarshal should give config2.
	assert.DeepEqual(t, config1, config2)

	// Can't round-trip to JSON.
	json2, err := json.Marshal(config1)
	assert.NilError(t, err)
	assert.Assert(t, !cmp.Equal(json1, json2))

	// Check JSON marshaling round-trip support.
	config3 := ExperimentConfig{}
	assert.NilError(t, json.Unmarshal(json2, &config3))
	assert.DeepEqual(t, config1, config3)
}
