import { GridCell, GridCellKind } from '@glideapps/glide-data-grid';

import { DateString, decode, optional } from 'ioTypes';
import { FlatRun } from 'types';

import { FilterFormSetWithoutId } from './FilterForm/components/type';

export const run: FlatRun = {
  archived: false,
  checkpointCount: 1,
  checkpointSize: 43090,
  duration: 256,
  endTime: decode(optional(DateString), '2024-06-03T17:50:38.703259Z'),
  experiment: {
    description: '',
    forkedFrom: 6634,
    id: 6833,
    isMultitrial: true,
    name: 'iris_tf_keras_adaptive_search',
    progress: 0.9444444,
    resourcePool: 'compute-pool',
    searcherMetric: 'val_categorical_accuracy',
    searcherType: 'adaptive_asha',
    unmanaged: false,
  },
  hyperparameters: {
    global_batch_size: 22,
    layer1_dense_size: 29,
    learning_rate: 0.00004998215062737775,
    learning_rate_decay: 0.000001,
  },
  id: 45888,
  labels: ['a', 'b'],
  parentArchived: false,
  projectId: 1,
  projectName: 'Uncategorized',
  searcherMetricValue: 0.46666666865348816,
  startTime: decode(optional(DateString), '2024-06-03T17:46:22.682019Z'),
  state: 'COMPLETED',
  summaryMetrics: {
    avgMetrics: {
      categorical_accuracy: {
        count: 1,
        last: 0.2968127429485321,
        max: 0.2968127429485321,
        min: 0.2968127429485321,
        sum: 0.2968127429485321,
        type: 'number',
      },
      loss: {
        count: 1,
        last: 2.4582924842834473,
        max: 2.4582924842834473,
        min: 2.4582924842834473,
        sum: 2.4582924842834473,
        type: 'number',
      },
    },
    validationMetrics: {
      val_categorical_accuracy: {
        count: 1,
        last: 0.46666666865348816,
        max: 0.46666666865348816,
        min: 0.46666666865348816,
        sum: 0.46666666865348816,
        type: 'number',
      },
      val_loss: {
        count: 1,
        last: 1.8627476692199707,
        max: 1.8627476692199707,
        min: 1.8627476692199707,
        sum: 1.8627476692199707,
        type: 'number',
      },
    },
  },
  userId: 1354,
  workspaceId: 1,
  workspaceName: 'Uncategorized',
};

export const cell: GridCell = {
  allowOverlay: false,
  copyData: '45888',
  cursor: 'pointer',
  data: {
    kind: 'link-cell',
    link: {
      href: '/experiments/6833/trials/45888',
      title: '45888',
      unmanaged: false,
    },
    navigateOn: 'click',
    underlineOffset: 6,
  },
  kind: GridCellKind.Custom,
  readonly: true,
};

export const filterFormSetWithoutId: FilterFormSetWithoutId = {
  filterGroup: {
    children: [],
    conjunction: 'and',
    kind: 'group',
  },
  showArchived: false,
};
