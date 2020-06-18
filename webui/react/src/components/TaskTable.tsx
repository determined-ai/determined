import { Table } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import React from 'react';
import TimeAgo from 'timeago-react';

import { CommandState, CommandTask, CommandType, CommonProps } from 'types';
import { alphanumericSorter, commandStateSorter, stringTimeSorter } from 'utils/data';
import { canBeOpened } from 'utils/task';
import { commandTypeToLabel } from 'utils/types';

import Avatar from './Avatar';
import Badge from './Badge';
import { BadgeType } from './Badge';
import Icon from './Icon';
import { makeClickHandler } from './Link';
import linkCss from './Link.module.scss';
import TaskActionDropdown from './TaskActionDropdown';
import css from './TaskTable.module.scss';

interface Props extends CommonProps {
  tasks?: CommandTask[];
}

  type Renderer<T> = (text: string, record: T, index: number) => React.ReactNode

const typeRenderer: Renderer<CommandTask> = (_, record) =>
  (<Icon name={record.type.toLowerCase()}
    title={commandTypeToLabel[record.type as unknown as CommandType]} />);
const startTimeRenderer: Renderer<CommandTask> = (_, record) => (
  <span title={new Date(parseInt(record.startTime) * 1000).toTimeString()}>
    <TimeAgo datetime={record.startTime} />
  </span>
);
const stateRenderer: Renderer<CommandTask> = (_, record) => (
  <Badge state={record.state} type={BadgeType.State} />
);
const actionsRenderer: Renderer<CommandTask> = (_, record) =>
  (<TaskActionDropdown task={record} />);
const userRenderer: Renderer<CommandTask> = (_, record) =>
  (<Avatar name={record.username || record.id} />);

const columns: ColumnsType<CommandTask> = [
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
    render: stateRenderer,
    sorter: (a, b): number => commandStateSorter(a.state as CommandState, b.state as CommandState),
    title: 'State',
  },
  {
    render: userRenderer,
    sorter: (a, b): number =>
      alphanumericSorter(a.username || a.ownerId.toString(), b.username || b.ownerId.toString()),
    title: 'User',
  },
  {
    render: actionsRenderer,
    title: '',
  },
];

const TaskTable: React.FC<Props> = ({ tasks }: Props) => {
  return (
    <Table
      className={css.base}
      columns={columns}
      dataSource={tasks}
      loading={tasks === undefined}
      rowClassName={(record) => canBeOpened(record) ? linkCss.base : ''}
      rowKey="id"
      onRow={(record) => {
        return {
          // can't use an actual link element on the whole row since anchor tag is not a valid
          // direct tr child https://developer.mozilla.org/en-US/docs/Web/HTML/Element/tr
          onClick: canBeOpened(record) ? makeClickHandler(record.url as string) : undefined,
        };
      }} />

  );
};

export default TaskTable;
