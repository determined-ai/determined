import React from 'react';

import { useGlasbey } from 'hooks/useGlasbey';
import { RunMetricData } from 'hooks/useMetrics';
import { Scale } from 'types';
import { generateTestRunData } from 'utils/tests/generateTestData';

import CompareHyperparameters from './CompareHyperparameters';
export const METRIC_DATA: RunMetricData = {
  data: {
    1: {
      '{"group":"training","name":"loss"}': {
        data: {
          Batches: [[2, 0.5823304653167725]],
          Epoch: [],
          Time: [[1656260140.728, 0.5823304653167725]],
        },
        name: 'training.loss',
      },
      '{"group":"validation","name":"accuracy"}': {
        data: {
          Batches: [[2, 0.8522093949044586]],
          Epoch: [],
          Time: [[1656260146.436, 0.8522093949044586]],
        },
        name: 'validation.accuracy',
      },
      '{"group":"validation","name":"validation_loss"}': {
        data: {
          Batches: [[2, 0.49773818169050155]],
          Epoch: [],
          Time: [[1656260146.436, 0.49773818169050155]],
        },
        name: 'validation.validation_loss',
      },
    },
    2: {
      '{"group":"training","name":"loss"}': {
        data: {
          Batches: [[2, 0.5823304653167725]],
          Epoch: [],
          Time: [[1656260140.728, 0.5823304653167725]],
        },
        name: 'training.loss',
      },
      '{"group":"validation","name":"accuracy"}': {
        data: {
          Batches: [[2, 0.8522093949044586]],
          Epoch: [],
          Time: [[1656260146.436, 0.8522093949044586]],
        },
        name: 'validation.accuracy',
      },
      '{"group":"validation","name":"validation_loss"}': {
        data: {
          Batches: [[2, 0.49773818169050155]],
          Epoch: [],
          Time: [[1656260146.436, 0.49773818169050155]],
        },
        name: 'validation.validation_loss',
      },
    },
    3: {
      '{"group":"training","name":"loss"}': {
        data: {
          Batches: [[2, 0.5823304653167725]],
          Epoch: [],
          Time: [[1656260140.728, 0.5823304653167725]],
        },
        name: 'training.loss',
      },
      '{"group":"validation","name":"accuracy"}': {
        data: {
          Batches: [[2, 0.8522093949044586]],
          Epoch: [],
          Time: [[1656260146.436, 0.8522093949044586]],
        },
        name: 'validation.accuracy',
      },
      '{"group":"validation","name":"validation_loss"}': {
        data: {
          Batches: [[2, 0.49773818169050155]],
          Epoch: [],
          Time: [[1656260146.436, 0.49773818169050155]],
        },
        name: 'validation.validation_loss',
      },
    },
  },
  isLoaded: true,
  metricHasData: {
    '{"group":"training","name":"loss"}': true,
    '{"group":"validation","name":"accuracy"}': true,
    '{"group":"validation","name":"validation_loss"}': true,
  },
  metrics: [
    {
      group: 'training',
      name: 'loss',
    },
    {
      group: 'validation',
      name: 'accuracy',
    },
    {
      group: 'validation',
      name: 'validation_loss',
    },
  ],
  scale: 'linear',
  selectedMetrics: [
    {
      group: 'training',
      name: 'loss',
    },
    {
      group: 'validation',
      name: 'accuracy',
    },
    {
      group: 'validation',
      name: 'validation_loss',
    },
  ],
  setScale: (): Scale => {
    return Scale.Linear;
  },
};

export const SELECTED_EXPERIMENTS = [
  {
    bestTrial: {
      autoRestarts: 0,
      bestAvailableCheckpoint: null,
      bestValidationMetric: {
        endTime: '2023-04-20T16:20:22.902226Z',
        metrics: {
          loss: 1,
        },
        totalBatches: 1,
      },
      checkpointCount: 1,
      endTime: '2022-06-26T16:16:04.171606Z',
      experimentId: 1156,
      hyperparameters: {
        dropout1: 0.532803505916605,
        dropout2: 0.39400711778394015,
        global_batch_size: 64,
        learning_rate: 0.06716139157036664,
        n_filters1: 54,
        n_filters2: 70,
      },
      id: 3400,
      latestValidationMetric: {
        endTime: '2022-06-26T16:15:46.436495Z',
        metrics: {
          accuracy: 0.8522093949044586,
          validation_loss: 0.49773818169050155,
        },
        totalBatches: 2,
      },
      searcherMetricsVal: 1,
      startTime: '2022-06-26T16:08:36.678225Z',
      state: 'COMPLETED',
      summaryMetrics: {
        avgMetrics: {
          loss: {
            count: 1,
            last: 0.5823304653167725,
            max: 0.582330465316772,
            mean: 0.582330465316772,
            min: 0.582330465316772,
            sum: 0.582330465316772,
            type: 'number',
          },
        },
        validationMetrics: {
          accuracy: {
            count: 1,
            last: 0.8522093949044586,
            max: 0.852209394904459,
            mean: 0.852209394904459,
            min: 0.852209394904459,
            sum: 0.852209394904459,
            type: 'number',
          },
          validation_loss: {
            count: 1,
            last: 0.49773818169050155,
            max: 0.497738181690502,
            mean: 0.497738181690502,
            min: 0.497738181690502,
            sum: 0.497738181690502,
            type: 'number',
          },
        },
      },
      totalBatchesProcessed: 100,
      totalCheckpointSize: 83008221,
    },
    experiment: {
      archived: false,
      checkpoints: 1,
      checkpointSize: 83008221,
      config: {
        bind_mounts: [],
        checkpoint_policy: 'best',
        checkpoint_storage: {
          access_key: null,
          bucket: 'det-determined-master-us-west-2-573932760021',
          endpoint_url: null,
          prefix: null,
          save_experiment_best: 0,
          save_trial_best: 1,
          save_trial_latest: 1,
          secret_key: null,
          type: 's3',
        },
        data: {},
        data_layer: {
          container_storage_path: null,
          host_storage_path: null,
          type: 'shared_fs',
        },
        debug: false,
        description: 'Continuation of trial 3355, experiment 1101 (Fork of dsdfs)',
        entrypoint: 'model_def:MNistTrial',
        environment: {
          add_capabilities: [],
          drop_capabilities: [],
          environment_variables: {
            cpu: [],
            cuda: [],
            rocm: [],
          },
          force_pull_image: false,
          image: {
            cpu: 'determinedai/environments:py-3.8-pytorch-1.10-lightning-1.5-tf-2.8-cpu-3e933ea',
            cuda: 'determinedai/environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpu-3e933ea',
            rocm: 'determinedai/environments:rocm-4.2-pytorch-1.9-tf-2.5-rocm-3e933ea',
          },
          pod_spec: null,
          ports: {},
          registry_auth: null,
        },
        hyperparameters: {
          dropout1: {
            type: 'const',
            val: 0.532803505916605,
          },
          dropout2: {
            type: 'const',
            val: 0.39400711778394015,
          },
          global_batch_size: {
            type: 'const',
            val: 64,
          },
          learning_rate: {
            type: 'const',
            val: 0.06716139157036664,
          },
          n_filters1: {
            type: 'const',
            val: 54,
          },
          n_filters2: {
            type: 'const',
            val: 70,
          },
        },
        labels: null,
        max_restarts: 5,
        min_checkpoint_period: {
          batches: 0,
        },
        min_validation_period: {
          batches: 0,
        },
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
          sync_timings: true,
        },
        project: 'Uncategorized',
        records_per_epoch: 10,
        reproducibility: {
          experiment_seed: 1654719872,
        },
        resources: {
          agent_label: '',
          devices: [],
          max_slots: null,
          native_parallel: false,
          priority: null,
          resource_pool: 'compute-pool',
          shm_size: null,
          slots_per_trial: 1,
          weight: 1,
        },
        scheduling_unit: 100,
        searcher: {
          max_length: {
            batches: 2,
          },
          metric: 'validation_loss',
          name: 'single',
          smaller_is_better: true,
          source_checkpoint_uuid: null,
          source_trial_id: 3355,
        },
        workspace: 'Uncategorized',
      },
      description: 'Continuation of trial 3355, experiment 1101 (Fork of dsdfs)',
      duration: 578,
      endTime: '2022-06-26T16:16:04.230714Z',
      forkedFrom: 1101,
      hyperparameters: {
        dropout1: {
          type: 'const',
          val: 0.532803505916605,
        },
        dropout2: {
          type: 'const',
          val: 0.39400711778394015,
        },
        global_batch_size: {
          type: 'const',
          val: 64,
        },
        learning_rate: {
          type: 'const',
          val: 0.06716139157036664,
        },
        n_filters1: {
          type: 'const',
          val: 54,
        },
        n_filters2: {
          type: 'const',
          val: 70,
        },
      },
      id: 1156,
      jobId: 'fb9d3c06-bf9d-4275-8d9d-94eedb1b90dc',
      labels: [],
      name: 'mnist_pytorch_adaptive_search',
      notes: '',
      numTrials: 1,
      progress: 1,
      projectId: 1,
      projectName: 'Uncategorized',
      resourcePool: 'compute-pool',
      searcherType: 'single',
      startTime: '2022-06-26T16:06:26.503777Z',
      state: 'COMPLETED',
      trialIds: [],
      unmanaged: false,
      userId: 2,
      workspaceId: 1,
      workspaceName: 'Uncategorized',
    },
  },
];

export const TRIALS = [
  {
    autoRestarts: 0,
    bestAvailableCheckpoint: null,
    bestValidationMetric: {
      endTime: '2023-04-20T16:20:22.902226Z',
      metrics: {
        loss: 1,
      },
      totalBatches: 1,
    },
    checkpointCount: 1,
    endTime: '2022-06-26T16:16:04.171606Z',
    experimentId: 1156,
    hyperparameters: {
      dropout1: 0.532803505916605,
      dropout2: 0.39400711778394015,
      global_batch_size: 64,
      learning_rate: 0.06716139157036664,
      n_filters1: 54,
      n_filters2: 70,
    },
    id: 1,
    latestValidationMetric: {
      endTime: '2022-06-26T16:15:46.436495Z',
      metrics: {
        accuracy: 0.8522093949044586,
        validation_loss: 0.49773818169050155,
      },
      totalBatches: 2,
    },
    searcherMetricsVal: 1,
    startTime: '2022-06-26T16:08:36.678225Z',
    state: 'COMPLETED',
    summaryMetrics: {
      avgMetrics: {
        loss: {
          count: 1,
          last: 0.5823304653167725,
          max: 0.582330465316772,
          mean: 0.582330465316772,
          min: 0.582330465316772,
          sum: 0.582330465316772,
          type: 'number',
        },
      },
      validationMetrics: {
        accuracy: {
          count: 1,
          last: 0.8522093949044586,
          max: 0.852209394904459,
          mean: 0.852209394904459,
          min: 0.852209394904459,
          sum: 0.852209394904459,
          type: 'number',
        },
        validation_loss: {
          count: 1,
          last: 0.49773818169050155,
          max: 0.497738181690502,
          mean: 0.497738181690502,
          min: 0.497738181690502,
          sum: 0.497738181690502,
          type: 'number',
        },
      },
    },
    totalBatchesProcessed: 100,
    totalCheckpointSize: 83008221,
  },
  {
    autoRestarts: 0,
    bestAvailableCheckpoint: null,
    bestValidationMetric: {
      endTime: '2023-04-20T16:20:22.902226Z',
      metrics: {
        loss: 1,
      },
      totalBatches: 1,
    },
    checkpointCount: 1,
    endTime: '2022-06-26T16:16:04.171606Z',
    experimentId: 1156,
    hyperparameters: {
      dropout1: 0.532803505916605,
      dropout2: 0.39400711778394015,
      global_batch_size: 64,
      learning_rate: 0.06716139157036664,
      n_filters1: 54,
      n_filters2: 70,
    },
    id: 2,
    latestValidationMetric: {
      endTime: '2022-06-26T16:15:46.436495Z',
      metrics: {
        accuracy: 0.8522093949044586,
        validation_loss: 0.49773818169050155,
      },
      totalBatches: 2,
    },
    searcherMetricsVal: 1,
    startTime: '2022-06-26T16:08:36.678225Z',
    state: 'COMPLETED',
    summaryMetrics: {
      avgMetrics: {
        loss: {
          count: 1,
          last: 0.5823304653167725,
          max: 0.582330465316772,
          mean: 0.582330465316772,
          min: 0.582330465316772,
          sum: 0.582330465316772,
          type: 'number',
        },
      },
      validationMetrics: {
        accuracy: {
          count: 1,
          last: 0.8522093949044586,
          max: 0.852209394904459,
          mean: 0.852209394904459,
          min: 0.852209394904459,
          sum: 0.852209394904459,
          type: 'number',
        },
        validation_loss: {
          count: 1,
          last: 0.49773818169050155,
          max: 0.497738181690502,
          mean: 0.497738181690502,
          min: 0.497738181690502,
          sum: 0.497738181690502,
          type: 'number',
        },
      },
    },
    totalBatchesProcessed: 100,
    totalCheckpointSize: 83008221,
  },
  {
    autoRestarts: 0,
    bestAvailableCheckpoint: null,
    bestValidationMetric: {
      endTime: '2023-04-20T16:20:22.902226Z',
      metrics: {
        loss: 1,
      },
      totalBatches: 1,
    },
    checkpointCount: 1,
    endTime: '2022-06-26T16:16:04.171606Z',
    experimentId: 1156,
    hyperparameters: {
      dropout1: 0.532803505916605,
      dropout2: 0.39400711778394015,
      global_batch_size: 64,
      learning_rate: 0.06716139157036664,
      n_filters1: 54,
      n_filters2: 70,
    },
    id: 3,
    latestValidationMetric: {
      endTime: '2022-06-26T16:15:46.436495Z',
      metrics: {
        accuracy: 0.8522093949044586,
        validation_loss: 0.49773818169050155,
      },
      totalBatches: 2,
    },
    searcherMetricsVal: 1,
    startTime: '2022-06-26T16:08:36.678225Z',
    state: 'COMPLETED',
    summaryMetrics: {
      avgMetrics: {
        loss: {
          count: 1,
          last: 0.5823304653167725,
          max: 0.582330465316772,
          mean: 0.582330465316772,
          min: 0.582330465316772,
          sum: 0.582330465316772,
          type: 'number',
        },
      },
      validationMetrics: {
        accuracy: {
          count: 1,
          last: 0.8522093949044586,
          max: 0.852209394904459,
          mean: 0.852209394904459,
          min: 0.852209394904459,
          sum: 0.852209394904459,
          type: 'number',
        },
        validation_loss: {
          count: 1,
          last: 0.49773818169050155,
          max: 0.497738181690502,
          mean: 0.497738181690502,
          min: 0.497738181690502,
          sum: 0.497738181690502,
          type: 'number',
        },
      },
    },
    totalBatchesProcessed: 100,
    totalCheckpointSize: 83008221,
  },
];

export const SELECTED_RUNS = [
  generateTestRunData(1),
  generateTestRunData(2),
  generateTestRunData(3),
];

interface Props {
  state: 'empty' | 'uncomparable_metrics' | 'no_metrics' | 'normal';
}
export const CompareTrialHyperparametersWithMocks: React.FC<Props> = ({
  state,
}: Props): JSX.Element => {
  const colorMap = useGlasbey(SELECTED_EXPERIMENTS.map((exp) => exp.experiment.id));
  const metricData =
    state === 'uncomparable_metrics'
      ? { ...METRIC_DATA, data: {} }
      : state === 'no_metrics'
        ? { ...METRIC_DATA, metrics: [] }
        : METRIC_DATA;
  return (
    <CompareHyperparameters
      colorMap={colorMap}
      metricData={metricData}
      projectId={1}
      // @ts-expect-error Mock data does not need type checking
      selectedExperiments={state === 'empty' ? [] : SELECTED_EXPERIMENTS}
      // @ts-expect-error Mock data does not need type checking
      trials={state === 'empty' ? [] : TRIALS}
    />
  );
};

export const CompareRunHyperparametersWithMocks: React.FC<Props> = ({
  state,
}: Props): JSX.Element => {
  const colorMap = useGlasbey(SELECTED_RUNS.map((run) => run.id));
  const metricData =
    state === 'uncomparable_metrics'
      ? { ...METRIC_DATA, data: {} }
      : state === 'no_metrics'
        ? { ...METRIC_DATA, metrics: [] }
        : METRIC_DATA;
  return (
    <CompareHyperparameters
      colorMap={colorMap}
      metricData={metricData}
      projectId={1}
      selectedRuns={state === 'empty' ? [] : SELECTED_RUNS}
    />
  );
};
