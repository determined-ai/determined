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
    width: 100,
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
    width: 120,
  },
  {
    render: expermentDurationRenderer,
    sorter: (a, b): number => experimentDuration(a) - experimentDuration(b),
    title: 'Duration',
    width: 100,
  },
  {
    // TODO bring in actual trial counts once available.
    render: (): number => Math.floor(Math.random() * 100),
    title: 'Trials',
    width: 100,
  },
  {
    render: stateRenderer,
    sorter: (a, b): number => runStateSorter(a.state, b.state),
    title: 'State',
    width: 120,
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
    width: 70,
  },
  {
    render: actionsRenderer,
    title: '',
    width: 36,
  },
];
