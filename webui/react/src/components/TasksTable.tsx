import { Table } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import React from 'react';
import TimeAgo from 'timeago-react';

import Avatar from 'components/Avatar';
import Badge from 'components/Badge';
import linkCss from 'components/Link.module.scss';
import { CommonProps, Task } from 'types';
import { alphanumericSorter, stateSorter, stringTimeSorter } from 'utils/data';
import { canBeOpened } from 'utils/task';

import { BadgeType } from './Badge';
import Icon from './Icon';
import { handleClick } from './Link';
import TaskActionDropdown from './TaskActionDropdown';
import css from './TasksTable.module.scss';

interface Props extends CommonProps {
  tasks: Task[];
}

  type Renderer<T> = (text: string, record: T, index: number) => React.ReactNode

const typeRenderer: Renderer<Task> = (_, record) =>
  (<Icon name={record.type.toLowerCase()} />);
const startTimeRenderer: Renderer<Task> = (_, record) =>
  (
    <span title={record.startTime}>
      <TimeAgo datetime={record.startTime} />
    </span>
  );
const stateRenderer: Renderer<Task> = (_, record) => (
  <Badge state={record.state} type={BadgeType.State} />
);
const actionsRenderer: Renderer<Task> = (_, record) => (<TaskActionDropdown task={record} />);
const userRenderer: Renderer<Task> = (_, record) => (<Avatar name={record.id} />);

const columns: ColumnsType<Task> = [
  {
    dataIndex: 'id',
    sorter: (a, b): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
  },
  {
    render: typeRenderer,
    sorter: (a, b): number => alphanumericSorter(a.type, b.type),
    title: 'Type',
  },
  {
    dataIndex: 'title',
    sorter: (a, b): number => alphanumericSorter(a.title, b.title),
    title: 'Description',
  },
  {
    defaultSortOrder: 'descend',
    render: startTimeRenderer,
    sorter: (a, b): number => stringTimeSorter(a.startTime, b.startTime),
    title: 'Start Time',
  },
  {
    title: 'Duration',
  },
  {
    render: stateRenderer,
    sorter: (a, b): number => stateSorter(a.state, b.state),
    title: 'State',
  },
  {
    dataIndex: 'ownerId',
    render: userRenderer,
    title: 'User ID',
  },
  {
    render: actionsRenderer,
    title: '',
  },
];

const TasksTable: React.FC<Props> = ({ tasks }: Props) => {
  return (
    <Table
      className={css.base}
      columns={columns}
      dataSource={tasks}
      rowClassName={(record) => canBeOpened(record) ? linkCss.base : ''}
      rowKey="id"
      onRow={(record) => {
        return {
          // can't use an actual link element on the whole row since anchor tag is not a valid
          // direct tr child https://developer.mozilla.org/en-US/docs/Web/HTML/Element/tr
          onClick: canBeOpened(record) ? handleClick(record.url as string) : undefined,
        };
      }} />

  );
};

export default TasksTable;
