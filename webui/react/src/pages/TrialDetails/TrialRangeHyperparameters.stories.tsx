import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import StoreDecorator from 'storybook/StoreDecorator';
import { CheckpointStorageType, ExperimentBase, ExperimentSearcherName,
  HyperparameterType, RunState, TrialDetails } from 'types';

import TrialRangeHyperparameters from './TrialRangeHyperparameters';

export default {
  component: TrialRangeHyperparameters,
  decorators: [ StoreDecorator ],
  title: 'TrialRangeHyperparameters',
};

const TrialRangeHyperparametersContainer = () => {
  const sampleExp: ExperimentBase = {
    archived: false,
    config: {
      checkpointPolicy: 'best',
      checkpointStorage: {
        hostPath: '/tmp',
        saveExperimentBest: 0,
        saveTrialBest: 1,
        saveTrialLatest: 1,
        storagePath: 'determined-checkpoint',
        type: CheckpointStorageType.SharedFS,
      },
      dataLayer: { type: 'shared_fs' },
      hyperparameters: {
        categorical: {
          maxval: 64,
          minval: 8,
          type: HyperparameterType.Categorical,
          vals: [ 8, 16, 32, 64 ],
        },
        constant: {
          type: HyperparameterType.Constant,
          val: 64,
        },
        double: {
          maxval: 0.8,
          minval: 0.2,
          type: HyperparameterType.Double,
        },
        log: {
          maxval: 1,
          minval: 0.0001,
          type: HyperparameterType.Log,
        },
      },
      labels: [],
      name: 'mnist_pytorch_adaptive_search',
      profiling: { enabled: false },
      resources: {},
      searcher: {
        metric: 'validation_loss',
        name: ExperimentSearcherName.AdaptiveAsha,
        smallerIsBetter: true,
      },
    },
    configRaw: {
      bind_mounts: [],
      checkpoint_policy: 'best',
      checkpoint_storage: {
        host_path: '/tmp',
        propagation: 'rprivate',
        save_experiment_best: 0,
        save_trial_best: 1,
        save_trial_latest: 1,
        storage_path: 'determined-checkpoint',
        type: 'shared_fs',
      },
      data:
      { url: 'https://s3-us-west-2.amazonaws.com/determined-ai-test-data/pytorch_mnist.tar.gz' },
      data_layer: {
        container_storage_path: null,
        host_storage_path: null,
        type: 'shared_fs',
      },
      debug: false,
      description: null,
      entrypoint: 'model_def:MNistTrial',
      environment: {
        add_capabilities: [],
        drop_capabilities: [],
        environment_variables: {
          cpu: [],
          gpu: [],
        },
        force_pull_image: false,
        image: {
          cpu: 'determinedai/environments:py-3.7-pytorch-1.7-tf-1.15-cpu-da845fc',
          gpu: 'determinedai/environments:cuda-10.2-pytorch-1.7-tf-1.15-gpu-da845fc',
        },
        pod_spec: null,
        ports: {},
        registry_auth: null,
      },
      hyperparameters: {
        dropout1: {
          maxval: 0.8,
          minval: 0.2,
          type: 'double',
        },
        dropout2: {
          maxval: 0.8,
          minval: 0.2,
          type: 'double',
        },
        global_batch_size: {
          type: 'const',
          val: 64,
        },
        learning_rate: {
          maxval: 1,
          minval: 0.0001,
          type: 'double',
        },
        n_filters1: {
          maxval: 64,
          minval: 8,
          type: 'int',
        },
        n_filters2: {
          maxval: 72,
          minval: 8,
          type: 'int',
        },
      },
      labels: [],
      max_restarts: 5,
      min_checkpoint_period: { batches: 0 },
      min_validation_period: { batches: 0 },
      name: 'mnist_pytorch_adaptive_search',
      optimizations: {
        aggregation_frequency: 1,
        auto_tune_tensor_fusion: false,
        average_aggregated_gradients: true,
        average_training_metrics: false,
        grad_updates_size_file: null,
        gradient_compression: false,
        mixed_precision: 'O0',
        tensor_fusion_cycle_time: 5,
        tensor_fusion_threshold: 64,
      },
      perform_initial_validation: false,
      profiling: {
        begin_on_batch: 0,
        enabled: false,
        end_after_batch: null,
      },
      records_per_epoch: 0,
      reproducibility: { experiment_seed: 1623252417 },
      resources: {
        agent_label: '',
        devices: [],
        max_slots: null,
        native_parallel: false,
        priority: null,
        resource_pool: 'default',
        shm_size: null,
        slots_per_trial: 1,
        weight: 1,
      },
      scheduling_unit: 100,
      searcher: {
        bracket_rungs: [],
        divisor: 4,
        max_concurrent_trials: 0,
        max_length: { batches: 937 },
        max_rungs: 5,
        max_trials: 16,
        metric: 'validation_loss',
        mode: 'standard',
        name: 'adaptive_asha',
        smaller_is_better: true,
        source_checkpoint_uuid: null,
        source_trial_id: null,
        stop_once: false,
      },
    },
    hyperparameters: {
      categorical: {
        maxval: 64,
        minval: 8,
        type: HyperparameterType.Categorical,
        vals: [ 8, 16, 32, 64 ],
      },
      constant: {
        type: HyperparameterType.Constant,
        val: 64,
      },
      double: {
        maxval: 0.8,
        minval: 0.2,
        type: HyperparameterType.Double,
      },
      log: {
        maxval: 1,
        minval: 0.0001,
        type: HyperparameterType.Log,
      },
    },
    id: 1,
    name: 'Sample Experiment',
    resourcePool: 'default',
    startTime: '2021-06-09T15:26:57.610700Z',
    state: RunState.Completed,
    username: 'determined',
  };
  const sampleTrial: TrialDetails = {
    endTime: '2021-06-09T15:35:27.464642Z',
    experimentId: 1,
    hyperparameters: {
      categorical: 16,
      constant: 64,
      double: 0.675007115766233,
      log: 0.5138800609919691,
    },
    id: 1,
    startTime: '2021-06-09T15:26:58.003220Z',
    state: RunState.Completed,
    totalBatchesProcessed: 58,
    workloads: [
      {
        validation: {
          endTime: '2021-06-09T15:28:20.278182Z',
          metrics: {
            accuracy: 0.9087380573248408,
            validation_loss: 0.29681510450970977,
          },
          startTime: '2021-06-09T15:28:16.319452Z',
          totalBatches: 58,
        },
      },
      {
        checkpoint: {
          endTime: '2021-06-09T15:28:16.314554Z',
          resources: {
            'code/': 0,
            'code/adaptive.yaml': 678,
            'code/const.yaml': 434,
            'code/data.py': 1444,
            'code/distributed.yaml': 499,
            'code/layers.py': 568,
            'code/model_def.py': 3789,
            'code/README.md': 1407,
            'state_dict.pth': 13691537,
          },
          startTime: '2021-06-09T15:28:16.109105Z',
          totalBatches: 58,
          uuid: '08b7345e-1dd8-4ec2-a49c-054814d8929e',
        },
      },
      {
        training: {
          endTime: '2021-06-09T15:28:16.099435Z',
          metrics: { loss: 0.9266772866249084 },
          startTime: '2021-06-09T15:26:58.013003Z',
          totalBatches: 58,
        },
      },
    ],
  };

  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return <TrialRangeHyperparameters experiment={sampleExp} trial={sampleTrial} />;
};

export const Default = (): React.ReactNode => {
  return <TrialRangeHyperparametersContainer />;
};
