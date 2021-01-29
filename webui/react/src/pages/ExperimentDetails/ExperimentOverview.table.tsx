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
    key: V1GetExperimentTrialsRequestSortBy.STATE,
    render: stateRenderer,
    title: 'State',
  },
  {
    dataIndex: 'totalBatchesProcessed',
    key: 'batches',
    title: 'Batches',
  },
  {
    key: 'bestValidation',
    title: 'Best Validation Metric',
  },
  {
    key: 'latestValidation',
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
    key: 'duration',
    render: (_: string, record: TrialItem): ReactNode => durationRenderer(record),
    title: 'Duration',
  },
  {
    key: 'checkpoint',
    title: 'Checkpoint',
  },
];
