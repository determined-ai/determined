import { ColumnType } from 'antd/es/table';
import { ReactNode } from 'react';

import {
  durationRenderer, humanReadableFloatRenderer, relativeTimeRenderer, stateRenderer,
} from 'components/Table';
import { TrialItem } from 'types';
import { alphanumericSorter, numericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { getDuration } from 'utils/time';

export const columns: ColumnType<TrialItem>[] = [
  {
    dataIndex: 'id',
    key: 'id',
    sorter: (a: TrialItem, b: TrialItem): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    key: 'state',
    render: stateRenderer,
    sorter: (a: TrialItem, b: TrialItem): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    dataIndex: 'totalBatchesProcessed',
    key: 'batches',
    sorter: (a: TrialItem, b: TrialItem): number => {
      return numericSorter(a.totalBatchesProcessed, b.totalBatchesProcessed);
    },
    title: 'Batches',
  },
  {
    key: 'bestValidation',
    render: (_: string, record: TrialItem): ReactNode =>
      record.bestValidationMetric && humanReadableFloatRenderer(record.bestValidationMetric),
    sorter: (a: TrialItem, b: TrialItem): number => {
      return numericSorter(a.bestValidationMetric, b.bestValidationMetric);
    },
    title: 'Best Validation Metric',
  },
  {
    key: 'latestValidation',
    title: 'Latest Validation Metric',
  },
  {
    key: 'startTime',
    render: (_: string, record: TrialItem): ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: (a: TrialItem, b: TrialItem): number => stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    key: 'duration',
    render: (_: string, record: TrialItem): ReactNode => durationRenderer(record),
    sorter: (a: TrialItem, b: TrialItem): number => getDuration(a) - getDuration(b),
    title: 'Duration',
  },
  {
    key: 'checkpoint',
    title: 'Checkpoint',
  },
];
