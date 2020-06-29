import { Button, notification, Table, Tooltip } from 'antd';
import { ColumnsType, ColumnType } from 'antd/lib/table';
import axios from 'axios';
import React, { MouseEventHandler, useCallback, useMemo, useState } from 'react';

import { killCommand } from 'services/api';
import { AnyTask, CommandTask, CommandType, CommonProps } from 'types';
import { alphanumericSorter } from 'utils/data';
import { canBeOpened } from 'utils/task';
import { commandTypeToLabel, isTaskKillable } from 'utils/types';

import Badge, { BadgeType } from './Badge';
import Icon from './Icon';
import { makeClickHandler } from './Link';
import linkCss from './Link.module.scss';
import {
  actionsColumn, ellipsisRenderer, Renderer, startTimeColumn, stateColumn, userColumn,
} from './Table';
import css from './Table.module.scss';
import TableBatch from './TableBatch';

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
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);

  const showBatch = selectedRowKeys.length !== 0;

  const taskMap = useMemo(() => {
    return (tasks || []).reduce((acc, task) => {
      acc[task.id] = task;
      return acc;
    }, {} as Record<string, CommandTask>);
  }, [ tasks ]);

  const selectedTasks = useMemo(() => {
    return selectedRowKeys.map(key => taskMap[key]);
  }, [ selectedRowKeys, taskMap ]);

  const hasKillable = useMemo(() => {
    for (let i = 0; i < selectedTasks.length; i++) {
      if (isTaskKillable(selectedTasks[i])) return true;
    }
    return false;
  }, [ selectedTasks ]);

  const handleBatchKill = useCallback(async () => {
    try {
      const source = axios.CancelToken.source();
      const promises = selectedTasks.map(task => killCommand({
        cancelToken: source.token,
        commandId: task.id,
        commandType: task.type,
      }));
      await Promise.all(promises);
    } catch (e) {
      notification.warn({
        description: 'Please try again later.',
        message: 'Unable to Kill Selected Tasks',
      });
    }
  }, [ selectedTasks ]);
  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  return (
    <div>
      <TableBatch message="Apply batch operations to multiple tasks." show={showBatch}>
        <Button
          danger
          disabled={!hasKillable}
          type="primary"
          onClick={handleBatchKill}>Kill</Button>
      </TableBatch>
      <Table
        className={css.base}
        columns={columns}
        dataSource={tasks}
        loading={tasks === undefined}
        rowClassName={(record): string => canBeOpened(record) ? linkCss.base : ''}
        rowKey="id"
        rowSelection={{
          onChange: handleTableRowSelect,
          selectedRowKeys,
        }}
        size="small"
        onRow={tableRowClickHandler} />
    </div>
  );
};

export default TaskTable;
