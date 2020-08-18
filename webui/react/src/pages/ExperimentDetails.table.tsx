import { ColumnType } from 'antd/es/table';
import { ReactNode } from 'react';

import { durationRenderer, relativeTimeRenderer, stateRenderer } from 'components/Table';
import { TrialItem } from 'types';
import { alphanumericSorter, numericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { humanReadableFloat } from 'utils/string';
import { getDuration } from 'utils/time';

export const columns: ColumnType<TrialItem>[] = [
  {
    dataIndex: 'id',
    sorter: (a: TrialItem, b: TrialItem): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    render: stateRenderer,
    sorter: (a: TrialItem, b: TrialItem): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    dataIndex: 'totalBatchesProcessed',
    sorter: (a: TrialItem, b: TrialItem): number => {
      return numericSorter(a.totalBatchesProcessed, b.totalBatchesProcessed);
    },
    title: 'Batches',
  },
  {
    render: (_: string, record: TrialItem): ReactNode => {
      return record.bestValidationMetric ? humanReadableFloat(record.bestValidationMetric) : null;
    },
    sorter: (a: TrialItem, b: TrialItem): number => {
      return numericSorter(a.bestValidationMetric, b.bestValidationMetric);
    },
    title: 'Best Validation Metric',
  },
  { title: 'Latest Validation Metric' },
  {
    render: (_: string, record: TrialItem): ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: (a: TrialItem, b: TrialItem): number => stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    render: (_: string, record: TrialItem): ReactNode => durationRenderer(record),
    sorter: (a: TrialItem, b: TrialItem): number => getDuration(a) - getDuration(b),
    title: 'Duration',
  },
  { title: 'Checkpoint' },
];
