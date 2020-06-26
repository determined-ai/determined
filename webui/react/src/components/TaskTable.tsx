import { Table, Tooltip } from 'antd';
import { ColumnsType, ColumnType } from 'antd/lib/table';
import React, { MouseEventHandler } from 'react';

import { AnyTask, CommandTask, CommandType, CommonProps } from 'types';
import { alphanumericSorter } from 'utils/data';
import { canBeOpened } from 'utils/task';
import { commandTypeToLabel } from 'utils/types';

import Badge, { BadgeType } from './Badge';
import Icon from './Icon';
import { makeClickHandler } from './Link';
import linkCss from './Link.module.scss';
import {
  actionsColumn, ellipsisRenderer, Renderer, startTimeColumn, stateColumn, userColumn,
} from './Table';
import css from './Table.module.scss';

interface Props extends CommonProps {
  tasks?: CommandTask[];
}

const idRenderer: Renderer<CommandTask> = id => {
  const shortId = id.split('-')[0];
  return (
    <Tooltip title={id}>
      <div className={css.centerVertically}>
        <Badge type={BadgeType.Id}>{shortId}</Badge>
      </div>
    </Tooltip>
  );
};

const typeRenderer: Renderer<CommandTask> = (_, record) => {
  return (
    <Tooltip placement="topLeft" title={commandTypeToLabel[record.type as unknown as CommandType]}>
      <div className={css.centerVertically}>
        <Icon name={record.type.toLowerCase()} />
      </div>
    </Tooltip>
  );
};

const columns: ColumnsType<CommandTask> = [
  {
    dataIndex: 'id',
    ellipsis: { showTitle: false },
    render: idRenderer,
    sorter: (a, b): number => alphanumericSorter(a.id, b.id),
    title: 'Short ID',
    width: 100,
  },
  {
    render: typeRenderer,
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
  startTimeColumn as ColumnType<CommandTask>,
  stateColumn as ColumnType<CommandTask>,
  userColumn as ColumnType<CommandTask>,
  actionsColumn as ColumnType<CommandTask>,
];

export const tableRowClickHandler = (record: AnyTask): {onClick?: MouseEventHandler} => ({
  /*
   * Can't use an actual link element on the whole row since anchor tag
   * is not a valid direct tr child.
   * https://developer.mozilla.org/en-US/docs/Web/HTML/Element/tr
   */
  onClick: canBeOpened(record) ? makeClickHandler(record.url as string) : undefined,
});

const TaskTable: React.FC<Props> = ({ tasks }: Props) => {
  return (
    <Table
      className={css.base}
      columns={columns}
      dataSource={tasks}
      loading={tasks === undefined}
      rowClassName={(record): string => canBeOpened(record) ? linkCss.base : ''}
      rowKey="id"
      size="small"
      onRow={tableRowClickHandler} />
  );
};

export default TaskTable;
