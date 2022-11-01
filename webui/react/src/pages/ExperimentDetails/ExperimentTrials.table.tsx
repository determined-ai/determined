import { ReactNode } from 'react';

import { ColumnDef } from 'components/Table/InteractiveTable';
import { durationRenderer, expStateRenderer, relativeTimeRenderer } from 'components/Table/Table';
import { V1GetExperimentTrialsRequestSortBy } from 'services/api-ts-sdk';
import { TrialItem } from 'types';

import { DEFAULT_COLUMN_WIDTHS } from './ExperimentTrials.settings';

export const columns: ColumnDef<TrialItem>[] = [
  {
    dataIndex: 'id',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['id'],
    key: V1GetExperimentTrialsRequestSortBy.ID,
    sorter: true,
    title: 'ID',
  },
  {
    dataIndex: 'state',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['state'],
    key: V1GetExperimentTrialsRequestSortBy.STATE,
    render: expStateRenderer,
    sorter: true,
    title: 'State',
  },
  {
    dataIndex: 'totalBatchesProcessed',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['totalBatchesProcessed'],
    key: V1GetExperimentTrialsRequestSortBy.BATCHESPROCESSED,
    sorter: true,
    title: 'Batches',
  },
  {
    dataIndex: 'bestValidationMetric',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['bestValidationMetric'],
    key: V1GetExperimentTrialsRequestSortBy.BESTVALIDATIONMETRIC,
    sorter: true,
    title: 'Best Validation Metric',
  },
  {
    dataIndex: 'latestValidationMetric',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['latestValidationMetric'],
    key: V1GetExperimentTrialsRequestSortBy.LATESTVALIDATIONMETRIC,
    sorter: true,
    title: 'Latest Validation Metric',
  },
  {
    dataIndex: 'startTime',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['startTime'],
    key: V1GetExperimentTrialsRequestSortBy.STARTTIME,
    render: (_: string, record: TrialItem): ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'Start Time',
  },
  {
    dataIndex: 'duration',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['duration'],
    key: V1GetExperimentTrialsRequestSortBy.DURATION,
    render: (_: string, record: TrialItem): ReactNode => durationRenderer(record),
    sorter: true,
    title: 'Duration',
  },
  {
    dataIndex: 'autoRestarts',
    defaultWidth: DEFAULT_COLUMN_WIDTHS['autoRestarts'],
    key: V1GetExperimentTrialsRequestSortBy.RESTARTS,
    sorter: true,
    title: 'Auto Restarts',
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
