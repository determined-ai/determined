import { ColumnType } from 'antd/es/table';

import {
  relativeTimeRenderer, stateRenderer, taskIdRenderer, taskTypeRenderer, userRenderer,
} from 'components/Table';
import { CommandTask } from 'types';
import {
  alphanumericSorter, commandStateSorter, stringTimeSorter,
} from 'utils/data';

export const columns: ColumnType<CommandTask>[] = [
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
