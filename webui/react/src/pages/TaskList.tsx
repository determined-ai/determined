import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Input, Modal, Table } from 'antd';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useMemo, useState } from 'react';

import Grid from 'components/Grid';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Page from 'components/Page';
import { Indicator } from 'components/Spinner';
import {
  defaultRowClassName, getPaginationConfig, isAlternativeAction, MINIMUM_PAGE_SIZE,
} from 'components/Table';
import { TaskRenderer } from 'components/Table';
import TableBatch from 'components/TableBatch';
import TaskActionDropdown from 'components/TaskActionDropdown';
import TaskFilter from 'components/TaskFilter';
import Auth from 'contexts/Auth';
import { Commands, Notebooks, Shells, Tensorboards } from 'contexts/Commands';
import Users from 'contexts/Users';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import useRestApi from 'hooks/useRestApi';
import useStorage from 'hooks/useStorage';
import { openBlank } from 'routes/utils';
import {
  getCommands, getNotebooks, getShells, getTensorboards, killCommand,
} from 'services/api';
import { ApiSorter, EmptyParams } from 'services/types';
import { ShirtSize } from 'themes';
import { ALL_VALUE, Command, CommandTask, CommandType, TaskFilters } from 'types';
import { alphanumericSorter, getPath, numericSorter } from 'utils/data';
import { canBeOpened, filterTasks } from 'utils/task';
import { commandToTask, isTaskKillable } from 'utils/types';

import css from './TaskList.module.scss';
import { columns as defaultColumns } from './TaskList.table';

enum TensorBoardSourceType {
  Experiment = 'Experiment',
  Trial = 'Trial',
}

interface TensorBoardSource {
  id: number;
  path: string;
  type: TensorBoardSourceType;
}

interface SourceInfo {
  path: string;
  plural: string;
  sources: TensorBoardSource[];
}

const defaultFilters: TaskFilters<CommandType> = {
  limit: MINIMUM_PAGE_SIZE,
  states: [ ALL_VALUE ],
  types: {
    [CommandType.Command]: false,
    [CommandType.Notebook]: false,
    [CommandType.Shell]: false,
    [CommandType.Tensorboard]: false,
  },
  username: undefined,
};

const defaultSorter: ApiSorter = {
  descend: true,
  key: 'startTime',
};

const STORAGE_PATH = 'task-list';
const STORAGE_FILTERS_KEY = 'filters';
const STORAGE_SORTER_KEY = 'sorter';

const TaskList: React.FC = () => {
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();
  const storage = useStorage(STORAGE_PATH);
  const initFilters = storage.getWithDefault(
    STORAGE_FILTERS_KEY,
    { ...defaultFilters, username: getPath<string>(auth, 'user.username') },
  );
  const [ filters, setFilters ] = useState<TaskFilters<CommandType>>(initFilters);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ sorter, setSorter ] = useState<ApiSorter>(initSorter);
  const [ search, setSearch ] = useState('');
  const [ sourcesModal, setSourcesModal ] = useState<SourceInfo>();
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

  const handleSourceShow = useCallback((info: SourceInfo) => setSourcesModal(info), []);
  const handleSourceDismiss = useCallback(() => setSourcesModal(undefined), []);

  const handleActionComplete = useCallback(() => fetchTasks(), [ fetchTasks ]);

  const columns = useMemo(() => {
    const nameRenderer: TaskRenderer = (_, record) => {
      if (record.type !== CommandType.Tensorboard || !record.misc) return record.name;

      const info = {
        path: '',
        plural: '',
        sources: [] as TensorBoardSource[],
      };
      record.misc.experimentIds.forEach(id => {
        info.sources.push({
          id,
          path: `/experiments/${id}`,
          type: TensorBoardSourceType.Experiment,
        });
      });
      record.misc.trialIds.forEach(id => {
        info.sources.push({
          id,
          path: `/trials/${id}`,
          type: TensorBoardSourceType.Trial,
        });
      });
      info.plural = info.sources.length > 1 ? 's' : '';
      info.sources.sort((a, b) => {
        if (a.type !== b.type) return alphanumericSorter(a.type, b.type);
        return numericSorter(a.id, b.id);
      });

      return <div className={css.sourceName}>
        <span>{record.name}</span>
        <button className="ignoreTableRowClick" onClick={() => handleSourceShow(info)}>
          Show {info.sources.length} Source{info.plural}
        </button>
      </div>;
    };

    const actionRenderer: TaskRenderer = (_, record) => (
      <TaskActionDropdown task={record} onComplete={handleActionComplete} />
    );

    const newColumns = [ ...defaultColumns ].map(column => {
      column.sortOrder = null;
      if (column.key === sorter.key) column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      if (column.key === 'name') column.render = nameRenderer;
      if (column.key === 'action') column.render = actionRenderer;
      return column;
    });

    return newColumns;
  }, [ handleActionComplete, handleSourceShow, sorter ]);

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
    storage.set(STORAGE_FILTERS_KEY, filters);
    setFilters(filters);
  }, [ setFilters, storage ]);

  const handleBatchKill = useCallback(async () => {
    try {
      const promises = selectedTasks
        .filter(task => isTaskKillable(task))
        .map(task => killCommand({ commandId: task.id, commandType: task.type }));
      await Promise.all(promises);

      /*
       * Deselect selected rows since their states may have changed where they
       * are no longer part of the filter criteria.
       */
      setSelectedRowKeys([]);

      // Refetch task list to get updates based on batch action.
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
        all the eligible selected tasks?
      `,
      icon: <ExclamationCircleOutlined />,
      okText: 'Kill',
      onOk: handleBatchKill,
      title: 'Confirm Batch Kill',
    });
  }, [ handleBatchKill ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, sorter) => {
    if (Array.isArray(sorter)) return;

    const { columnKey, order } = sorter as SorterResult<CommandTask>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    storage.set(STORAGE_SORTER_KEY, { descend: order === 'descend', key: columnKey as string });
    setSorter({ descend: order === 'descend', key: columnKey as string });

    storage.set(STORAGE_FILTERS_KEY, { ...filters, limit: tablePagination.pageSize });
  }, [ columns, filters, setSorter, storage ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  const handleTableRow = useCallback((record: CommandTask) => ({
    onClick: (event: React.MouseEvent) => {
      if (isAlternativeAction(event) || !canBeOpened(record)) return;
      openBlank(record.url as string);
    },
  }), []);

  return (
    <Page id="tasks" title="Tasks">
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
        <TableBatch selectedRowCount={selectedRowKeys.length}>
          <Button
            danger
            disabled={!hasKillable}
            type="primary"
            onClick={handleConfirmation}>Kill</Button>
        </TableBatch>
        <Table
          columns={columns}
          dataSource={filteredTasks}
          loading={{
            indicator: <Indicator />,
            spinning: !hasLoaded,
          }}
          pagination={getPaginationConfig(filteredTasks.length, filters.limit)}
          rowClassName={record => defaultRowClassName(canBeOpened(record))}
          rowKey="id"
          rowSelection={{ onChange: handleTableRowSelect, selectedRowKeys }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange}
          onRow={handleTableRow} />
      </div>
      <Modal
        footer={null}
        style={{ minWidth: '60rem' }}
        title={`
          ${sourcesModal?.sources.length}
          TensorBoard Source${sourcesModal?.plural}
        `}
        visible={!!sourcesModal}
        onCancel={handleSourceDismiss}>
        <div className={css.sourceLinks}>
          <Grid gap={ShirtSize.medium} minItemWidth={12}>
            {sourcesModal?.sources.map(source => <Link
              key={source.id}
              path={source.path}>{source.type} {source.id}</Link>)}
          </Grid>
        </div>
      </Modal>
    </Page>
  );
};

export default TaskList;
