import { ColumnType } from 'antd/lib/table';
import React from 'react';

import {
  experimentActionRenderer, experimentArchivedRenderer, experimentProgressRenderer,
  expermentDurationRenderer, relativeTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import { ExperimentX } from 'types';
import { alphanumericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { getDuration } from 'utils/time';

export const columns: ColumnType<ExperimentX>[] = [
  {
    dataIndex: 'id',
    key: 'id',
    sorter: (a: ExperimentX, b: ExperimentX): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    dataIndex: 'name',
    key: 'name',
    sorter: (a: ExperimentX, b: ExperimentX): number => alphanumericSorter(a.name, b.name),
    title: 'Name',
  },
  {
    key: 'startTime',
    render: (_: number, record: ExperimentX): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: (a: ExperimentX, b: ExperimentX): number =>
      stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    key: 'duration',
    render: expermentDurationRenderer,
    sorter: (a: ExperimentX, b: ExperimentX): number => getDuration(a) - getDuration(b),
    title: 'Duration',
  },
  // TODO bring in actual trial counts once available.
  {
    key: 'state',
    render: stateRenderer,
    sorter: (a: ExperimentX, b: ExperimentX): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    key: 'progress',
    render: experimentProgressRenderer,
    sorter: (a: ExperimentX, b: ExperimentX): number => (a.progress || 0) - (b.progress || 0),
    title: 'Progress',
  },
  {
    key: 'archived',
    render: experimentArchivedRenderer,
    sorter: (a: ExperimentX, b: ExperimentX): number =>
      (a.archived === b.archived) ? 0 : (a.archived ? 1 : -1),
    title: 'Archived',
  },
  {
    key: 'user',
    render: userRenderer,
    sorter: (a: ExperimentX, b: ExperimentX): number =>
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
