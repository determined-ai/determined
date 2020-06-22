import { ColumnType } from 'antd/lib/table';
import React from 'react';
import TimeAgo from 'timeago-react';

import Avatar from 'components/Avatar';
import { BadgeType } from 'components/Badge';
import Badge from 'components/Badge';
import TaskActionDropdown from 'components/TaskActionDropdown';
import { CommandState, CommandTask, Experiment } from 'types';
import { alphanumericSorter, commandStateSorter,
  stringTimeSorter } from 'utils/data';
import { experimentToTask, isExperiment, oneOfProperties } from 'utils/types';

type TableRecord = CommandTask | Experiment;

export type Renderer<T> = (text: string, record: T, index: number) => React.ReactNode

const userRenderer: Renderer<TableRecord> = (_, record) => {
  if (isExperiment(record)) {
    // TODO present username once available on experiments endpoint.
    return <Avatar name={record.ownerId.toString()} />;
  } else {
    return <Avatar name={record.username} />;
  }
};

export const userColumn: ColumnType<TableRecord> = {
  render: userRenderer,
  sorter: (a: TableRecord, b: TableRecord): number => {
    const aValue = oneOfProperties<string|number>(a, [ 'username', 'ownerId' ]).toString();
    const bValue = oneOfProperties<string|number>(b, [ 'username', 'ownerId' ]).toString();
    return alphanumericSorter(aValue, bValue);
  },
  title: 'User',
};

export const stateRenderer: Renderer<TableRecord> = (_, record) => (
  <Badge state={record.state} type={BadgeType.State} />
);

export const stateColumn: ColumnType<TableRecord> = {
  render: stateRenderer,
  sorter: (a, b): number => commandStateSorter(a.state as CommandState, b.state as CommandState),
  title: 'State',
};

const startTimeRenderer: Renderer<TableRecord> = (_, record) => (
  <span title={new Date(parseInt(record.startTime) * 1000).toTimeString()}>
    <TimeAgo datetime={record.startTime} />
  </span>
);

export const startTimeColumn: ColumnType<TableRecord> = {
  defaultSortOrder: 'descend',
  render: startTimeRenderer,
  sorter: (a, b): number => stringTimeSorter(a.startTime, b.startTime),
  title: 'Start Time',
};

const actionsRenderer: Renderer<TableRecord> =
  (_, record) => {
    if (isExperiment(record)) {
      return <TaskActionDropdown task={experimentToTask(record)} />;
    } else {
      return <TaskActionDropdown task={record} />;
    }
  };

export const actionsColumn: ColumnType<TableRecord> = {
  render: actionsRenderer,
  title: '',
};
