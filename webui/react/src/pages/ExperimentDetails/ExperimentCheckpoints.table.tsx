import { ColumnDef } from 'components/InteractiveTable';
import { stateRenderer } from 'components/Table';
import { V1GetExperimentCheckpointsRequestSortBy } from 'services/api-ts-sdk';
import { CoreApiGenericCheckpoint } from 'types';

import { DEFAULT_COLUMN_WIDTHS } from './ExperimentCheckpoints.settings';

export const columns: ColumnDef<CoreApiGenericCheckpoint>[] = [
  {
    dataIndex: 'uuid',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['uuid'],
    key: V1GetExperimentCheckpointsRequestSortBy.UUID,
    sorter: true,
    title: 'UUID',
  },
  {
    dataIndex: 'totalBatches',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['totalBatches'],
    key: V1GetExperimentCheckpointsRequestSortBy.BATCHNUMBER,
    sorter: true,
    title: 'Total Batches',
  },
  {
    dataIndex: 'searcherMetric',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['searcherMetric'],
    key: V1GetExperimentCheckpointsRequestSortBy.SEARCHERMETRIC,
    sorter: true,
    title: 'Searcher Metric',
  },
  {
    dataIndex: 'state',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['state'],
    key: V1GetExperimentCheckpointsRequestSortBy.STATE,
    render: stateRenderer,
    sorter: true,
    title: 'State',
  },
  {
    dataIndex: 'checkpoint',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['checkpoint'],
    key: 'checkpoint',
    title: 'Checkpoint',
  },
  {
    align: 'right',
    className: 'fullCell',
    dataIndex: 'action',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
    fixed: 'right',
    key: 'actions',
    title: '',
    width: DEFAULT_COLUMN_WIDTHS['action'],
  },
];
