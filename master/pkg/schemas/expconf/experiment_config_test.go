package expconf

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
)

type ExpConfigTestCase struct {
	Name     string
	Bytes    []byte
	Expected ExperimentConfig
}

func TestExpConfig(t *testing.T) {
	testCases := []ExpConfigTestCase{
		{
			Name: "Test experimnet config with nested hps",
			Bytes: []byte(`
    bind_mounts:
      - host_path: /asdf
        container_path: /asdf
        read_only: true
        propagation: "rprivate"
    checkpoint_policy: best
    checkpoint_storage:
      type: shared_fs
      host_path: /tmp
      storage_path: determined-cp
      propagation: rprivate
      container_path: /asdf
      checkpoint_path: /qwer
      tensorboard_path: /zxcv
      save_experiment_best: 0
      save_trial_best: 1
      save_trial_latest: 1
    data:
      any: thing
    data_layer:
      container_storage_path: null
      type: shared_fs
    debug: false
    description: pytorch-noop description
    entrypoint: long.module.path.model_def:NoopPytorchTrial
    environment:
      environment_variables: {}
      force_pull_image: false
      image:
        cpu: determinedai/environments:py-3.7-pytorch-1.7-lightning-1.2-tf-2.4-cpu-da845fc
        gpu: determinedai/environments:cuda-11.0-pytorch-1.7-lightning-1.2-tf-2.4-gpu-da845fc
      pod_spec: null
      ports:
        qwer: 1234
        asdf: 5678
      registry_auth:
        username: usr
        password: pwd
        auth: ath
        email: eml
        serveraddress: srvaddr
        identitytoken: idtkn
        registrytoken: rgtkn
      add_capabilities:
        - CAP_CHOWN
      drop_capabilities:
        - CAP_KILL
    hyperparameters:
      global_batch_size:
        type: const
        val: 32
      list_hparam:
        - 10
        - type: const
          val: asdf
        - type: int
          minval: 1
          maxval: 2
      dict_hparam:
        double_hparam:
          type: double
          minval: 1
          maxval: 10
        log_hparam:
          type: log
          minval: 1
          maxval: 10
          base: 1
      categorical_hparam:
        type: categorical
        vals: [1, 2, 3, 4]
      nested_hparam:
        hp1: 32
    labels: []
    max_restarts: 5
    min_validation_period:
      batches: 0
    name: pytorch-noop
    optimizations:
      aggregation_frequency: 1
      auto_tune_tensor_fusion: false
      average_aggregated_gradients: true
      average_training_metrics: false
      gradient_compression: false
      grad_updates_size_file: "/tmp/hi I am a size file"
      mixed_precision: O0
      tensor_fusion_cycle_time: 5
      tensor_fusion_threshold: 64
    perform_initial_validation: false
    profiling:
      enabled: true
      begin_on_batch: 0
      end_after_batch: 1
    records_per_epoch: 0
    reproducibility:
      experiment_seed: 1606239866
    resources:
      agent_label: 'big_al'
      devices:
        - host_path: "/dev/infiniband"
          container_path: "/dev/infiniband"
          mode: "rmw"
      slots_per_trial: 15
      weight: 1000
      max_slots: 900
      priority: 55
      resource_pool: 'asdf'
      native_parallel: false
    scheduling_unit: 100
    searcher:
      max_length:
        batches: 1000
      metric: loss
      name: single
      smaller_is_better: true
      source_checkpoint_uuid: null
      source_trial_id: null
    security:
      kerberos:
        config_file: xyz
    tensorboard_storage:
      type: shared_fs
      host_path: /tmp
      container_path: /asdf
      checkpoint_path: /qwer
      tensorboard_path: /zxcv
            `),
			Expected: ExperimentConfig{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			jByts, err := schemas.JSONFromYaml(tc.Bytes)
			assert.NilError(t, err)
			config, err := ParseAnyExperimentConfigYAML(jByts)
			assert.NilError(t, err)
			err = schemas.IsComplete(config)
			assert.NilError(t, err)
		})
	}
}
func TestBindMountsMerge(t *testing.T) {
	e1 := ExperimentConfig{
		RawBindMounts: BindMountsConfig{
			BindMount{
				RawHostPath:      "/host/e1",
				RawContainerPath: "/container/e1",
			},
		},
	}
	e2 := ExperimentConfig{
		RawBindMounts: BindMountsConfig{
			BindMount{
				RawHostPath:      "/host/e2",
				RawContainerPath: "/container/e2",
			},
		},
	}
	out := schemas.Merge(e1, e2).(ExperimentConfig)
	assert.Assert(t, len(out.RawBindMounts) == 2)
	assert.Assert(t, out.RawBindMounts[0].RawHostPath == "/host/e1")
	assert.Assert(t, out.RawBindMounts[1].RawHostPath == "/host/e2")
}

func TestName(t *testing.T) {
	config := ExperimentConfig{
		RawName: Name{
			RawString: ptrs.StringPtr("my_name"),
		},
	}

	// Test marshaling.
	bytes, err := json.Marshal(config)
	assert.NilError(t, err)

	rawObj := map[string]interface{}{}
	err = json.Unmarshal(bytes, &rawObj)
	assert.NilError(t, err)

	var expect interface{} = "my_name"
	assert.DeepEqual(t, rawObj["name"], expect)

	// Test unmarshaling.
	newConfig := ExperimentConfig{}
	err = json.Unmarshal(bytes, &newConfig)
	assert.NilError(t, err)

	assert.DeepEqual(t, newConfig.Name().String(), "my_name")
}
