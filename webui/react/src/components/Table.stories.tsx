import { Table } from 'antd';
import { ColumnType } from 'antd/es/table';
import React from 'react';

import RouterDecorator from 'storybook/RouterDecorator';
import { CommandTask } from 'types';
import { alphanumericSorter, commandStateSorter, stringTimeSorter } from 'utils/sort';
import { generateCommandTask } from 'utils/task';

import {
  defaultRowClassName, relativeTimeRenderer, stateRenderer,
  taskIdRenderer, taskTypeRenderer, userRenderer,
} from './Table';
import css from './Table.module.scss';

export default {
  component: Table,
  decorators: [ RouterDecorator ],
  parameters: { layout: 'padded' },
  title: 'Table',
};

const columns: ColumnType<CommandTask>[] = [
  {
    dataIndex: 'id',
    key: 'id',
    render: taskIdRenderer,
    sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.id, b.id),
    title: 'Short ID',
  },
  {
    key: 'type',
    render: taskTypeRenderer,
    sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.type, b.type),
    title: 'Type',
  },
  {
    key: 'name',
    // render: added in TaskList.tsx
    sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.name, b.name),
    title: 'Name',
  },
  {
    key: 'startTime',
    render: (_: number, record: CommandTask): React.ReactNode =>
      relativeTimeRenderer(new Date(record.startTime)),
    sorter: (a: CommandTask, b: CommandTask): number => stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    key: 'state',
    render: stateRenderer,
    sorter: (a: CommandTask, b: CommandTask): number => commandStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    dataIndex: 'resourcePool',
    key: 'resourcePool',
    sorter: true,
    title: 'Resource Pool',
  },
  {
    key: 'user',
    render: userRenderer,
    sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.username, b.username),
    title: 'User',
  },
  {
    align: 'right',
    className: 'fullCell',
    key: 'action',
    title: '',
  },
];

const commandTasks: CommandTask[] = new Array(20)
  .fill(null)
  .map((_, index) => generateCommandTask(index));

export const LoadingTable = (): React.ReactNode => {
  return <Table loading={true} />;
};

export const EmptyTable = (): React.ReactNode => {
  return <Table dataSource={[]} />;
};

export const TaskTable = (): React.ReactNode => {
  return <Table
    className={css.base}
    columns={columns}
    dataSource={commandTasks}
    loading={commandTasks === undefined}
    rowClassName={defaultRowClassName({ clickable: true })}
    rowKey="id"
    showSorterTooltip={false} />;
};
