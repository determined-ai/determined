import { ColumnsType } from 'antd/lib/table';
import { CompareFn } from 'antd/lib/table/interface';

import {
  actionsRenderer, experimentDescriptionRenderer, experimentProgressRenderer,
  expermentDurationRenderer, startTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import { ExperimentItem } from 'types';
import { alphanumericSorter, runStateSorter, stringTimeSorter } from 'utils/data';
import { experimentDuration } from 'utils/time';
// import { ColumnType } from 'antd/es/table/interface';

type AlphaNumeric = number | string;

function sortRecords<T extends Record<string, unknown>>(key: string): CompareFn<T> {
  const compareFn: CompareFn<T> = (a: T, b: T) => {
    const [ aValue, bValue ] = [ a[key], b[key] ];
    if (typeof aValue === typeof bValue) {
      if (
        (typeof aValue === 'string' && typeof bValue === 'string')
      || (typeof aValue === 'number' && typeof bValue === 'number')
      )
        return alphanumericSorter(aValue, bValue);
    }
    return 0;
  };
  return compareFn;
}

export const columns: ColumnsType<ExperimentItem> = [
  {
    dataIndex: 'id',
    sorter: sortRecords<ExperimentItem>('id'),
    title: 'ID',
  },
  {
    dataIndex: 'name',
    render: experimentDescriptionRenderer,
    sorter: sortRecords<ExperimentItem>('name'),
    title: 'Name',
  },
  {
    defaultSortOrder: 'descend',
    render: startTimeRenderer,
    sorter: ((a, b) => stringTimeSorter(a.startTime, b.startTime)) as CompareFn<ExperimentItem>,
    title: 'Start Time',
  },
  {
    render: expermentDurationRenderer,
    sorter: (a, b) => experimentDuration(a) - experimentDuration(b),
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
    sorter: sortRecords<ExperimentItem>('username'),
    title: 'User',
  },
  {
    align: 'right',
    render: actionsRenderer,
    title: '',
  },
];
