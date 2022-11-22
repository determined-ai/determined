import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal, Space } from 'antd';
import {
  FilterDropdownProps,
  FilterValue,
  SorterResult,
  TablePaginationConfig,
} from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import FilterCounter from 'components/FilterCounter';
import Grid from 'components/Grid';
import Link from 'components/Link';
import Page from 'components/Page';
import InteractiveTable, {
  ColumnDef,
  InteractiveTableSettings,
} from 'components/Table/InteractiveTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  relativeTimeRenderer,
  stateRenderer,
  taskIdRenderer,
  taskNameRenderer,
  TaskRenderer,
  taskTypeRenderer,
  userRenderer,
} from 'components/Table/Table';
import TableBatch from 'components/Table/TableBatch';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import TaskActionDropdown from 'components/TaskActionDropdown';
import { commandTypeToLabel } from 'constants/states';
import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getCommands, getJupyterLabs, getShells, getTensorBoards, killTask } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import usePolling from 'shared/hooks/usePolling';
import { ValueOf } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { alphaNumericSorter, dateTimeStringSorter, numericSorter } from 'shared/utils/sort';
import { ShirtSize } from 'themes';
import { ExperimentAction as Action, AnyTask, CommandState, CommandTask, CommandType } from 'types';
import handleError from 'utils/error';
import { commandStateSorter, filterTasks, isTaskKillable, taskFromCommandTask } from 'utils/task';
import { getDisplayName } from 'utils/user';

import css from './TaskList.module.scss';
import settingsConfig, {
  ALL_SORTKEY,
  DEFAULT_COLUMN_WIDTHS,
  isOfSortKey,
  Settings,
} from './TaskList.settings';

const TensorBoardSourceType = {
  Experiment: 'Experiment',
  Trial: 'Trial',
} as const;

type TensorBoardSourceType = ValueOf<typeof TensorBoardSourceType>;

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

const filterKeys: Array<keyof Settings> = ['search', 'state', 'type', 'user'];

const TaskList: React.FC = () => {
  const {
    users,
    auth: { user },
  } = useStore();
  const [canceler] = useState(new AbortController());
  const [tasks, setTasks] = useState<CommandTask[] | undefined>(undefined);
  const [sourcesModal, setSourcesModal] = useState<SourceInfo>();
  const pageRef = useRef<HTMLElement>(null);

  const { activeSettings, resetSettings, settings, updateSettings } =
    useSettings<Settings>(settingsConfig);

  const fetchUsers = useFetchUsers(canceler);

  const loadedTasks = useMemo(() => tasks?.map(taskFromCommandTask) || [], [tasks]);

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
  }, [loadedTasks, settings, users]);

  const taskMap = useMemo(() => {
    return (loadedTasks || []).reduce((acc, task) => {
      acc[task.id] = task;
      return acc;
    }, {} as Record<string, CommandTask>);
  }, [loadedTasks]);

  const selectedTasks = useMemo(() => {
    return (settings.row || []).map((id) => taskMap[id]).filter((task) => !!task);
  }, [settings.row, taskMap]);

  const hasKillable = useMemo(() => {
    for (const task of selectedTasks) {
      if (isTaskKillable(task)) return true;
    }
    return false;
  }, [selectedTasks]);

  const filterCount = useMemo(() => activeSettings(filterKeys).length, [activeSettings]);

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [updateSettings]);

  const resetFilters = useCallback(() => {
    resetSettings([...filterKeys, 'tableOffset']);
    clearSelected();
  }, [clearSelected, resetSettings]);

  const fetchTasks = useCallback(async () => {
    try {
      const [commands, jupyterLabs, shells, tensorboards] = await Promise.all([
        getCommands({ signal: canceler.signal }),
        getJupyterLabs({ signal: canceler.signal }),
        getShells({ signal: canceler.signal }),
        getTensorBoards({ signal: canceler.signal }),
      ]);
      const newTasks = [...commands, ...jupyterLabs, ...shells, ...tensorboards];
      setTasks((prev) => {
        if (isEqual(prev, newTasks)) return prev;
        return newTasks;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch tasks.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [canceler]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchUsers(), fetchTasks()]);
  }, [fetchTasks, fetchUsers]);

  const handleSourceShow = useCallback((info: SourceInfo) => setSourcesModal(info), []);
  const handleSourceDismiss = useCallback(() => setSourcesModal(undefined), []);

  const handleActionComplete = useCallback(() => fetchAll(), [fetchAll]);

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" />, []);

  const handleNameSearchApply = useCallback(
    (newSearch: string) => {
      updateSettings({ row: undefined, search: newSearch || undefined });
    },
    [updateSettings],
  );

  const handleNameSearchReset = useCallback(() => {
    updateSettings({ row: undefined, search: undefined });
  }, [updateSettings]);

  const nameFilterSearch = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterSearch
        {...filterProps}
        value={settings.search || ''}
        onReset={handleNameSearchReset}
        onSearch={handleNameSearchApply}
      />
    ),
    [handleNameSearchApply, handleNameSearchReset, settings.search],
  );

  const handleTypeFilterApply = useCallback(
    (types: string[]) => {
      updateSettings({
        row: undefined,
        type: types.length !== 0 ? (types as CommandType[]) : undefined,
      });
    },
    [updateSettings],
  );

  const handleTypeFilterReset = useCallback(() => {
    updateSettings({ row: undefined, type: undefined });
  }, [updateSettings]);

  const typeFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        values={settings.type}
        width={180}
        onFilter={handleTypeFilterApply}
        onReset={handleTypeFilterReset}
      />
    ),
    [handleTypeFilterApply, handleTypeFilterReset, settings.type],
  );

  const handleStateFilterApply = useCallback(
    (states: string[]) => {
      updateSettings({
        row: undefined,
        state: states.length !== 0 ? (states as CommandState[]) : undefined,
      });
    },
    [updateSettings],
  );

  const handleStateFilterReset = useCallback(() => {
    updateSettings({ row: undefined, state: undefined });
  }, [updateSettings]);

  const stateFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        values={settings.state}
        onFilter={handleStateFilterApply}
        onReset={handleStateFilterReset}
      />
    ),
    [handleStateFilterApply, handleStateFilterReset, settings.state],
  );

  const handleUserFilterApply = useCallback(
    (users: string[]) => {
      updateSettings({
        row: undefined,
        user: users.length !== 0 ? users : undefined,
      });
    },
    [updateSettings],
  );

  const handleUserFilterReset = useCallback(() => {
    updateSettings({ row: undefined, user: undefined });
  }, [updateSettings]);

  const userFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        searchable
        values={settings.user}
        onFilter={handleUserFilterApply}
        onReset={handleUserFilterReset}
      />
    ),
    [handleUserFilterApply, handleUserFilterReset, settings.user],
  );

  const columns = useMemo(() => {
    const nameNSourceRenderer: TaskRenderer = (_, record, index) => {
      if (record.type !== CommandType.TensorBoard || !record.misc) {
        return taskNameRenderer(_, record, index);
      }

      const info = {
        path: '',
        plural: '',
        sources: [] as TensorBoardSource[],
      };
      record.misc.experimentIds.forEach((id) => {
        info.sources.push({
          id,
          path: paths.experimentDetails(id),
          type: TensorBoardSourceType.Experiment,
        });
      });
      record.misc.trialIds.forEach((id) => {
        info.sources.push({
          id,
          path: paths.trialDetails(id),
          type: TensorBoardSourceType.Trial,
        });
      });
      info.plural = info.sources.length > 1 ? 's' : '';
      info.sources.sort((a, b) => {
        if (a.type !== b.type) return alphaNumericSorter(a.type, b.type);
        return numericSorter(a.id, b.id);
      });

      return (
        <div className={css.sourceName}>
          {taskNameRenderer(_, record, index)}
          <button className="ignoreTableRowClick" onClick={() => handleSourceShow(info)}>
            Show {info.sources.length} Source{info.plural}
          </button>
        </div>
      );
    };

    const actionRenderer: TaskRenderer = (_, record) => (
      <TaskActionDropdown task={record} onComplete={handleActionComplete} />
    );

    return [
      {
        dataIndex: 'id',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['id'],
        key: 'id',
        render: taskIdRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => alphaNumericSorter(a.id, b.id),
        title: 'Short ID',
      },
      {
        dataIndex: 'type',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['type'],
        filterDropdown: typeFilterDropdown,
        filters: Object.values(CommandType).map((value) => ({
          text: (
            <div className={css.typeFilter}>
              <Icon name={value.toLocaleLowerCase()} />
              <span>{commandTypeToLabel[value]}</span>
            </div>
          ),
          value,
        })),
        isFiltered: (settings: Settings) => !!settings.type,
        key: 'type',
        render: taskTypeRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => alphaNumericSorter(a.type, b.type),
        title: 'Type',
      },
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        isFiltered: (settings: Settings) => !!settings.search,
        key: 'name',
        render: nameNSourceRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => alphaNumericSorter(a.name, b.name),
        title: 'Name',
      },
      {
        dataIndex: 'startTime',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['startTime'],
        key: 'startTime',
        render: (_: number, record: CommandTask): React.ReactNode => {
          return relativeTimeRenderer(new Date(record.startTime));
        },
        sorter: (a: CommandTask, b: CommandTask): number => {
          return dateTimeStringSorter(a.startTime, b.startTime);
        },
        title: 'Start Time',
      },
      {
        dataIndex: 'state',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['state'],
        filterDropdown: stateFilterDropdown,
        filters: Object.values(CommandState).map((value) => ({
          text: <Badge state={value} type={BadgeType.State} />,
          value,
        })),
        isFiltered: (settings: Settings) => !!settings.state,
        key: 'state',
        render: stateRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => commandStateSorter(a.state, b.state),
        title: 'State',
      },
      {
        dataIndex: 'resourcePool',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['resourcePool'],
        key: 'resourcePool',
        sorter: (a: CommandTask, b: CommandTask): number =>
          alphaNumericSorter(a.resourcePool, b.resourcePool),
        title: 'Resource Pool',
      },
      {
        dataIndex: 'user',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['user'],
        filterDropdown: userFilterDropdown,
        filters: users.map((user) => ({ text: getDisplayName(user), value: user.id })),
        isFiltered: (settings: Settings) => !!settings.user,
        key: 'user',
        render: userRenderer,
        sorter: (a: CommandTask, b: CommandTask): number => {
          return alphaNumericSorter(
            getDisplayName(users.find((u) => u.id === a.userId)),
            getDisplayName(users.find((u) => u.id === b.userId)),
          );
        },
        title: 'User',
      },
      {
        align: 'right',
        className: 'fullCell',
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        fixed: 'right',
        key: 'action',
        render: actionRenderer,
        title: '',
      },
    ] as ColumnDef<CommandTask>[];
  }, [
    handleActionComplete,
    handleSourceShow,
    nameFilterSearch,
    stateFilterDropdown,
    tableSearchIcon,
    typeFilterDropdown,
    userFilterDropdown,
    users,
  ]);

  const handleBatchKill = useCallback(async () => {
    try {
      const promises = selectedTasks
        .filter((task) => isTaskKillable(task))
        .map((task) => killTask(task));
      await Promise.all(promises);

      /*
       * Deselect selected rows since their states may have changed where they
       * are no longer part of the filter criteria.
       */
      updateSettings({ row: undefined });

      // Refetch task list to get updates based on batch action.
      fetchAll();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to Kill Selected Tasks',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [fetchAll, selectedTasks, updateSettings]);

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
  }, [handleBatchKill]);

  const handleBatchAction = useCallback(
    (action?: string) => {
      if (action === Action.Kill) showConfirmation();
    },
    [showConfirmation],
  );

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      tableFilters: Record<string, FilterValue | null>,
      tableSorter: SorterResult<CommandTask> | SorterResult<CommandTask>[],
    ) => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<CommandTask>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;
      const newSettings = {
        sortDesc: order === 'descend',
        sortKey: isOfSortKey(columnKey) ? columnKey : ALL_SORTKEY[0],
        tableLimit: tablePagination.pageSize,
        tableOffset: (tablePagination.current ?? 1 - 1) * (tablePagination.pageSize ?? 0),
      };
      const shouldPush = settings.tableOffset !== newSettings.tableOffset;
      updateSettings(newSettings, shouldPush);
    },
    [columns, settings, updateSettings],
  );

  const handleTableRowSelect = useCallback(
    (rowKeys: React.Key[]) => {
      updateSettings({ row: rowKeys as string[] });
    },
    [updateSettings],
  );

  usePolling(fetchAll, { rerunOnNewFn: true });

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  const TaskActionDropdownCM = useCallback(
    ({
      record,
      onVisibleChange,
      children,
    }: {
      children: React.ReactNode;
      onVisibleChange?: (visible: boolean) => void;
      record: AnyTask;
    }) => (
      <TaskActionDropdown
        curUser={user}
        task={record}
        onComplete={handleActionComplete}
        onVisibleChange={onVisibleChange}>
        {children}
      </TaskActionDropdown>
    ),
    [user, handleActionComplete],
  );

  return (
    <Page
      containerRef={pageRef}
      id="tasks"
      options={
        <Space>
          {filterCount > 0 && (
            <FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />
          )}
        </Space>
      }
      title="Tasks">
      <div className={css.base}>
        <TableBatch
          actions={[{ disabled: !hasKillable, label: Action.Kill, value: Action.Kill }]}
          selectedRowCount={(settings.row ?? []).length}
          onAction={handleBatchAction}
          onClear={clearSelected}
        />
        <InteractiveTable
          columns={columns}
          containerRef={pageRef}
          ContextMenu={TaskActionDropdownCM}
          dataSource={filteredTasks}
          loading={tasks === undefined}
          pagination={getFullPaginationConfig(
            {
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            },
            filteredTasks.length,
          )}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="id"
          rowSelection={{
            onChange: handleTableRowSelect,
            preserveSelectedRowKeys: true,
            selectedRowKeys: settings.row ?? [],
          }}
          settings={settings as InteractiveTableSettings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
          onChange={handleTableChange}
        />
      </div>
      <Modal
        footer={null}
        open={!!sourcesModal}
        style={{ minWidth: '600px' }}
        title={`
          ${sourcesModal?.sources.length}
          TensorBoard Source${sourcesModal?.plural}
        `}
        onCancel={handleSourceDismiss}>
        <div className={css.sourceLinks}>
          <Grid gap={ShirtSize.Medium} minItemWidth={120}>
            {sourcesModal?.sources.map((source) => (
              <Link key={source.id} path={source.path}>
                {source.type} {source.id}
              </Link>
            ))}
          </Grid>
        </div>
      </Modal>
    </Page>
  );
};

export default TaskList;
