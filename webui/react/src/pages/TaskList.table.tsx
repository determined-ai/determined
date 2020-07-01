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
    sorter: (a, b): number => alphanumericSorter(a.id, b.id),
    title: 'Short ID',
  },
  {
    render: taskTypeRenderer,
    sorter: (a, b): number => alphanumericSorter(a.type, b.type),
    title: 'Type',
  },
  {
    dataIndex: 'title',
    sorter: (a, b): number => alphanumericSorter(a.title, b.title),
    title: 'Name',
  },
  {
    defaultSortOrder: 'descend',
    render: startTimeRenderer,
    sorter: (a, b): number => stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    render: stateRenderer,
    sorter: (a, b): number => commandStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    render: userRenderer,
    sorter: (a, b): number => alphanumericSorter(a.username, b.username),
    title: 'User',
  },
  {
    render: actionsRenderer,
    title: '',
  },
];
