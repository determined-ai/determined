import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Modal } from 'antd';
import { ColumnType, FilterDropdownProps, SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Grid from 'components/Grid';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import tableCss from 'components/ResponsiveTable.module.scss';
import {
  defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE, relativeTimeRenderer,
  stateRenderer, taskIdRenderer, taskNameRenderer, taskTypeRenderer, userRenderer,
} from 'components/Table';
import { TaskRenderer } from 'components/Table';
import TableBatch from 'components/TableBatch';
import TableFilterDropdown from 'components/TableFilterDropdown';
import TableFilterSearch from 'components/TableFilterSearch';
import TaskActionDropdown from 'components/TaskActionDropdown';
import { useStore } from 'contexts/Store';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { paths } from 'routes/utils';
import { getCommands, getNotebooks, getShells, getTensorboards, killTask } from 'services/api';
import { ApiSorter } from 'services/types';
import { ShirtSize } from 'themes';
import { CommandState, CommandTask, CommandType, TaskFilters } from 'types';
import { isEqual } from 'utils/data';
import {
  alphanumericSorter, commandStateSorter, numericSorter, stringTimeSorter,
} from 'utils/sort';
import { capitalize } from 'utils/string';
import { filterTasks } from 'utils/task';
import { commandToTask, isTaskKillable } from 'utils/types';

import css from './TaskList.module.scss';

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
  states: undefined,
  types: undefined,
  users: undefined,
};

const defaultSorter: ApiSorter = {
  descend: true,
  key: 'startTime',
};

const STORAGE_PATH = 'task-list';
const STORAGE_FILTERS_KEY = 'filters';
const STORAGE_SORTER_KEY = 'sorter';

const TaskList: React.FC = () => {
  const { auth, users } = useStore();
  const storage = useStorage(STORAGE_PATH);
  const initFilters = storage.getWithDefault(
    STORAGE_FILTERS_KEY,
    {
      ...defaultFilters,
      users: (!auth.user || auth.user?.isAdmin) ? defaultFilters.users : [ auth.user?.username ],
    },
  );
  const [ canceler ] = useState(new AbortController());
  const [ tasks, setTasks ] = useState<CommandTask[]>([]);
  const [ filters, setFilters ] = useState<TaskFilters<CommandType>>(initFilters);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ sorter, setSorter ] = useState<ApiSorter>(initSorter);
  const [ search, setSearch ] = useState('');
  const [ sourcesModal, setSourcesModal ] = useState<SourceInfo>();
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);

  const fetchUsers = useFetchUsers(canceler);

  const loadedTasks = tasks.map(commandToTask);

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

  const fetchTasks = useCallback(async () => {
    try {
      const [ commands, notebooks, shells, tensorboards ] = await Promise.all([
        getCommands({ signal: canceler.signal }),
        getNotebooks({ signal: canceler.signal }),
        getShells({ signal: canceler.signal }),
        getTensorboards({ signal: canceler.signal }),
      ]);
      const newTasks = [ ...commands, ...notebooks, ...shells, ...tensorboards ];
      setTasks(prev => {
        if (isEqual(prev, newTasks)) return prev;
        return newTasks;
      });
    } catch (e) {
      handleError({ message: 'Unable to fetch tasks.', silent: true, type: ErrorType.Api });
    }
  }, [ canceler ]);

  const fetchAll = useCallback((): void => {
    fetchUsers();
    fetchTasks();
  }, [ fetchTasks, fetchUsers ]);

  const handleSourceShow = useCallback((info: SourceInfo) => setSourcesModal(info), []);
  const handleSourceDismiss = useCallback(() => setSourcesModal(undefined), []);

  const handleActionComplete = useCallback(() => fetchAll(), [ fetchAll ]);

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" />, []);

  const handleNameSearchApply = useCallback((newSearch: string) => {
    setSearch(newSearch);
  }, []);

  const handleNameSearchReset = useCallback(() => {
    setSearch('');
  }, []);

  const nameFilterSearch = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterSearch
      {...filterProps}
      value={search}
      onReset={handleNameSearchReset}
      onSearch={handleNameSearchApply}
    />
  ), [ handleNameSearchApply, handleNameSearchReset, search ]);

  const updateFilters = useCallback((filters: TaskFilters<CommandType>): void => {
    storage.set(STORAGE_FILTERS_KEY, filters);
    setSelectedRowKeys([]);
    setFilters(filters);
  }, [ setFilters, storage ]);

  const handleTypeFilterApply = useCallback((types: string[]) => {
    updateFilters({ ...filters, types: types.length !== 0 ? types as CommandType[] : undefined });
  }, [ filters, updateFilters ]);

  const handleTypeFilterReset = useCallback(() => {
    updateFilters({ ...filters, types: undefined });
  }, [ filters, updateFilters ]);

  const typeFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      values={filters.types}
      width={180}
      onFilter={handleTypeFilterApply}
      onReset={handleTypeFilterReset} />
  ), [ filters.types, handleTypeFilterApply, handleTypeFilterReset ]);

  const handleStateFilterApply = useCallback((states: string[]) => {
    updateFilters({ ...filters, states: states.length !== 0 ? states : undefined });
  }, [ filters, updateFilters ]);

  const handleStateFilterReset = useCallback(() => {
    updateFilters({ ...filters, states: undefined });
  }, [ filters, updateFilters ]);

  const stateFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      values={filters.states}
      onFilter={handleStateFilterApply}
      onReset={handleStateFilterReset} />
  ), [ filters.states, handleStateFilterApply, handleStateFilterReset ]);

  const handleUserFilterApply = useCallback((users: string[]) => {
    updateFilters({ ...filters, users: users.length !== 0 ? users : undefined });
  }, [ filters, updateFilters ]);

  const handleUserFilterReset = useCallback(() => {
    updateFilters({ ...filters, users: undefined });
  }, [ filters, updateFilters ]);

  const userFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      searchable
      values={filters.users}
      onFilter={handleUserFilterApply}
      onReset={handleUserFilterReset} />
  ), [ filters.users, handleUserFilterApply, handleUserFilterReset ]);

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

    const tableColumns: ColumnType<CommandTask>[] = [
      {
        dataIndex: 'id',
        key: 'id',
        render: taskIdRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.id, b.id),
        title: 'Short ID',
      },
      {
        filterDropdown: typeFilterDropdown,
        filters: Object.values(CommandType).map(value => ({
          text: (
            <div className={css.typeFilter}>
              <Icon name={value.toLocaleLowerCase()} />
              <span>{capitalize(value)}</span>
            </div>
          ),
          value,
        })),
        key: 'type',
        onHeaderCell: () => filters.types ? { className: tableCss.headerFilterOn } : {},
        render: taskTypeRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.type, b.type),
        title: 'Type',
      },
      {
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        key: 'name',
        onHeaderCell: () => search !== '' ? { className: tableCss.headerFilterOn } : {},
        render: nameNSourceRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.name, b.name),
        title: 'Name',
      },
      {
        key: 'startTime',
        render: (_: number, record: CommandTask): React.ReactNode => {
          return relativeTimeRenderer(new Date(record.startTime));
        },
        sorter: (a: CommandTask, b: CommandTask): number => {
          return stringTimeSorter(a.startTime, b.startTime);
        },
        title: 'Start Time',
      },
      {
        filterDropdown: stateFilterDropdown,
        filters: Object.values(CommandState)
          .map((value) => ({
            text: <Badge state={value} type={BadgeType.State} />,
            value,
          })),
        key: 'state',
        onHeaderCell: () => filters.states ? { className: tableCss.headerFilterOn } : {},
        render: stateRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => commandStateSorter(a.state, b.state),
        title: 'State',
      },
      {
        dataIndex: 'resourcePool',
        key: 'resourcePool',
        sorter: true,
        title: 'Resource Pool',
      },
      {
        filterDropdown: userFilterDropdown,
        filters: users.map(user => ({ text: user.username, value: user.username })),
        key: 'user',
        onHeaderCell: () => filters.users ? { className: tableCss.headerFilterOn } : {},
        render: userRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => {
          return alphanumericSorter(a.username, b.username);
        },
        title: 'User',
      },
      {
        align: 'right',
        className: 'fullCell',
        key: 'action',
        render: actionRenderer,
        title: '',
      },
    ];

    return tableColumns.map(column => {
      column.sortOrder = null;
      if (column.key === sorter.key) column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      return column;
    });
  }, [
    filters,
    handleActionComplete,
    handleSourceShow,
    nameFilterSearch,
    stateFilterDropdown,
    search,
    sorter,
    tableSearchIcon,
    typeFilterDropdown,
    userFilterDropdown,
    users,
  ]);

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

    const updatedSorter = { descend: order === 'descend', key: columnKey as string };
    storage.set(STORAGE_SORTER_KEY, updatedSorter);
    setSorter(updatedSorter);

    const updatedFilters = { ...filters, limit: tablePagination.pageSize };
    storage.set(STORAGE_FILTERS_KEY, updatedFilters);
    setFilters(updatedFilters);
  }, [ columns, filters, setSorter, storage ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  usePolling(fetchAll);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <Page id="tasks" title="Tasks">
      <div className={css.base}>
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
          loading={!hasLoaded}
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
        style={{ minWidth: '600px' }}
        title={`
          ${sourcesModal?.sources.length}
          TensorBoard Source${sourcesModal?.plural}
        `}
        visible={!!sourcesModal}
        onCancel={handleSourceDismiss}>
        <div className={css.sourceLinks}>
          <Grid gap={ShirtSize.medium} minItemWidth={120}>
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
