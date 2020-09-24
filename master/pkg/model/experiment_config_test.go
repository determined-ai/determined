package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
)

const description = "provided"

func intP(x int) *int {
	return &x
}

func zeroizeRandomSeedsBeforeCompare(a *ExperimentConfig, b *ExperimentConfig) {
	// Because the default random seed is determined by the time, once in a great while these tests
	// will fail due to the experiment configs being created at different times.
	a.Reproducibility.ExperimentSeed = 0
	b.Reproducibility.ExperimentSeed = 0
}

func TestLabelsMap(t *testing.T) {
	actual := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "labels": {"l1": true, "l2": true}
}`), &actual))

	expected := DefaultExperimentConfig(nil)
	expected.Description = description
	expected.Labels = map[string]bool{
		"l1": true, "l2": true,
	}
	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestLabelsList(t *testing.T) {
	actual := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "labels": ["l1", "l2"]
}`), &actual))

	expected := DefaultExperimentConfig(nil)
	expected.Description = description
	expected.Labels = map[string]bool{
		"l1": true, "l2": true,
	}
	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestLabelsJoin(t *testing.T) {
	actual := DefaultExperimentConfig(nil)
	actual.Labels = map[string]bool{"l3": true}
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "labels": {"l1": true, "l2": true}
}`), &actual))

	expected := DefaultExperimentConfig(nil)
	expected.Description = description
	expected.Labels = map[string]bool{
		"l1": true, "l2": true, "l3": true,
	}
	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestRecordsPerEpochMissing(t *testing.T) {
	conf := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "searcher": {
    "name": "single",
    "metric": "loss",
    "smaller_is_better": false,
    "max_length": {
      "batches": 1000
    }
  },
  "min_checkpoint_period": {"epochs": 1}
}`), &conf))

	assert.ErrorContains(t, check.Validate(conf), "Must specify records_per_epoch")
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
	config1 := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal(json1, &config1))
	assert.DeepEqual(t, config1.Description, "test")

	// Check that user provided null string persists.
	config2 := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal(json2, &config2))
	assert.DeepEqual(t, config2.Description, "")

	// Check that unprovided description field will get filled.
	config3 := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal(json3, &config3))
	assert.Assert(t, strings.HasPrefix(config3.Description, "Experiment"))
}

func validGridSearchConfig() ExperimentConfig {
	config := DefaultExperimentConfig(nil)

	// This is unrelated to grid search but must be set to make the default config pass validation.
	config.CheckpointStorage.SharedFSConfig.HostPath = "/"
	config.Entrypoint = "model_def:TrialClass"
	config.MinCheckpointPeriod = NewLengthInBatches(1000)
	config.MinValidationPeriod = NewLengthInBatches(2000)

	// Construct a valid grid search config and hyperparameters.
	config.Searcher = SearcherConfig{
		GridConfig: &GridConfig{
			MaxLength: NewLengthInBatches(1000),
		},
	}
	config.Hyperparameters = map[string]Hyperparameter{
		GlobalBatchSize: {
			ConstHyperparameter: &ConstHyperparameter{
				Val: 64,
			},
		},
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
  "min_validation_period": {
	"batches": 1000
  },
  "min_checkpoint_period": {
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
  "scheduling_unit": 32,
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
	config1 := DefaultExperimentConfig(nil)
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
		MinCheckpointPeriod: NewLength(Batches, 2000),
		MinValidationPeriod: NewLength(Batches, 1000),
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
		SchedulingUnit: 32,
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

	json2, err := json.Marshal(config1)
	assert.NilError(t, err)
	assert.Assert(t, !cmp.Equal(json1, json2))

	// Check JSON marshaling round-trip support.
	config3 := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal(json2, &config3))
	assert.DeepEqual(t, config1, config3)
}

func TestMasterConfigImage(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		Image: &RuntimeItem{
			CPU: "test/cpu",
			GPU: "test/gpu",
		},
	}
	actual := DefaultExperimentConfig(masterDefault)
	actual.Description = description

	expected := DefaultExperimentConfig(nil)
	expected.Environment.Image.CPU = "test/cpu"
	expected.Environment.Image.GPU = "test/gpu"
	expected.Description = description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestOverrideMasterConfigImage(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		Image: &RuntimeItem{
			CPU: "test/cpu",
			GPU: "test/gpu",
		},
	}
	actual := DefaultExperimentConfig(masterDefault)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "environment": {"image":  "my-test-image"}
}`), &actual))

	expected := DefaultExperimentConfig(nil)
	myTestImage := "my-test-image"
	expected.Environment.Image.CPU = myTestImage
	expected.Environment.Image.GPU = myTestImage
	expected.Environment.Image.GPU = myTestImage
	expected.Environment.Image.GPU = myTestImage
	expected.Description = description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestMasterConfigPullPolicy(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		ForcePullImage: true,
	}
	actual := DefaultExperimentConfig(masterDefault)
	actual.Description = description

	expected := DefaultExperimentConfig(nil)
	expected.Environment.ForcePullImage = true
	expected.Description = description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestOverrideMasterConfigPullPolicy(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		ForcePullImage: true,
	}
	actual := DefaultExperimentConfig(masterDefault)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "environment": {"force_pull_image": false}
}`), &actual))

	expected := DefaultExperimentConfig(nil)
	expected.Description = description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestMasterConfigRegistryAuth(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		RegistryAuth: &types.AuthConfig{
			Username: "best-user",
			Password: "secret-password",
		},
	}
	actual := DefaultExperimentConfig(masterDefault)
	actual.Description = description

	expected := DefaultExperimentConfig(nil)
	expected.Environment.RegistryAuth = &types.AuthConfig{
		Username: "best-user",
		Password: "secret-password",
	}
	expected.Description = description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestOverrideMasterConfigRegistryAuth(t *testing.T) {
	masterDefault := &TaskContainerDefaultsConfig{
		RegistryAuth: &types.AuthConfig{
			Username: "best-user",
		},
	}
	actual := DefaultExperimentConfig(masterDefault)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "description": "provided",
  "environment": {"registry_auth": {"username": "worst-user"}}
}`), &actual))

	expected := DefaultExperimentConfig(nil)
	expected.Environment.RegistryAuth = &types.AuthConfig{
		Username: "worst-user",
	}
	expected.Description = description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestNoDebugConfig(t *testing.T) {
	actual := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "searcher": {
    "name": "single",
    "metric": "loss",
    "smaller_is_better": false,
    "max_length": {
      "batches": 1000
    }
  }
}`), &actual))

	marshaled, err := json.Marshal(actual)
	assert.NilError(t, err)

	var rawConfig map[string]interface{}
	assert.NilError(t, json.Unmarshal(marshaled, &rawConfig))

	_, ok := rawConfig["debug"]
	assert.Assert(t, !ok)
}

func TestFalseDebugConfig(t *testing.T) {
	actual := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "debug": false
}`), &actual))

	expected := DefaultExperimentConfig(nil)
	expected.Debug = &DebugConfig{}
	expected.Description = actual.Description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestTrueDebugConfig(t *testing.T) {
	actual := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "debug": true
}`), &actual))

	expected := DefaultExperimentConfig(nil)
	expected.Debug = &trueDebugConfig
	expected.Description = actual.Description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}

func TestFullDebugConfig(t *testing.T) {
	actual := DefaultExperimentConfig(nil)
	assert.NilError(t, json.Unmarshal([]byte(`{
  "debug": {
    "root_log_level": "INFO",
    "storage_log_level": "WARNING",
    "debug_all_workers": true,
    "horovod_verbose": false,
    "nccl_debug": "INFO",
    "nccl_debug_subsys": "^INIT",
    "resource_profile_period_sec": 0.5
  }
}`), &actual))

	debugConfigWarning := "WARNING"
	debugConfigInit := "^INIT"

	expected := DefaultExperimentConfig(nil)
	expected.Debug = &DebugConfig{
		RootLogLevel:             &debugConfigInfo,
		StorageLogLevel:          &debugConfigWarning,
		DebugAllWorkers:          true,
		HorovodVerbose:           false,
		NCCLDebug:                &debugConfigInfo,
		NCCLDebugSubsys:          &debugConfigInit,
		ResourceProfilePeriodSec: 0.5,
	}
	expected.Description = actual.Description

	zeroizeRandomSeedsBeforeCompare(&actual, &expected)
	assert.DeepEqual(t, actual, expected)
}
