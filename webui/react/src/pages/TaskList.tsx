import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Modal } from 'antd';
import { ColumnType, FilterDropdownProps, SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import FilterCounter from 'components/FilterCounter';
import Grid from 'components/Grid';
import Icon from 'components/Icon';
import Link from 'components/Link';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import tableCss from 'components/ResponsiveTable.module.scss';
import {
  defaultRowClassName, getFullPaginationConfig, MINIMUM_PAGE_SIZE, relativeTimeRenderer,
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
import { parseUrl } from 'routes/utils';
import { getCommands, getNotebooks, getShells, getTensorboards, killTask } from 'services/api';
import { ApiSorter } from 'services/types';
import { ShirtSize } from 'themes';
import { CommandState, CommandTask, CommandType, Pagination, TaskFilters } from 'types';
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

const URL_ALL = 'all';

const STORAGE_PATH = 'task-list';
const STORAGE_FILTERS_KEY = 'filters';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const TaskList: React.FC = () => {
  const { users } = useStore();
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initFilters = storage.getWithDefault(STORAGE_FILTERS_KEY, { ...defaultFilters });
  const [ isUrlParsed, setIsUrlParsed ] = useState(false);
  const [ canceler ] = useState(new AbortController());
  const [ tasks, setTasks ] = useState<CommandTask[] | undefined>(undefined);
  const [ filters, setFilters ] = useState<TaskFilters<CommandType>>(initFilters);
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ pagination, setPagination ] = useState<Pagination>(
    { limit: initLimit, offset: 0 },
  );
  const [ sorter, setSorter ] = useState<ApiSorter>(initSorter);
  const [ search, setSearch ] = useState('');
  const [ sourcesModal, setSourcesModal ] = useState<SourceInfo>();
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);

  const fetchUsers = useFetchUsers(canceler);

  const loadedTasks = useMemo(() => tasks?.map(commandToTask) || [], [ tasks ]);

  /*
   * When filters changes update the page URL.
   */
  useEffect(() => {
    if (!isUrlParsed) return;

    const searchParams = new URLSearchParams;
    const url = parseUrl(window.location.href);

    // limit
    searchParams.append('limit', pagination.limit.toString());

    // offset
    searchParams.append('offset', pagination.offset.toString());

    // search
    searchParams.append('search', search);

    // sortDesc
    searchParams.append('sortDesc', sorter.descend ? '1' : '0');

    // sortKey
    searchParams.append('sortKey', (sorter.key || '') as string);

    // type
    if (filters.types && filters.types.length > 0) {
      filters.types.forEach(type => searchParams.append('type', type));
    } else {
      searchParams.append('type', URL_ALL);
    }

    // states
    if (filters.states && filters.states.length > 0) {
      filters.states.forEach(state => searchParams.append('state', state));
    } else {
      searchParams.append('state', URL_ALL);
    }

    // users
    if (filters.users && filters.users.length > 0) {
      filters.users.forEach(user => searchParams.append('user', user));
    } else {
      searchParams.append('user', URL_ALL);
    }

    window.history.pushState(
      {},
      '',
      url.origin + url.pathname + '?' + searchParams.toString(),
    );
  }, [ filters, isUrlParsed, pagination, search, sorter ]);

  /*
   * On first load: if filters are specified in URL, override default.
   */
  useEffect(() => {
    if (isUrlParsed) return;

    // If search params are not set, we default to user preferences
    const url = parseUrl(window.location.href);
    if (url.search === '') {
      setIsUrlParsed(true);
      return;
    }

    const urlSearchParams = url.searchParams;

    // limit
    const limit = urlSearchParams.get('limit');
    if (limit != null && !isNaN(parseInt(limit))) {
      pagination.limit = parseInt(limit);
    }

    // offset
    const offset = urlSearchParams.get('offset');
    if (offset != null && !isNaN(parseInt(offset))) {
      pagination.offset = parseInt(offset);
    }

    // search
    const search = urlSearchParams.get('search');
    if (search != null) {
      setSearch(search);
    }

    // sortDesc
    const sortDesc = urlSearchParams.get('sortDesc');
    if (sortDesc != null) {
      sorter.descend = (sortDesc === '1');
    }

    // sortKey
    const sortKey = urlSearchParams.get('sortKey');
    if (sortKey != null &&
      [ 'id', 'type', 'name', 'startTime', 'state', 'resourcePool', 'username' ]
        .includes(sortKey)) {
      sorter.key = sortKey as keyof CommandTask;
    }

    // types
    const type = urlSearchParams.getAll('type');
    if (type != null) {
      filters.types = (type.includes(URL_ALL) ? undefined : type as CommandType[]);
    }

    // states
    const state = urlSearchParams.getAll('state');
    if (state != null) {
      filters.states = (state.includes(URL_ALL) ? undefined : state);
    }

    // users
    const user = urlSearchParams.getAll('user');
    if (user != null) {
      filters.users = (user.includes(URL_ALL) ? undefined : user);
    }

    setFilters(filters);
    setIsUrlParsed(true);
    setPagination(pagination);
    setSorter(sorter);
  }, [ filters, isUrlParsed, pagination, search, sorter ]);

  useEffect(() => {
    const limit = pagination.limit;
    setFilters(filters => { return { ...filters, limit }; });
  }, [ pagination.limit ]);

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

  const resetFilters = useCallback(() => {
    updateFilters({ ...defaultFilters, limit: filters.limit });
  }, [ updateFilters, filters ]);

  const activeFilterCount = useMemo(() => {
    let count = 0;
    const filtersToIgnore = new Set([ 'limit' ]);
    const isInactive = (x: unknown) => x === undefined;
    Object.entries(filters).forEach(([ key, value ]) => {
      if (filtersToIgnore.has(key)) return;
      if (!isInactive(value)) count++;
    });
    return count;
  }, [ filters ]);

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

    const updatedPagination = {
      limit: tablePagination.pageSize,
      offset: (tablePagination.current - 1) * tablePagination.pageSize,
    };
    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPagination(updatedPagination);
  }, [ columns, setSorter, storage ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  usePolling(fetchAll);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const clearSelected = useCallback(() => {
    setSelectedRowKeys([]);
  }, []);

  return (
    <Page
      id="tasks"
      options={<FilterCounter activeFilterCount={activeFilterCount} onReset={resetFilters} /> }
      title="Tasks">
      <div className={css.base}>
        <TableBatch selectedRowCount={selectedRowKeys.length} onClear={clearSelected}>
          <Button
            danger
            disabled={!hasKillable}
            type="primary"
            onClick={handleConfirmation}>Kill</Button>
        </TableBatch>
        <ResponsiveTable<CommandTask>
          columns={columns}
          dataSource={filteredTasks}
          loading={tasks === undefined}
          pagination={getFullPaginationConfig(pagination, filteredTasks.length)}
          rowClassName={() => defaultRowClassName({ clickable: false })}
          rowKey="id"
          rowSelection={{
            onChange: handleTableRowSelect,
            preserveSelectedRowKeys: true,
            selectedRowKeys,
          }}
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
