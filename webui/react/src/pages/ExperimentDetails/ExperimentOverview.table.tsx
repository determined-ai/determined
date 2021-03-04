import { ColumnType } from 'antd/es/table';
import { ReactNode } from 'react';

import {
  durationRenderer, relativeTimeRenderer, stateRenderer,
} from 'components/Table';
import { V1GetExperimentTrialsRequestSortBy } from 'services/api-ts-sdk';
import { TrialItem } from 'types';

export const columns: ColumnType<TrialItem>[] = [
  {
    dataIndex: 'id',
    key: V1GetExperimentTrialsRequestSortBy.ID,
    sorter: true,
    title: 'ID',
  },
  {
    dataIndex: 'totalBatchesProcessed',
    key: V1GetExperimentTrialsRequestSortBy.BATCHESPROCESSED,
    sorter: true,
    title: 'Batches',
  },
  {
    key: V1GetExperimentTrialsRequestSortBy.STATE,
    render: stateRenderer,
    title: 'State',
  },
  {
    key: V1GetExperimentTrialsRequestSortBy.BESTVALIDATIONMETRIC,
    sorter: true,
    title: 'Best Validation Metric',
  },
  {
    key: V1GetExperimentTrialsRequestSortBy.LATESTVALIDATIONMETRIC,
    sorter: true,
    title: 'Latest Validation Metric',
  },
  {
    key: V1GetExperimentTrialsRequestSortBy.STARTTIME,
    render: (_: string, record: TrialItem): ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: true,
    title: 'Start Time',
  },
  {
    key: V1GetExperimentTrialsRequestSortBy.DURATION,
    render: (_: string, record: TrialItem): ReactNode => durationRenderer(record),
    sorter: true,
    title: 'Duration',
  },
  {
    key: 'checkpoint',
    title: 'Checkpoint',
  },
];
