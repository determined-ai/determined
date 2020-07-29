import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Input, Modal, Table } from 'antd';
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
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import useRestApi from 'hooks/useRestApi';
import useStorage from 'hooks/useStorage';
import { getCommands, getNotebooks, getShells, getTensorboards, killCommand } from 'services/api';
import { EmptyParams } from 'services/types';
import { ALL_VALUE, Command, CommandTask, CommandType, TaskFilters } from 'types';
import { getPath } from 'utils/data';
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
  const initFilters = storage.getWithDefault('filters', {
    ...defaultFilters, username: getPath<string>(auth, 'user.username'),
  });
  const [ filters, setFilters ] = useState<TaskFilters<CommandType>>(initFilters);
  const [ search, setSearch ] = useState('');
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);
  const [ commandsResponse, triggerCommandsRequest ] =
    useRestApi<EmptyParams, Command[]>(getCommands, {});
  const [ notebooksResponse, triggerNotebooksRequest ] =
    useRestApi<EmptyParams, Command[]>(getNotebooks, {});
  const [ shellsResponse, triggerShellsRequest ] =
    useRestApi<EmptyParams, Command[]>(getShells, {});
  const [ tensorboardsResponse, triggerTensorboardsRequest ] =
    useRestApi<EmptyParams, Command[]>(getTensorboards, {});

  const sources = [ commands, notebooks, shells, tensorboards ];

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

  const fetchTasks = useCallback((): void => {
    triggerCommandsRequest({});
    triggerNotebooksRequest({});
    triggerShellsRequest({});
    triggerTensorboardsRequest({});
  }, [
    triggerCommandsRequest,
    triggerNotebooksRequest,
    triggerShellsRequest,
    triggerTensorboardsRequest,
  ]);

  /*
   * Check once every second to see if all task endpoints have resolved.
   * Consider it a failure if the number of checks exceed 10 times.
   */
  const checkForTaskListUpdate = useCallback((): Promise<void> => {
    return new Promise((resolve, reject) => {
      let counter = 0;
      const timer = setInterval(() => {
        if (!commandsResponse.isLoading && !notebooksResponse.isLoading &&
            !shellsResponse.isLoading && !tensorboardsResponse.isLoading) {
          clearInterval(timer);
          resolve();
        } else if (counter > 10) reject();
        counter++;
      }, 1000);
    });
  }, [
    commandsResponse.isLoading,
    notebooksResponse.isLoading,
    shellsResponse.isLoading,
    tensorboardsResponse.isLoading,
  ]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleFilterChange = useCallback((filters: TaskFilters<CommandType>): void => {
    storage.set('filters', filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

  const handleBatchKill = useCallback(async () => {
    try {
      const promises = selectedTasks
        .filter(task => isTaskKillable(task))
        .map(task => killCommand({ commandId: task.id, commandType: task.type }));
      await Promise.all(promises);
      fetchTasks();
      await checkForTaskListUpdate();
    } catch (e) {
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to Kill Selected Tasks',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ checkForTaskListUpdate, fetchTasks, selectedTasks ]);

  const handleConfirmation = useCallback(() => {
    Modal.confirm({
      content: `
        Are you sure you want to kill
        all the eligible selected experiments?
      `,
      icon: <ExclamationCircleOutlined />,
      okText: 'Kill',
      onOk: handleBatchKill,
      title: 'Confirm Batch Kill',
    });
  }, [ handleBatchKill ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  const handleTableRow = useCallback((record: CommandTask) => ({
    onClick: canBeOpened(record) ? makeClickHandler(record.url as string) : undefined,
  }), []);

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
            onClick={handleConfirmation}>Kill</Button>
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
