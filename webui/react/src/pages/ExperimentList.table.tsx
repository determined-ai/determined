import { ColumnType } from 'antd/lib/table';
import React from 'react';

import {
  experimentActionRenderer, experimentArchivedRenderer, experimentProgressRenderer,
  expermentDurationRenderer, relativeTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import { ExperimentItem } from 'types';
import { alphanumericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { getDuration } from 'utils/time';

export const columns: ColumnType<ExperimentItem>[] = [
  {
    dataIndex: 'id',
    key: 'id',
    sorter: (a: ExperimentItem, b: ExperimentItem): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    dataIndex: 'name',
    key: 'name',
    sorter: (a: ExperimentItem, b: ExperimentItem): number => alphanumericSorter(a.name, b.name),
    title: 'Name',
  },
  {
    key: 'startTime',
    render: (_: number, record: ExperimentItem): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: (a: ExperimentItem, b: ExperimentItem): number =>
      stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    key: 'duration',
    render: expermentDurationRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number => getDuration(a) - getDuration(b),
    title: 'Duration',
  },
  // TODO bring in actual trial counts once available.
  {
    key: 'state',
    render: stateRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    key: 'progress',
    render: experimentProgressRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number => (a.progress || 0) - (b.progress || 0),
    title: 'Progress',
  },
  {
    key: 'archived',
    render: experimentArchivedRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number =>
      (a.archived === b.archived) ? 0 : (a.archived ? 1 : -1),
    title: 'Archived',
  },
  {
    key: 'user',
    render: userRenderer,
    sorter: (a: ExperimentItem, b: ExperimentItem): number =>
      alphanumericSorter(a.username, b.username),
    title: 'User',
  },
  {
    align: 'right',
    className: 'fullCell',
    key: 'action',
    render: experimentActionRenderer,
    title: '',
  },
];
