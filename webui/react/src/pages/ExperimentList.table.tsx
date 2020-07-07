import { ColumnsType } from 'antd/lib/table';

import {
  actionsRenderer, experimentDescriptionRenderer, experimentProgressRenderer,
  expermentDurationRenderer, startTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import { ExperimentItem } from 'types';
import { alphanumericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { experimentDuration } from 'utils/time';

export const columns: ColumnsType<ExperimentItem> = [
  {
    dataIndex: 'id',
    sorter: (a, b): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    dataIndex: 'name',
    render: experimentDescriptionRenderer,
    sorter: (a, b): number => alphanumericSorter(a.name, b.name),
    title: 'Name',
  },
  {
    defaultSortOrder: 'descend',
    render: startTimeRenderer,
    sorter: (a, b): number => stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    render: expermentDurationRenderer,
    sorter: (a, b): number => experimentDuration(a) - experimentDuration(b),
    title: 'Duration',
  },
  {
    // TODO bring in actual trial counts once available.
    render: (): number => Math.floor(Math.random() * 100),
    title: 'Trials',
  },
  {
    render: stateRenderer,
    sorter: (a, b): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    render: experimentProgressRenderer,
    sorter: (a, b): number => (a.progress || 0) - (b.progress || 0),
    title: 'Progress',
  },
  {
    render: userRenderer,
    sorter: (a, b): number => alphanumericSorter(a.username, b.username),
    title: 'User',
  },
  {
    align: 'right',
    render: actionsRenderer,
    title: '',
  },
];
