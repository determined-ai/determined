import { Table } from 'antd';
import { ColumnsType, ColumnType } from 'antd/lib/table';
import React, { MouseEventHandler } from 'react';

import { actionsColumn, Renderer, startTimeColumn, stateColumn,
  userColumn } from 'table/columns';
import { AnyTask, CommandTask, CommandType, CommonProps } from 'types';
import { alphanumericSorter } from 'utils/data';
import { canBeOpened } from 'utils/task';
import { commandTypeToLabel } from 'utils/types';

import Icon from './Icon';
import { makeClickHandler } from './Link';
import linkCss from './Link.module.scss';
import css from './TaskTable.module.scss';

interface Props extends CommonProps {
  tasks?: CommandTask[];
}

const typeRenderer: Renderer<CommandTask> = (_, record) =>
  (<Icon name={record.type.toLowerCase()}
    title={commandTypeToLabel[record.type as unknown as CommandType]} />);

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
  startTimeColumn as ColumnType<CommandTask>,
  stateColumn as ColumnType<CommandTask>,
  userColumn as ColumnType<CommandTask>,
  actionsColumn as ColumnType<CommandTask>,
];

export const tableRowClickHandler = (record: AnyTask): {onClick?: MouseEventHandler} => (
  {
    /*
           * Can't use an actual link element on the whole row since anchor tag
           * is not a valid direct tr child.
           * https://developer.mozilla.org/en-US/docs/Web/HTML/Element/tr
           */
    onClick: canBeOpened(record) ? makeClickHandler(record.url as string) : undefined,
  }
);

const TaskTable: React.FC<Props> = ({ tasks }: Props) => {
  return (
    <Table
      className={css.base}
      columns={columns}
      dataSource={tasks}
      loading={tasks === undefined}
      rowClassName={(record): string => canBeOpened(record) ? linkCss.base : ''}
      rowKey="id"
      onRow={tableRowClickHandler} />
  );
};

export default TaskTable;
