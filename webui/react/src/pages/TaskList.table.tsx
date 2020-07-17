import { ColumnsType } from 'antd/lib/table';

import {
  actionsRenderer, startTimeRenderer, stateRenderer, taskIdRenderer,
  taskTypeRenderer, userRenderer,
} from 'components/Table';
import { CommandTask } from 'types';
import { alphanumericSorter, commandStateSorter, stringTimeSorter } from 'utils/data';

export const columns: ColumnsType<CommandTask> = [
  {
    dataIndex: 'id',
    render: taskIdRenderer,
    sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.id, b.id),
    title: 'Short ID',
  },
  {
    render: taskTypeRenderer,
    sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.type, b.type),
    title: 'Type',
  },
  {
    dataIndex: 'name',
    sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.name, b.name),
    title: 'Name',
  },
  {
    defaultSortOrder: 'descend',
    render: startTimeRenderer,
    sorter: (a: CommandTask, b: CommandTask): number =>
      stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    render: stateRenderer,
    sorter: (a: CommandTask, b: CommandTask): number => commandStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    render: userRenderer,
    sorter: (a: CommandTask, b: CommandTask): number =>
      alphanumericSorter(a.username, b.username),
    title: 'User',
  },
  {
    render: actionsRenderer,
    title: '',
  },
];
