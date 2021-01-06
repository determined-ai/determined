import { ColumnType } from 'antd/es/table';
import { ReactNode } from 'react';

import {
  durationRenderer, relativeTimeRenderer, stateRenderer,
} from 'components/Table';
import { ResourcePool } from 'types/ResourcePool';
import { alphanumericSorter, numericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { getDuration } from 'utils/time';
import { getMetricValue } from 'utils/types';

export const columns: ColumnType<ResourcePool>[] = [
  {
    dataIndex: 'id',
    key: 'id',
    sorter: (a: ResourcePool, b: ResourcePool): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    key: 'state',
    render: stateRenderer,
    sorter: (a: ResourcePool, b: ResourcePool): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    dataIndex: 'totalBatchesProcessed',
    key: 'batches',
    sorter: (a: ResourcePool, b: ResourcePool): number => {
      return numericSorter(a.totalBatchesProcessed, b.totalBatchesProcessed);
    },
    title: 'Batches',
  },
  {
    key: 'bestValidation',
    sorter: (a: ResourcePool, b: ResourcePool): number => {
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
    render: (_: string, record: ResourcePool): ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: (a: ResourcePool, b: ResourcePool): number =>
      stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    key: 'duration',
    render: (_: string, record: ResourcePool): ReactNode => durationRenderer(record),
    sorter: (a: ResourcePool, b: ResourcePool): number => getDuration(a) - getDuration(b),
    title: 'Duration',
  },
  {
    key: 'checkpoint',
    title: 'Checkpoint',
  },
];
