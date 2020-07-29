import { ColumnType } from 'antd/lib/table';
import React from 'react';

import {
  experimentActionRenderer, experimentProgressRenderer, expermentDurationRenderer,
  relativeTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import { ExperimentItem } from 'types';
import { alphanumericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { getDuration } from 'utils/time';

export const columns: ColumnType<ExperimentItem>[] = [
  {
    dataIndex: 'id',
    sorter: (a: ExperimentItem, b: ExperimentItem): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    defaultSortOrder: 'descend',
    render: (_: number, record: ExperimentItem): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: (a: ExperimentItem, b: ExperimentItem): number =>
      stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    render: expermentDurationRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number => getDuration(a) - getDuration(b),
    title: 'Duration',
  },
  {
    // TODO bring in actual trial counts once available.
    render: (): number => Math.floor(Math.random() * 100),
    title: 'Trials',
  },
  {
    render: stateRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    render: experimentProgressRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number => (a.progress || 0) - (b.progress || 0),
    title: 'Progress',
  },
  {
    render: userRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number =>
      alphanumericSorter(a.username, b.username),
    title: 'User',
  },
  {
    align: 'right',
    render: experimentActionRenderer,
    title: '',
  },
];
