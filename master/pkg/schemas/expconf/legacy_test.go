//nolint:exhaustivestruct
package expconf

import (
	"testing"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
)

type LegacyConfigTestCase struct {
	Name     string
	Bytes    []byte
	Expected LegacyConfig
}

func TestLegacyConfig(t *testing.T) {
	testCases := []LegacyConfigTestCase{
		// Test case with a 0.12.13 experiment config (before the remove steps project).
		{
			Name: "0.12.13 config with steps in config",
			Bytes: []byte(`
                batches_per_step: 1
                checkpoint_policy: best
                checkpoint_storage:
                  host_path: /tmp
                  save_experiment_best: 10
                  save_trial_best: 10
                  save_trial_latest: 10
                  storage_path: determined-cp
                  container_path: qwer
                  checkpoint_path: asdf
                  tensorboard_path: zxcv
                  propagation: rprivate
                  type: shared_fs
                data_layer:
                  container_storage_path: null
                  type: shared_fs
                debug: false
                description: noop_trial
                entrypoint: model_def:NoopTrial
                environment:
                  environment_variables:
                  - "HOME=/where/the/heart/is"
                  force_pull_image: false
                  image:
                    cpu: determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.15-cpu-aaa3750
                    gpu: determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.15-gpu-aaa3750
                  ports: null
                hyperparameters:
                  global_batch_size:
                    type: const
                    val: 64
                internal: null
                labels:
                - 0.12.13
                max_restarts: 0
                min_checkpoint_period: null
                min_validation_period: null
                optimizations:
                  aggregation_frequency: 1
                  auto_tune_tensor_fusion: false
                  average_aggregated_gradients: true
                  average_training_metrics: false
                  gradient_compression: false
                  mixed_precision: O0
                  tensor_fusion_cycle_time: 5
                  tensor_fusion_threshold: 64
                reproducibility:
                  experiment_seed: 1621971794
                resources:
                  agent_label: ''
                  native_parallel: false
                  slots_per_trial: 1
                  weight: 1
                searcher:
                  max_steps: 1000
                  max_trials: 1
                  metric: error
                  name: random
                  smaller_is_better: false
                  source_checkpoint_uuid: null
                  source_trial_id: null
            `),
			Expected: LegacyConfig{
				CheckpointStorage: CheckpointStorageConfig{
					RawSharedFSConfig: &SharedFSConfig{
						RawHostPath:        ptrs.Ptr("/tmp"),
						RawContainerPath:   ptrs.Ptr("qwer"),
						RawCheckpointPath:  ptrs.Ptr("asdf"),
						RawTensorboardPath: ptrs.Ptr("zxcv"),
						RawStoragePath:     ptrs.Ptr("determined-cp"),
						RawPropagation:     ptrs.Ptr("rprivate"),
					},
					RawSaveExperimentBest: ptrs.Ptr(10),
					RawSaveTrialBest:      ptrs.Ptr(10),
					RawSaveTrialLatest:    ptrs.Ptr(10),
				},
				BindMounts: BindMountsConfig{},
				Environment: EnvironmentConfig{
					RawEnvironmentVariables: &EnvironmentVariablesMap{
						RawCPU:  []string{"HOME=/where/the/heart/is"},
						RawCUDA: []string{"HOME=/where/the/heart/is"},
						RawROCM: []string{"HOME=/where/the/heart/is"},
					},
					RawImage: &EnvironmentImageMap{
						RawCPU:  ptrs.Ptr("determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.15-cpu-aaa3750"),
						RawCUDA: ptrs.Ptr("determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.15-gpu-aaa3750"),
						RawROCM: ptrs.Ptr("determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-2b7e2a1"),
					},
					RawPorts:            map[string]int{},
					RawProxyPorts:       &ProxyPortsConfigV0{},
					RawForcePullImage:   ptrs.Ptr(false),
					RawAddCapabilities:  []string{},
					RawDropCapabilities: []string{},
				},
				Hyperparameters: Hyperparameters{
					"global_batch_size": {
						RawConstHyperparameter: &ConstHyperparameterV0{RawVal: float64(64)},
					},
				},
				Searcher: LegacySearcher{
					Name:            "random",
					SmallerIsBetter: false,
					Metric:          "error",
				},
			},
		},
		// Test case with a 0.14.3 experiment config (before removing adaptive, adaptive_simple,
		// and sync_halving).
		{
			Name: "0.14.3 config with EOL searcher",
			Bytes: []byte(`
                checkpoint_policy: best
                checkpoint_storage:
                  host_path: /tmp
                  save_experiment_best: 10
                  save_trial_best: 10
                  save_trial_latest: 10
                  storage_path: determined-cp
                  type: shared_fs
                data_layer:
                  container_storage_path: null
                  type: shared_fs
                debug: false
                description: noop pytorch
                entrypoint: model_def:OneVarPyTorchTrial
                environment:
                  environment_variables:
                    cpu:
                    - "HOME=/where/the/heart/is"
                    gpu:
                    - "HOME=/where/the/cuda/is"
                  force_pull_image: false
                  image:
                    cpu: determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.15-cpu-067db2b
                    gpu: determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.15-gpu-067db2b
                  pod_spec:
                    apiVersion: v1
                    kind: Pod
                    metadata:
                      labels:
                        customLabel: test-label
                    spec:
                      schedulerName: coscheduler
                      priorityClassName: determined-medium-priority
                      containers:
                        - name: determined-container
                          volumeMounts:
                          - name: test-volume
                            mountPath: /test
                      volumes:
                      - name: test-volume
                        hostPath:
                          path: /data
                  ports: null
                hyperparameters:
                  global_batch_size:
                    type: const
                    val: 32
                internal: null
                labels:
                - 0.14.3
                max_restarts: 0
                min_checkpoint_period:
                  batches: 0
                min_validation_period:
                  batches: 0
                optimizations:
                  aggregation_frequency: 1
                  auto_tune_tensor_fusion: false
                  average_aggregated_gradients: true
                  average_training_metrics: true
                  gradient_compression: false
                  mixed_precision: O0
                  tensor_fusion_cycle_time: 5
                  tensor_fusion_threshold: 64
                perform_initial_validation: false
                records_per_epoch: 0
                reproducibility:
                  experiment_seed: 1621979432
                resources:
                  agent_label: ''
                  native_parallel: false
                  resource_pool: default
                  slots_per_trial: 1
                  weight: 1
                scheduling_unit: 100
                searcher:
                  divisor: 4
                  max_length:
                    batches: 3
                  max_rungs: 5
                  max_trials: 3
                  metric: loss
                  mode: standard
                  name: adaptive_simple
                  smaller_is_better: true
                  source_checkpoint_uuid: null
                  source_trial_id: null
            `),
			Expected: LegacyConfig{
				CheckpointStorage: CheckpointStorageConfig{
					RawSharedFSConfig: &SharedFSConfig{
						RawHostPath:    ptrs.Ptr("/tmp"),
						RawStoragePath: ptrs.Ptr("determined-cp"),
						RawPropagation: ptrs.Ptr("rprivate"),
					},
					RawSaveExperimentBest: ptrs.Ptr(10),
					RawSaveTrialBest:      ptrs.Ptr(10),
					RawSaveTrialLatest:    ptrs.Ptr(10),
				},
				BindMounts: BindMountsConfig{},
				Environment: EnvironmentConfig{
					RawEnvironmentVariables: &EnvironmentVariablesMap{
						RawCPU:  []string{"HOME=/where/the/heart/is"},
						RawCUDA: []string{"HOME=/where/the/cuda/is"},
						RawROCM: []string{},
					},
					RawImage: &EnvironmentImageMap{
						RawCPU:  ptrs.Ptr("determinedai/environments:py-3.6.9-pytorch-1.4-tf-1.15-cpu-067db2b"),
						RawCUDA: ptrs.Ptr("determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.15-gpu-067db2b"),
						RawROCM: ptrs.Ptr("determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-2b7e2a1"),
					},
					RawPodSpec: &PodSpec{
						TypeMeta: metaV1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metaV1.ObjectMeta{
							Labels: map[string]string{"customLabel": "test-label"},
						},
						Spec: k8sV1.PodSpec{
							Volumes: []k8sV1.Volume{
								{
									Name: "test-volume",
									VolumeSource: k8sV1.VolumeSource{
										HostPath: &k8sV1.HostPathVolumeSource{
											Path: "/data",
											Type: nil,
										},
									},
								},
							},
							Containers: []k8sV1.Container{{
								Name:      "determined-container",
								Resources: k8sV1.ResourceRequirements{},
								VolumeMounts: []k8sV1.VolumeMount{{
									Name:      "test-volume",
									MountPath: "/test",
								}},
							}},
							SchedulerName:     "coscheduler",
							PriorityClassName: "determined-medium-priority",
						},
						Status: k8sV1.PodStatus{},
					},
					RawPorts:            map[string]int{},
					RawProxyPorts:       &ProxyPortsConfigV0{},
					RawForcePullImage:   ptrs.Ptr(false),
					RawAddCapabilities:  []string{},
					RawDropCapabilities: []string{},
				},
				Hyperparameters: Hyperparameters{
					"global_batch_size": {
						RawConstHyperparameter: &ConstHyperparameterV0{RawVal: float64(32)},
					},
				},
				Searcher: LegacySearcher{
					Name:            "adaptive_simple",
					Metric:          "loss",
					SmallerIsBetter: true,
				},
			},
		},
		// Test case with a 0.15.5 experiment config.
		{
			Name: "0.15.5 config with EOL searcher",
			Bytes: []byte(`
                bind_mounts:
                - container_path: /tmp/asdf
                  host_path: /tmp/asdf
                  propagation: rprivate
                  read_only: false
                checkpoint_policy: best
                checkpoint_storage:
                  host_path: /tmp
                  propagation: rprivate
                  save_experiment_best: 10
                  save_trial_best: 10
                  save_trial_latest: 10
                  storage_path: determined-cp
                  type: shared_fs
                data: {}
                data_layer:
                  container_storage_path: null
                  host_storage_path: null
                  type: shared_fs
                debug: false
                description: rb-test-dist-ctx
                entrypoint: model_def:OneVarPyTorchTrial
                environment:
                  add_capabilities: []
                  drop_capabilities: []
                  environment_variables:
                    cpu: []
                    gpu: []
                  force_pull_image: false
                  image:
                    cpu: determinedai/environments:py-3.7-pytorch-1.7-tf-1.15-cpu-6eceaca
                    gpu: determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-6eceaca
                  ports: {}
                  registry_auth: null
                hyperparameters:
                  global_batch_size:
                    type: const
                    val: 32
                labels: []
                max_restarts: 0
                min_checkpoint_period:
                  batches: 0
                min_validation_period:
                  batches: 1
                optimizations:
                  aggregation_frequency: 1
                  auto_tune_tensor_fusion: false
                  average_aggregated_gradients: true
                  average_training_metrics: true
                  grad_updates_size_file: null
                  gradient_compression: false
                  mixed_precision: O0
                  tensor_fusion_cycle_time: 5
                  tensor_fusion_threshold: 64
                perform_initial_validation: false
                profiling:
                  begin_on_batch: 0
                  enabled: false
                  end_after_batch: null
                records_per_epoch: 0
                reproducibility:
                  experiment_seed: 1622040996
                resources:
                  agent_label: ''
                  devices: []
                  max_slots: null
                  native_parallel: false
                  priority: null
                  resource_pool: default
                  shm_size: null
                  slots_per_trial: 1
                  weight: 1
                scheduling_unit: 100
                searcher:
                  max_length:
                    batches: 3
                  metric: loss
                  name: single
                  smaller_is_better: true
                  source_checkpoint_uuid: null
                  source_trial_id: null
            `),
			Expected: LegacyConfig{
				CheckpointStorage: CheckpointStorageConfig{
					RawSharedFSConfig: &SharedFSConfig{
						RawHostPath:    ptrs.Ptr("/tmp"),
						RawStoragePath: ptrs.Ptr("determined-cp"),
						RawPropagation: ptrs.Ptr("rprivate"),
					},
					RawSaveExperimentBest: ptrs.Ptr(10),
					RawSaveTrialBest:      ptrs.Ptr(10),
					RawSaveTrialLatest:    ptrs.Ptr(10),
				},
				BindMounts: BindMountsConfig{
					BindMount{
						RawHostPath:      "/tmp/asdf",
						RawContainerPath: "/tmp/asdf",
						RawReadOnly:      ptrs.Ptr(false),
						RawPropagation:   ptrs.Ptr("rprivate"),
					},
				},
				Environment: EnvironmentConfig{
					RawEnvironmentVariables: &EnvironmentVariablesMap{
						RawCPU:  []string{},
						RawCUDA: []string{},
						RawROCM: []string{},
					},
					RawImage: &EnvironmentImageMap{
						RawCPU:  ptrs.Ptr("determinedai/environments:py-3.7-pytorch-1.7-tf-1.15-cpu-6eceaca"),
						RawCUDA: ptrs.Ptr("determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-6eceaca"),
						RawROCM: ptrs.Ptr("determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-2b7e2a1"),
					},
					RawPorts:            map[string]int{},
					RawProxyPorts:       &ProxyPortsConfigV0{},
					RawForcePullImage:   ptrs.Ptr(false),
					RawAddCapabilities:  []string{},
					RawDropCapabilities: []string{},
				},
				Hyperparameters: Hyperparameters{
					"global_batch_size": {
						RawConstHyperparameter: &ConstHyperparameterV0{RawVal: float64(32)},
					},
				},
				Searcher: LegacySearcher{
					Name:            "single",
					Metric:          "loss",
					SmallerIsBetter: true,
				},
			},
		},
		// Test case with a 0.22.2 hdfs checkpoint storage config.
		{
			Name: "0.22.2 hdfs checkpoint storage config",
			Bytes: []byte(`
                checkpoint_policy: best
                checkpoint_storage:
                  hdfs_url: example.com
                  hdfs_path: /example
                  user: test
                  type: hdfs
                  save_experiment_best: 10
                  save_trial_best: 10
                  save_trial_latest: 10
                description: hdfs-test
                entrypoint: model_def:OneVarPyTorchTrial
                hyperparameters:
                  global_batch_size:
                    type: const
                    val: 32
                scheduling_unit: 100
                environment:
                  image:
                    cpu: determinedai/environments:py-3.7-pytorch-1.7-tf-1.15-cpu-6eceaca
                    gpu: determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-6eceaca
                searcher:
                  max_length:
                    batches: 3
                  metric: loss
                  name: single
                  smaller_is_better: true
            `),
			Expected: LegacyConfig{
				CheckpointStorage: CheckpointStorageConfig{
					RawSharedFSConfig: &SharedFSConfig{
						RawHostPath:    ptrs.Ptr("/legacy-hdfs-checkpoint-path"),
						RawPropagation: ptrs.Ptr("rprivate"),
					},
					RawSaveExperimentBest: ptrs.Ptr(10),
					RawSaveTrialBest:      ptrs.Ptr(10),
					RawSaveTrialLatest:    ptrs.Ptr(10),
				},
				Hyperparameters: Hyperparameters{
					"global_batch_size": {
						RawConstHyperparameter: &ConstHyperparameterV0{RawVal: float64(32)},
					},
				},
				Searcher: LegacySearcher{
					Name:            "single",
					Metric:          "loss",
					SmallerIsBetter: true,
				},
				BindMounts: BindMountsConfig{},
				Environment: EnvironmentConfig{
					RawEnvironmentVariables: &EnvironmentVariablesMap{
						RawCPU:  []string{},
						RawCUDA: []string{},
						RawROCM: []string{},
					},
					RawImage: &EnvironmentImageMap{
						RawCPU:  ptrs.Ptr("determinedai/environments:py-3.7-pytorch-1.7-tf-1.15-cpu-6eceaca"),
						RawCUDA: ptrs.Ptr("determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-6eceaca"),
						RawROCM: ptrs.Ptr("determinedai/environments:rocm-5.0-pytorch-1.10-tf-2.7-rocm-2b7e2a1"),
					},
					RawAddCapabilities:  []string{},
					RawDropCapabilities: []string{},
					RawForcePullImage:   ptrs.Ptr(false),
					RawPorts:            map[string]int{},
					RawProxyPorts:       &ProxyPortsConfigV0{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			jByts, err := schemas.JSONFromYaml(tc.Bytes)
			assert.NilError(t, err)

			legacyConfig, err := ParseLegacyConfigJSON(jByts)
			assert.NilError(t, err)

			assert.DeepEqual(t, legacyConfig, tc.Expected)
		})
	}
}
