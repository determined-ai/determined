import { Button, Input, notification, Table } from 'antd';
import axios from 'axios';
import React, { useCallback, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import { makeClickHandler } from 'components/Link';
import linkCss from 'components/Link.module.scss';
import Page from 'components/Page';
import TableBatch from 'components/TableBatch';
import TaskFilter from 'components/TaskFilter';
import Auth from 'contexts/Auth';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Users from 'contexts/Users';
import useStorage from 'hooks/useStorage';
import { killCommand } from 'services/api';
import { ALL_VALUE, CommandTask, CommandType, TaskFilters } from 'types';
import { canBeOpened, filterTasks } from 'utils/task';
import { commandToTask, isTaskKillable } from 'utils/types';

import css from './TaskList.module.scss';
import { columns } from './TaskList.table';

const defaultFilters: TaskFilters<CommandType> = {
  limit: 25,
  states: [ ALL_VALUE ],
  types: {
    [CommandType.Command]: false,
    [CommandType.Notebook]: false,
    [CommandType.Shell]: false,
    [CommandType.Tensorboard]: false,
  },
  username: undefined,
};

const TaskList: React.FC = () => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();
  const storage = useStorage('task-list');
  const initFilters = storage.getWithDefault('filters',
    { ...defaultFilters, username: (auth.user || {}).username });
  const [ filters, setFilters ] = useState<TaskFilters<CommandType>>(initFilters);
  const [ search, setSearch ] = useState('');
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);

  const sources = [
    commands,
    notebooks,
    shells,
    tensorboards,
  ];

  const loadedTasks = sources
    .filter(src => src.data !== undefined)
    .map(src => src.data || [])
    .reduce((acc, cur) => [ ...acc, ...cur ], [])
    .map(commandToTask);

  const hasLoaded = sources.find(src => src.hasLoaded);

  const filteredTasks = useMemo(() => {
    return filterTasks(loadedTasks, filters, users.data || [], search);
  }, [ filters, loadedTasks, search, users.data ]);

  const showBatch = selectedRowKeys.length !== 0;

  const taskMap = useMemo(() => {
    return (loadedTasks || []).reduce((acc, task) => {
      acc[task.id] = task;
      return acc;
    }, {} as Record<string, CommandTask>);
  }, [ loadedTasks ]);

  const selectedTasks = useMemo(() => {
    return selectedRowKeys.map(key => taskMap[key]);
  }, [ selectedRowKeys, taskMap ]);

  const hasKillable = useMemo(() => {
    for (let i = 0; i < selectedTasks.length; i++) {
      if (isTaskKillable(selectedTasks[i])) return true;
    }
    return false;
  }, [ selectedTasks ]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleFilterChange = useCallback((filters: TaskFilters<CommandType>): void => {
    storage.set('filters', filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

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

  const handleTableRow = useCallback((record: CommandTask) => ({
    onClick: canBeOpened(record) ? makeClickHandler(record.url as string) : undefined,
  }), []);

  // TODO select and batch operation:
  // https://ant.design/components/table/#components-table-demo-row-selection-and-operation
  return (
    <Page title="Tasks">
      <div className={css.base}>
        <div className={css.header}>
          <Input
            allowClear
            className={css.search}
            placeholder="ID or name"
            prefix={<Icon name="search" size="small" />}
            onChange={handleSearchChange} />
          <TaskFilter<CommandType>
            filters={filters}
            showExperiments={false}
            showLimit={false}
            onChange={handleFilterChange} />
        </div>
        <TableBatch message="Apply batch operations to multiple tasks." show={showBatch}>
          <Button
            danger
            disabled={!hasKillable}
            type="primary"
            onClick={handleBatchKill}>Kill</Button>
        </TableBatch>
        <Table
          columns={columns}
          dataSource={filteredTasks}
          loading={!hasLoaded}
          rowClassName={(record): string => canBeOpened(record) ? linkCss.base : ''}
          rowKey="id"
          rowSelection={{ onChange: handleTableRowSelect, selectedRowKeys }}
          size="small"
          onRow={handleTableRow} />
      </div>
    </Page>
  );
};

export default TaskList;
