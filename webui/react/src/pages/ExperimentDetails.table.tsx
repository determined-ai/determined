import { ColumnType } from 'antd/es/table';
import { ReactNode } from 'react';

import {
  durationRenderer, humanReadableFloatRenderer, relativeTimeRenderer, stateRenderer,
} from 'components/Table';
import { TrialItem2 } from 'types';
import { alphanumericSorter, numericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { getDuration } from 'utils/time';
import { getMetricValue } from 'utils/types';

export const columns: ColumnType<TrialItem2>[] = [
  {
    dataIndex: 'id',
    key: 'id',
    sorter: (a: TrialItem2, b: TrialItem2): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    key: 'state',
    render: stateRenderer,
    sorter: (a: TrialItem2, b: TrialItem2): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    dataIndex: 'totalBatchesProcessed',
    key: 'batches',
    sorter: (a: TrialItem2, b: TrialItem2): number => {
      return numericSorter(a.totalBatchesProcessed, b.totalBatchesProcessed);
    },
    title: 'Batches',
  },
  {
    key: 'bestValidation',
    sorter: (a: TrialItem2, b: TrialItem2): number => {
      return numericSorter(
        getMetricValue(a.bestValidationMetric),
        getMetricValue(b.bestValidationMetric),
      );
    },
    title: 'Best Validation Metric',
  },
  {
    key: 'latestValidation',
    title: 'Latest Validation Metric',
  },
  {
    key: 'startTime',
    render: (_: string, record: TrialItem2): ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: (a: TrialItem2, b: TrialItem2): number => stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    key: 'duration',
    render: (_: string, record: TrialItem2): ReactNode => durationRenderer(record),
    sorter: (a: TrialItem2, b: TrialItem2): number => getDuration(a) - getDuration(b),
    title: 'Duration',
  },
  {
    key: 'checkpoint',
    title: 'Checkpoint',
  },
];
