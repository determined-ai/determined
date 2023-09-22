import { Modal, Space } from 'antd';
import {
  FilterDropdownProps,
  FilterValue,
  SorterResult,
  TablePaginationConfig,
} from 'antd/es/table/interface';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import FilterCounter from 'components/FilterCounter';
import Grid from 'components/Grid';
import JupyterLabButton from 'components/JupyterLabButton';
import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import { ShirtSize } from 'components/kit/Theme';
import { Loadable } from 'components/kit/utils/loadable';
import Link from 'components/Link';
import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  relativeTimeRenderer,
  stateRenderer,
  taskIdRenderer,
  taskNameRenderer,
  TaskRenderer,
  taskTypeRenderer,
  taskWorkspaceRenderer,
  userRenderer,
} from 'components/Table/Table';
import TableBatch from 'components/Table/TableBatch';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import TaskActionDropdown from 'components/TaskActionDropdown';
import settingsConfig, {
  ALL_SORTKEY,
  DEFAULT_COLUMN_WIDTHS,
  isOfSortKey,
  Settings,
} from 'components/TaskList.settings';
import { commandTypeToLabel } from 'constants/states';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getCommands, getJupyterLabs, getShells, getTensorBoards, killTask } from 'services/api';
import userStore from 'stores/users';
import workspaceStore from 'stores/workspaces';
import {
  ExperimentAction as Action,
  AnyTask,
  CommandState,
  CommandTask,
  CommandType,
  ValueOf,
  Workspace,
} from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';
import { alphaNumericSorter, dateTimeStringSorter, numericSorter } from 'utils/sort';
import { commandStateSorter, filterTasks, isTaskKillable, taskFromCommandTask } from 'utils/task';
import { getDisplayName } from 'utils/user';

import BatchActionConfirmModalComponent from './BatchActionConfirmModal';
import { useModal } from './kit/Modal';
import css from './TaskList.module.scss';
import WorkspaceFilter from './WorkspaceFilter';

const TensorBoardSourceType = {
  Experiment: 'Experiment',
  Trial: 'Trial',
} as const;

type TensorBoardSourceType = ValueOf<typeof TensorBoardSourceType>;

interface Props {
  workspace?: Workspace;
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

const filterKeys: Array<keyof Settings> = ['search', 'state', 'type', 'user', 'workspace'];

const TaskList: React.FC<Props> = ({ workspace }: Props) => {
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));
  const [tasks, setTasks] = useState<CommandTask[] | undefined>(undefined);
  const [sourcesModal, setSourcesModal] = useState<SourceInfo>();
  const pageRef = useRef<HTMLElement>(null);
  const workspaceId = useMemo(() => workspace?.id.toString() ?? 'global', [workspace?.id]);
  const stgsConfig = useMemo(() => settingsConfig(workspaceId), [workspaceId]);
  const { activeSettings, resetSettings, settings, updateSettings } =
    useSettings<Settings>(stgsConfig);
  const { canCreateNSC, canCreateWorkspaceNSC } = usePermissions();
  const { canModifyWorkspaceNSC } = usePermissions();
  const canceler = useRef(new AbortController());

  const BatchActionConfirmModal = useModal(BatchActionConfirmModalComponent);

  const loadedTasks = useMemo(() => tasks?.map(taskFromCommandTask) || [], [tasks]);

  const filteredTasks = useMemo(() => {
    return filterTasks<CommandType, CommandTask>(
      loadedTasks,
      {
        limit: settings.tableLimit,
        states: settings.state,
        types: settings.type as CommandType[],
        users: settings.user,
        workspaces: settings.workspace,
      },
      users,
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
      if (isTaskKillable(task, canModifyWorkspaceNSC({ workspace: { id: task.workspaceId } })))
        return true;
    }
    return false;
  }, [selectedTasks, canModifyWorkspaceNSC]);

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
        getCommands({ signal: canceler.current.signal, workspaceId: workspace?.id }),
        getJupyterLabs({ signal: canceler.current.signal, workspaceId: workspace?.id }),
        getShells({ signal: canceler.current.signal, workspaceId: workspace?.id }),
        getTensorBoards({ signal: canceler.current.signal, workspaceId: workspace?.id }),
      ]);
      const newTasks = [...commands, ...jupyterLabs, ...shells, ...tensorboards];
      setTasks((prev) => {
        if (_.isEqual(prev, newTasks)) return prev;
        return newTasks;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch tasks.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [workspace?.id]);

  const handleSourceShow = useCallback((info: SourceInfo) => setSourcesModal(info), []);
  const handleSourceDismiss = useCallback(() => setSourcesModal(undefined), []);

  const handleActionComplete = useCallback(() => fetchTasks(), [fetchTasks]);

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" title="Search" />, []);

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

  const handleWorkspaceFilterApply = useCallback(
    (workspaces: string[]) => {
      updateSettings({
        row: undefined,
        workspace: workspaces.length !== 0 ? workspaces : undefined,
      });
    },
    [updateSettings],
  );

  const handleWorkspaceFilterReset = useCallback(() => {
    updateSettings({ row: undefined, workspace: undefined });
  }, [updateSettings]);

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
  const workspaceFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        values={settings.workspace?.map((ws) => ws)}
        width={220}
        onFilter={handleWorkspaceFilterApply}
        onReset={handleWorkspaceFilterReset}
      />
    ),
    [handleWorkspaceFilterApply, handleWorkspaceFilterReset, settings.workspace],
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
          <Button type="text" onClick={() => handleSourceShow(info)}>
            Show {info.sources.length} Source{info.plural}
          </Button>
        </div>
      );
    };

    const actionRenderer: TaskRenderer = (_, record) => (
      <TaskActionDropdown task={record} onComplete={handleActionComplete} />
    );

    const cols = [
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
        filterIcon: <Icon name="filter" title="filter" />,
        filters: Object.values(CommandType).map((value) => ({
          text: (
            <div className={css.typeFilter}>
              <Icon name={value} title={commandTypeToLabel[value]} />
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
        title: 'Started',
      },
      {
        dataIndex: 'state',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['state'],
        filterDropdown: stateFilterDropdown,
        filterIcon: <Icon name="filter" title="filter" />,
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
        filterIcon: <Icon name="filter" title="filter" />,
        filters: users.map((user) => ({ text: getDisplayName(user), value: user.id })),
        isFiltered: (settings: Settings) => !!settings.user,
        key: 'user',
        render: (_: string, r: CommandTask) => userRenderer(users.find((u) => u.id === r.userId)),
        sorter: (a: CommandTask, b: CommandTask): number => {
          return alphaNumericSorter(
            getDisplayName(users.find((u) => u.id === a.userId)),
            getDisplayName(users.find((u) => u.id === b.userId)),
          );
        },
        title: 'User',
      },
      workspaceId === 'global' && {
        align: 'center',
        dataIndex: 'workspace',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['workspace'],
        filterDropdown: workspaceFilterDropdown,
        filterIcon: <Icon name="filter" title="filter" />,
        filters: workspaces.map((ws) => ({
          text: <WorkspaceFilter workspace={ws} />,
          value: ws.id,
        })),
        isFiltered: (settings: Settings) => !!settings.workspace && !!settings.workspace.length,
        key: 'workspace',
        render: (v: string, record: CommandTask) => taskWorkspaceRenderer(record, workspaces),
        sorter: (a: CommandTask, b: CommandTask): number =>
          alphaNumericSorter(
            workspaces.find((w) => w.id === a.workspaceId)?.name ?? '',
            workspaces.find((w) => w.id === b.workspaceId)?.name ?? '',
          ),
        title: 'Workspace',
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
    ].filter(Boolean) as ColumnDef<CommandTask>[];

    return cols;
  }, [
    handleActionComplete,
    handleSourceShow,
    nameFilterSearch,
    stateFilterDropdown,
    tableSearchIcon,
    typeFilterDropdown,
    userFilterDropdown,
    workspaceFilterDropdown,
    users,
    workspaces,
    workspaceId,
  ]);

  const handleBatchKill = useCallback(async () => {
    try {
      const promises = selectedTasks
        .filter((task) =>
          isTaskKillable(task, canModifyWorkspaceNSC({ workspace: { id: task.workspaceId } })),
        )
        .map((task) => killTask(task));
      await Promise.all(promises);

      /*
       * Deselect selected rows since their states may have changed where they
       * are no longer part of the filter criteria.
       */
      updateSettings({ row: undefined });

      // Refetch task list to get updates based on batch action.
      fetchTasks();
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject: 'Unable to Kill Selected Tasks',
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [fetchTasks, selectedTasks, updateSettings, canModifyWorkspaceNSC]);

  const handleBatchAction = useCallback(
    () => BatchActionConfirmModal.open(),
    [BatchActionConfirmModal],
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
        tableOffset: ((tablePagination.current ?? 1) - 1) * (tablePagination.pageSize ?? 0),
      };
      updateSettings(newSettings);
    },
    [columns, updateSettings],
  );

  const handleTableRowSelect = useCallback(
    (rowKeys: React.Key[]) => {
      updateSettings({ row: rowKeys as string[] });
    },
    [updateSettings],
  );

  usePolling(fetchTasks, { rerunOnNewFn: true });

  useEffect(() => workspaceStore.fetch(), []);

  useEffect(() => {
    const currentCanceler = canceler.current;
    return () => currentCanceler.abort();
  }, []);

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
        curUser={currentUser}
        task={record}
        onComplete={handleActionComplete}
        onVisibleChange={onVisibleChange}>
        {children}
      </TaskActionDropdown>
    ),
    [currentUser, handleActionComplete],
  );

  return (
    <>
      <div className={css.options}>
        <Space>
          {filterCount > 0 && (
            <FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />
          )}
          <JupyterLabButton
            enabled={workspace ? canCreateWorkspaceNSC({ workspace }) : canCreateNSC}
            workspace={workspace}
          />
        </Space>
      </div>
      <div className={css.base}>
        <TableBatch
          actions={[{ disabled: !hasKillable, label: Action.Kill, value: Action.Kill }]}
          selectedRowCount={(settings.row ?? []).length}
          onAction={handleBatchAction}
          onClear={clearSelected}
        />
        <InteractiveTable<CommandTask, Settings>
          columns={columns}
          containerRef={pageRef}
          ContextMenu={TaskActionDropdownCM}
          dataSource={filteredTasks}
          defaultColumns={stgsConfig.settings.columns.defaultValue}
          loading={tasks === undefined || !settings}
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
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings}
          onChange={handleTableChange}
        />
      </div>
      <BatchActionConfirmModal.Component
        batchAction={Action.Kill}
        itemName="task"
        onConfirm={handleBatchKill}
      />
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
    </>
  );
};

export default TaskList;
