import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
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
  defaultRowClassName, getFullPaginationConfig, relativeTimeRenderer,
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
import useSettings from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getCommands, getNotebooks, getShells, getTensorboards, killTask } from 'services/api';
import { ShirtSize } from 'themes';
import { ExperimentAction as Action, CommandState, CommandTask, CommandType } from 'types';
import { isEqual } from 'utils/data';
import {
  alphanumericSorter, commandStateSorter, numericSorter, stringTimeSorter,
} from 'utils/sort';
import { capitalize } from 'utils/string';
import { filterTasks } from 'utils/task';
import { commandToTask, isTaskKillable } from 'utils/types';

import css from './TaskList.module.scss';
import settingsConfig, { Settings } from './TaskList.settings';

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

const filterKeys: Array<keyof Settings> = [ 'search', 'state', 'type', 'user' ];

const TaskList: React.FC = () => {
  const { users } = useStore();
  const [ canceler ] = useState(new AbortController());
  const [ tasks, setTasks ] = useState<CommandTask[] | undefined>(undefined);
  const [ sourcesModal, setSourcesModal ] = useState<SourceInfo>();

  const {
    activeSettings,
    resetSettings,
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig);

  const fetchUsers = useFetchUsers(canceler);

  const loadedTasks = useMemo(() => tasks?.map(commandToTask) || [], [ tasks ]);

  const filteredTasks = useMemo(() => {
    return filterTasks<CommandType, CommandTask>(
      loadedTasks,
      {
        limit: settings.tableLimit,
        states: settings.state,
        types: settings.type as CommandType[],
        users: settings.user,
      },
      users || [],
      settings.search,
    );
  }, [ loadedTasks, settings, users ]);

  const taskMap = useMemo(() => {
    return (loadedTasks || []).reduce((acc, task) => {
      acc[task.id] = task;
      return acc;
    }, {} as Record<string, CommandTask>);
  }, [ loadedTasks ]);

  const selectedTasks = useMemo(() => {
    return (settings.row || []).map(id => taskMap[id]).filter(task => !!task);
  }, [ settings.row, taskMap ]);

  const hasKillable = useMemo(() => {
    for (const task of selectedTasks) {
      if (isTaskKillable(task)) return true;
    }
    return false;
  }, [ selectedTasks ]);

  const filterCount = useMemo(() => activeSettings(filterKeys).length, [ activeSettings ]);

  const resetFilters = useCallback(() => {
    resetSettings([ ...filterKeys, 'tableOffset' ]);
  }, [ resetSettings ]);

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
    updateSettings({ row: undefined, search: newSearch || undefined });
  }, [ updateSettings ]);

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ row: undefined, search: undefined });
  }, [ updateSettings ]);

  const nameFilterSearch = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterSearch
      {...filterProps}
      value={settings.search || ''}
      onReset={handleNameSearchReset}
      onSearch={handleNameSearchApply}
    />
  ), [ handleNameSearchApply, handleNameSearchReset, settings.search ]);

  const handleTypeFilterApply = useCallback((types: string[]) => {
    updateSettings({
      row: undefined,
      type: types.length !== 0 ? types as CommandType[] : undefined,
    });
  }, [ updateSettings ]);

  const handleTypeFilterReset = useCallback(() => {
    updateSettings({ row: undefined, type: undefined });
  }, [ updateSettings ]);

  const typeFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      values={settings.type}
      width={180}
      onFilter={handleTypeFilterApply}
      onReset={handleTypeFilterReset} />
  ), [ handleTypeFilterApply, handleTypeFilterReset, settings.type ]);

  const handleStateFilterApply = useCallback((states: string[]) => {
    updateSettings({
      row: undefined,
      state: states.length !== 0 ? states as CommandState[] : undefined,
    });
  }, [ updateSettings ]);

  const handleStateFilterReset = useCallback(() => {
    updateSettings({ row: undefined, state: undefined });
  }, [ updateSettings ]);

  const stateFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      values={settings.state}
      onFilter={handleStateFilterApply}
      onReset={handleStateFilterReset} />
  ), [ handleStateFilterApply, handleStateFilterReset, settings.state ]);

  const handleUserFilterApply = useCallback((users: string[]) => {
    updateSettings({
      row: undefined,
      user: users.length !== 0 ? users : undefined,
    });
  }, [ updateSettings ]);

  const handleUserFilterReset = useCallback(() => {
    updateSettings({ row: undefined, user: undefined });
  }, [ updateSettings ]);

  const userFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      searchable
      values={settings.user}
      onFilter={handleUserFilterApply}
      onReset={handleUserFilterReset} />
  ), [ handleUserFilterApply, handleUserFilterReset, settings.user ]);

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
        onHeaderCell: () => settings.type ? { className: tableCss.headerFilterOn } : {},
        render: taskTypeRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => alphanumericSorter(a.type, b.type),
        title: 'Type',
      },
      {
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        key: 'name',
        onHeaderCell: () => settings.search ? { className: tableCss.headerFilterOn } : {},
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
        onHeaderCell: () => settings.state ? { className: tableCss.headerFilterOn } : {},
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
        onHeaderCell: () => settings.user ? { className: tableCss.headerFilterOn } : {},
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
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });
  }, [
    handleActionComplete,
    handleSourceShow,
    nameFilterSearch,
    stateFilterDropdown,
    settings,
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
      updateSettings({ row: undefined });

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
  }, [ fetchAll, selectedTasks, updateSettings ]);

  const showConfirmation = useCallback(() => {
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

  const handleBatchAction = useCallback((action?: string) => {
    if (action === Action.Kill) showConfirmation();
  }, [ showConfirmation ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<CommandTask>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    const newSettings = {
      sortDesc: order === 'descend',
      /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
      sortKey: columnKey as any,
      tableLimit: tablePagination.pageSize,
      tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
    };
    const shouldPush = settings.tableOffset !== newSettings.tableOffset;
    updateSettings(newSettings, shouldPush);
  }, [ columns, settings, updateSettings ]);

  const handleTableRowSelect = useCallback(rowKeys => {
    updateSettings({ row: rowKeys });
  }, [ updateSettings ]);

  usePolling(fetchAll);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [ updateSettings ]);

  return (
    <Page
      id="tasks"
      options={<FilterCounter activeFilterCount={filterCount} onReset={resetFilters} /> }
      title="Tasks">
      <div className={css.base}>
        <TableBatch
          actions={[ { disabled: !hasKillable, label: Action.Kill, value: Action.Kill } ]}
          selectedRowCount={(settings.row || []).length}
          onAction={handleBatchAction}
          onClear={clearSelected}
        />
        <ResponsiveTable<CommandTask>
          columns={columns}
          dataSource={filteredTasks}
          loading={tasks === undefined}
          pagination={getFullPaginationConfig({
            limit: settings.tableLimit,
            offset: settings.tableOffset,
          }, filteredTasks.length)}
          rowClassName={() => defaultRowClassName({ clickable: false })}
          rowKey="id"
          rowSelection={{
            onChange: handleTableRowSelect,
            preserveSelectedRowKeys: true,
            selectedRowKeys: settings.row,
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
