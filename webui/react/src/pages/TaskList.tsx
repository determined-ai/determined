import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Input, Modal } from 'antd';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Grid from 'components/Grid';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import { Indicator } from 'components/Spinner';
import {
  defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE, taskNameRenderer,
} from 'components/Table';
import { TaskRenderer } from 'components/Table';
import TableBatch from 'components/TableBatch';
import TaskActionDropdown from 'components/TaskActionDropdown';
import TaskFilter from 'components/TaskFilter';
import Auth from 'contexts/Auth';
import {
  Commands, Notebooks, Shells, Tensorboards, useFetchCommands, useFetchNotebooks, useFetchShells,
  useFetchTensorboards,
} from 'contexts/Commands';
import Users, { useFetchUsers } from 'contexts/Users';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { paths } from 'routes/utils';
import { killTask } from 'services/api';
import { ApiSorter } from 'services/types';
import { ShirtSize } from 'themes';
import { ALL_VALUE, CommandTask, CommandType, TaskFilters } from 'types';
import { alphanumericSorter, numericSorter } from 'utils/sort';
import { filterTasks } from 'utils/task';
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
  const [ canceler ] = useState(new AbortController());
  const auth = Auth.useStateContext();
  const users = Users.useStateContext();
  const commands = Commands.useStateContext();
  const notebooks = Notebooks.useStateContext();
  const shells = Shells.useStateContext();
  const tensorboards = Tensorboards.useStateContext();
  const storage = useStorage(STORAGE_PATH);
  const initFilters = storage.getWithDefault(
    STORAGE_FILTERS_KEY,
    (!auth.user || auth.user?.isAdmin) ? defaultFilters : {
      ...defaultFilters,
      username: auth.user?.username,
    },
  );
  const [ filters, setFilters ] = useState<TaskFilters<CommandType>>(initFilters);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ sorter, setSorter ] = useState<ApiSorter>(initSorter);
  const [ search, setSearch ] = useState('');
  const [ sourcesModal, setSourcesModal ] = useState<SourceInfo>();
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);

  const fetchUsers = useFetchUsers(canceler);
  const fetchCommands = useFetchCommands(canceler);
  const fetchNotebooks = useFetchNotebooks(canceler);
  const fetchShells = useFetchShells(canceler);
  const fetchTensorboards = useFetchTensorboards(canceler);

  const tasks = [ commands, notebooks, shells, tensorboards ];

  const loadedTasks = tasks
    .map(src => src || [])
    .reduce((acc, src) => [ ...acc, ...src ], [])
    .map(commandToTask);

  const hasLoaded = tasks.reduce((acc, src) => acc && !!src, true);

  const filteredTasks = useMemo(() => {
    return filterTasks(loadedTasks, filters, users || [], search);
  }, [ filters, loadedTasks, search, users ]);

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

  const fetchAll = useCallback((): void => {
    fetchUsers();
    fetchCommands();
    fetchNotebooks();
    fetchShells();
    fetchTensorboards();
  }, [
    fetchUsers,
    fetchCommands,
    fetchNotebooks,
    fetchShells,
    fetchTensorboards,
  ]);

  const handleSourceShow = useCallback((info: SourceInfo) => setSourcesModal(info), []);
  const handleSourceDismiss = useCallback(() => setSourcesModal(undefined), []);

  const handleActionComplete = useCallback(() => fetchAll(), [ fetchAll ]);

  const columns = useMemo(() => {

    const nameNSourceRenderer: TaskRenderer = (_, record, index) => {
      if (record.type !== CommandType.Tensorboard || !record.misc) {
        return taskNameRenderer(_, record, index);
      }

      const info = {
        path: '',
        plural: '',
        sources: [] as TensorBoardSource[],
      };
      record.misc.experimentIds.forEach(id => {
        info.sources.push({
          id,
          path: paths.experimentDetails(id),
          type: TensorBoardSourceType.Experiment,
        });
      });
      record.misc.trialIds.forEach(id => {
        info.sources.push({
          id,
          path: paths.trialDetails(id),
          type: TensorBoardSourceType.Trial,
        });
      });
      info.plural = info.sources.length > 1 ? 's' : '';
      info.sources.sort((a, b) => {
        if (a.type !== b.type) return alphanumericSorter(a.type, b.type);
        return numericSorter(a.id, b.id);
      });

      return <div className={css.sourceName}>
        {taskNameRenderer(_, record, index)}
        <button className="ignoreTableRowClick" onClick={() => handleSourceShow(info)}>
          Show {info.sources.length} Source{info.plural}
        </button>
      </div>;
    };

    const actionRenderer: TaskRenderer = (_, record) => (
      <TaskActionDropdown task={record} onComplete={handleActionComplete} />
    );

    return [ ...defaultColumns ].map(column => {
      column.sortOrder = null;
      if (column.key === sorter.key) column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      if (column.key === 'name') column.render = nameNSourceRenderer;
      if (column.key === 'action') column.render = actionRenderer;
      return column;
    });
  }, [ handleActionComplete, handleSourceShow, sorter ]);

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
        .map(task => killTask(task));
      await Promise.all(promises);

      /*
       * Deselect selected rows since their states may have changed where they
       * are no longer part of the filter criteria.
       */
      setSelectedRowKeys([]);

      // Refetch task list to get updates based on batch action.
      fetchAll();
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
  }, [ fetchAll, selectedTasks ]);

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

  usePolling(fetchAll);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

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
        <ResponsiveTable<CommandTask>
          columns={columns}
          dataSource={filteredTasks}
          loading={{
            indicator: <Indicator />,
            spinning: !hasLoaded,
          }}
          pagination={getPaginationConfig(filteredTasks.length, filters.limit)}
          rowClassName={() => defaultRowClassName({ clickable: false })}
          rowKey="id"
          rowSelection={{ onChange: handleTableRowSelect, selectedRowKeys }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange} />
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
