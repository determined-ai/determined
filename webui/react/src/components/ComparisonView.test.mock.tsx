import React from 'react';

import { useGlasbey } from 'hooks/useGlasbey';
import { RunMetricData } from 'hooks/useMetrics';
import { ExperimentWithTrial, Scale } from 'types';
import { generateTestRunData } from 'utils/tests/generateTestData';

import ComparisonView from './ComparisonView';

export const METRIC_DATA: RunMetricData = {
  data: {
    3400: {
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

export const SELECTED_EXPERIMENTS: ExperimentWithTrial[] = [
  {
    bestTrial: {
      autoRestarts: 0,
      bestAvailableCheckpoint: undefined,
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
            min: 0.852209394904459,
            sum: 0.852209394904459,
            type: 'number',
          },
          validation_loss: {
            count: 1,
            last: 0.49773818169050155,
            max: 0.497738181690502,
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
        checkpointPolicy: 'best',
        description: 'Continuation of trial 3355, experiment 1101 (Fork of dsdfs)',
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
        labels: [],
        maxRestarts: 5,
        name: 'mnist_pytorch_adaptive_search',
        profiling: {
          enabled: false,
        },
        resources: {},
        searcher: {
          max_length: undefined,
          metric: 'validation_loss',
          name: 'single',
          smallerIsBetter: true,
          sourceTrialId: 3355,
        },
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

export const SELECTED_RUNS = [generateTestRunData(), generateTestRunData(), generateTestRunData()];

interface Props {
  children: React.ReactElement;
  empty?: boolean;
  open: boolean;
  onWidthChange: (width: number) => void;
}

export const ExperimentComparisonViewWithMocks: React.FC<Props> = ({
  children,
  empty,
  onWidthChange,
  open,
}: Props): JSX.Element => {
  const colorMap = useGlasbey(SELECTED_EXPERIMENTS.map((exp) => exp.experiment.id));
  return (
    <ComparisonView
      colorMap={colorMap}
      experimentSelection={
        empty
          ? { selections: [], type: 'ONLY_IN' }
          : { selections: SELECTED_EXPERIMENTS.map((exp) => exp.experiment.id), type: 'ONLY_IN' }
      }
      fixedColumnsCount={2}
      initialWidth={200}
      open={open}
      projectId={1}
      onWidthChange={onWidthChange}>
      {children}
    </ComparisonView>
  );
};

export const RunComparisonViewWithMocks: React.FC<Props> = ({
  children,
  empty,
  onWidthChange,
  open,
}: Props): JSX.Element => {
  const colorMap = useGlasbey(SELECTED_RUNS.map((run) => run.id));
  return (
    <ComparisonView
      colorMap={colorMap}
      fixedColumnsCount={2}
      initialWidth={200}
      open={open}
      projectId={1}
      runSelection={
        empty
          ? { selections: [], type: 'ONLY_IN' }
          : { selections: SELECTED_RUNS.map((run) => run.id), type: 'ONLY_IN' }
      }
      onWidthChange={onWidthChange}>
      {children}
    </ComparisonView>
  );
};
