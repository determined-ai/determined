import { ColumnsType } from 'antd/lib/table';

import {
  actionsRenderer, ellipsisRenderer, startTimeRenderer, stateRenderer,
  taskIdRenderer, taskTypeRenderer, userRenderer,
} from 'components/Table';
import { CommandTask } from 'types';
import { alphanumericSorter, commandStateSorter, stringTimeSorter } from 'utils/data';

export const columns: ColumnsType<CommandTask> = [
  {
    dataIndex: 'id',
    ellipsis: { showTitle: false },
    render: taskIdRenderer,
    sorter: (a, b): number => alphanumericSorter(a.id, b.id),
    title: 'Short ID',
    width: 100,
  },
  {
    render: taskTypeRenderer,
    sorter: (a, b): number => alphanumericSorter(a.type, b.type),
    title: 'Type',
    width: 70,
  },
  {
    dataIndex: 'title',
    ellipsis: { showTitle: false },
    render: ellipsisRenderer,
    sorter: (a, b): number => alphanumericSorter(a.title, b.title),
    title: 'Name',
  },
  {
    defaultSortOrder: 'descend',
    render: startTimeRenderer,
    sorter: (a, b): number => stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
    width: 120,
  },
  {
    render: stateRenderer,
    sorter: (a, b): number => commandStateSorter(a.state, b.state),
    title: 'State',
    width: 120,
  },
  {
    render: userRenderer,
    sorter: (a, b): number => alphanumericSorter(a.username, b.username),
    title: 'User',
    width: 70,
  },
  {
    render: actionsRenderer,
    title: '',
    width: 40,
  },
];
