import { ExclamationCircleOutlined } from '@ant-design/icons';
import {
  Column,
  ColumnDef,
  ColumnOrderState,
  ColumnResizeMode,
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  SortingState,
  useReactTable,
} from '@tanstack/react-table';
import { Input, MenuProps, Typography } from 'antd';
import { Button, Dropdown, Menu, Modal, Space } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { HTMLProps, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { useDrag, useDrop } from 'react-dnd';

import Badge, { BadgeType } from 'components/Badge';
import { useSetDynamicTabBar } from 'components/DynamicTabs';
import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import FilterCounter from 'components/FilterCounter';
import Link from 'components/Link';
import Page from 'components/Page';
import InteractiveTable, {
  // ColumnDef,
  InteractiveTableSettings,
  onRightClickableCell,
} from 'components/Table/InteractiveTable';
import {
  checkmarkRenderer,
  defaultRowClassName,
  experimentDurationRenderer,
  experimentNameRenderer,
  experimentProgressRenderer,
  ExperimentRenderer,
  getFullPaginationConfig,
  relativeTimeRenderer,
  stateRenderer,
  userRenderer,
} from 'components/Table/Table';
import TableBatch from 'components/Table/TableBatch';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import TagList from 'components/TagList';
import Toggle from 'components/Toggle';
import { useStore } from 'contexts/Store';
import useExperimentTags from 'hooks/useExperimentTags';
import { useFetchUsers } from 'hooks/useFetch';
import useModalColumnsCustomize from 'hooks/useModal/Columns/useModalColumnsCustomize';
import useModalExperimentMove, {
  Settings as MoveExperimentSettings,
  settingsConfig as moveExperimentSettingsConfig,
} from 'hooks/useModal/Experiment/useModalExperimentMove';
import usePermissions from 'hooks/usePermissions';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import {
  activateExperiment,
  archiveExperiment,
  cancelExperiment,
  deleteExperiment,
  getExperimentLabels,
  getExperiments,
  killExperiment,
  openOrCreateTensorBoard,
  patchExperiment,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import { Determinedexperimentv1State, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { GetExperimentsParams } from 'services/types';
import Icon from 'shared/components/Icon/Icon';
import usePolling from 'shared/hooks/usePolling';
import { RecordKey, ValueOf } from 'shared/types';
import { ErrorLevel } from 'shared/utils/error';
import { validateDetApiEnum, validateDetApiEnumList } from 'shared/utils/service';
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  ExperimentAction as Action,
  CommandTask,
  ExperimentItem,
  ExperimentPagination,
  Project,
  ProjectExperiment,
  RunState,
} from 'types';
import handleError from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperimentsUnion,
  getProjectExperimentForExperimentItem,
} from 'utils/experiment';
import { getDisplayName } from 'utils/user';
import { openCommand } from 'utils/wait';

import settingsConfig, {
  DEFAULT_COLUMN_WIDTHS,
  DEFAULT_COLUMNS,
  ExperimentColumnName,
  ExperimentListSettings,
} from './ExperimentList.settings';
import css from './ProjectDetails.module.scss';

const filterKeys: Array<keyof ExperimentListSettings> = ['label', 'search', 'state', 'user'];

const batchActions = [
  Action.OpenTensorBoard,
  Action.Activate,
  Action.Move,
  Action.Pause,
  Action.Archive,
  Action.Unarchive,
  Action.Cancel,
  Action.Kill,
  Action.Delete,
];

interface Props {
  project: Project;
}

const ExperimentList: React.FC<Props> = ({ project }) => {
  const {
    users,
    auth: { user },
  } = useStore();

  const [experiments, setExperiments] = useState<ExperimentItem[]>([]);
  const [labels, setLabels] = useState<string[]>([]);

  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);
  const { updateSettings: updateDestinationSettings } = useSettings<MoveExperimentSettings>(
    moveExperimentSettingsConfig,
  );

  const permissions = usePermissions();

  const id = project?.id;

  useEffect(() => {
    updateDestinationSettings({ projectId: undefined, workspaceId: project?.workspaceId });
  }, [updateDestinationSettings, project?.workspaceId]);

  const { settings, updateSettings, resetSettings, activeSettings } =
    useSettings<ExperimentListSettings>(settingsConfig);

  const experimentMap = useMemo(() => {
    return (experiments || []).reduce((acc, experiment) => {
      acc[experiment.id] = getProjectExperimentForExperimentItem(experiment, project);
      return acc;
    }, {} as Record<RecordKey, ProjectExperiment>);
  }, [experiments, project]);

  const filterCount = useMemo(() => activeSettings(filterKeys).length, [activeSettings]);

  const availableBatchActions = useMemo(() => {
    const experiments = settings.row?.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, batchActions, permissions);
  }, [experimentMap, settings.row, permissions]);

  const statesString = useMemo(() => settings.state?.join('.'), [settings.state]);
  const pinnedString = useMemo(() => JSON.stringify(settings.pinned ?? {}), [settings.pinned]);
  const labelsString = useMemo(() => settings.label?.join('.'), [settings.label]);
  const usersString = useMemo(() => settings.user?.join('.'), [settings.user]);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = statesString
        ?.split('.')
        .map((state) => encodeExperimentState(state as RunState));
      const pinned = JSON.parse(pinnedString);
      const baseParams: GetExperimentsParams = {
        archived: settings.archived ? undefined : false,
        labels: settings.label,
        name: settings.search,
        orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
        projectId: id,
        sortBy: validateDetApiEnum(V1GetExperimentsRequestSortBy, settings.sortKey),
        states: validateDetApiEnumList(Determinedexperimentv1State, states),
        users: settings.user,
      };
      const pinnedIds = pinned?.[id] ?? [];
      let pinnedExpResponse: ExperimentPagination = { experiments: [], pagination: {} };
      if (pinnedIds.length > 0) {
        pinnedExpResponse = await getExperiments(
          {
            ...baseParams,
            experimentIdFilter: { incl: pinnedIds },
            limit: settings.tableLimit,
            offset: 0,
          },
          { signal: canceler.signal },
        );
      }
      const otherExpResponse = await getExperiments(
        {
          ...baseParams,
          experimentIdFilter: { notIn: pinnedIds },
          limit: settings.tableLimit - pinnedIds.length,
          offset:
            settings.tableOffset - (settings.tableOffset / settings.tableLimit) * pinnedIds.length,
        },
        { signal: canceler.signal },
      );

      // Due to showing pinned items in all pages, we need to adjust the number of total items
      const totalItems =
        (pinnedExpResponse.pagination.total ?? 0) + (otherExpResponse.pagination.total ?? 0);
      const expectedNumPages = Math.ceil(totalItems / settings.tableLimit);
      const imaginaryTotalItems = totalItems + pinnedIds.length * expectedNumPages;
      setTotal(imaginaryTotalItems);
      setExperiments([...pinnedExpResponse.experiments, ...otherExpResponse.experiments]);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    canceler.signal,
    id,
    settings.archived,
    labelsString,
    pinnedString,
    settings.search,
    settings.sortDesc,
    settings.sortKey,
    statesString,
    settings.tableLimit,
    settings.tableOffset,
    usersString,
  ]);

  const fetchLabels = useCallback(async () => {
    try {
      const labels = await getExperimentLabels({ project_id: id }, { signal: canceler.signal });
      labels.sort((a, b) => alphaNumericSorter(a, b));
      setLabels(labels);
    } catch (e) {
      handleError(e);
    }
  }, [canceler.signal, id]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchExperiments(), fetchUsers(), fetchLabels()]);
  }, [fetchExperiments, fetchUsers, fetchLabels]);

  const { stopPolling } = usePolling(fetchAll, { rerunOnNewFn: true });

  const experimentTags = useExperimentTags(fetchAll);

  const handleActionComplete = useCallback(() => fetchExperiments(), [fetchExperiments]);

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

  const handleLabelFilterApply = useCallback(
    (labels: string[]) => {
      updateSettings({
        label: labels.length !== 0 ? labels : undefined,
        row: undefined,
      });
    },
    [updateSettings],
  );

  const handleLabelFilterReset = useCallback(() => {
    updateSettings({ label: undefined, row: undefined });
  }, [updateSettings]);

  const labelFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        searchable
        values={settings.label}
        onFilter={handleLabelFilterApply}
        onReset={handleLabelFilterReset}
      />
    ),
    [handleLabelFilterApply, handleLabelFilterReset, settings.label],
  );

  const handleStateFilterApply = useCallback(
    (states: string[]) => {
      updateSettings({
        row: undefined,
        state: states.length !== 0 ? (states as RunState[]) : undefined,
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

  const saveExperimentDescription = useCallback(async (editedDescription: string, id: number) => {
    try {
      await patchExperiment({
        body: { description: editedDescription },
        experimentId: id,
      });
    } catch (e) {
      handleError(e, {
        isUserTriggered: true,
        publicMessage: 'Unable to save experiment description.',
        silent: false,
      });
      return e as Error;
    }
  }, []);

  const canEditExperiment =
    !!project &&
    permissions.canModifyExperimentMetadata({
      workspace: { id: project.workspaceId },
    });

  const ContextMenu = useCallback(
    ({
      record,
      onVisibleChange,
      children,
    }: {
      children?: React.ReactNode;
      onVisibleChange?: ((visible: boolean) => void) | undefined;
      record: ExperimentItem;
    }) => {
      return (
        <ExperimentActionDropdown
          experiment={getProjectExperimentForExperimentItem(record, project)}
          settings={settings}
          updateSettings={updateSettings}
          onComplete={handleActionComplete}
          onVisibleChange={onVisibleChange}>
          {children}
        </ExperimentActionDropdown>
      );
    },
    [handleActionComplete, project, settings, updateSettings],
  );

  const columns = useMemo(() => {
    const tagsRenderer = (value: string, record: ExperimentItem) => (
      <div className={css.tagsRenderer}>
        <Typography.Text
          ellipsis={{
            tooltip: <TagList disabled tags={record.labels} />,
          }}>
          <div>
            <TagList
              compact
              disabled={record.archived || project?.archived || !canEditExperiment}
              tags={record.labels}
              onChange={experimentTags.handleTagListChange(record.id)}
            />
          </div>
        </Typography.Text>
      </div>
    );

    const actionRenderer: ExperimentRenderer = (_, record: ExperimentItem) => {
      return <ContextMenu record={record} />;
    };

    const descriptionRenderer = (value: string, record: ExperimentItem) => (
      <Input
        className={css.descriptionRenderer}
        defaultValue={value}
        disabled={record.archived || !canEditExperiment}
        placeholder={record.archived ? 'Archived' : canEditExperiment ? 'Add description...' : ''}
        title="Edit description"
        onPressEnter={(e) => {
          const newDesc = e.currentTarget.value;
          saveExperimentDescription(newDesc, record.id);
          e.currentTarget.blur();
        }}
      />
    );

    const forkedFromRenderer = (value: string | number | undefined): React.ReactNode =>
      value ? <Link path={paths.experimentDetails(value)}>{value}</Link> : null;

    const filterDropdownState = () => {
      return (<Menu
        items={[
          RunState.Active,
          RunState.Paused,
          RunState.Canceled,
          RunState.Completed,
          RunState.Error,
        ].map((value) => (
          {
            key: value,
            label: <Badge key={value} state={value} type={BadgeType.State} />,
          }
        ))}
        onClick={(e) => {
          handleStateFilterApply([e.key]);
        }}
      />);
    };
    type cellValue = string | number | undefined;

    function IndeterminateCheckbox({
      indeterminate,
      className = '',
      ...rest
    }: { indeterminate?: boolean } & HTMLProps<HTMLInputElement>) {
      const ref = React.useRef<HTMLInputElement>(null!);

      React.useEffect(() => {
        if (typeof indeterminate === 'boolean') {
          ref.current.indeterminate = !rest.checked && indeterminate;
        }
      }, [ref, indeterminate]);

      return (
        <input
          className={className + ' cursor-pointer'}
          ref={ref}
          type="checkbox"
          {...rest}
        />
      );
    }

    return [

      {
        accessorKey: 'selection',
        cell: ({ row }) => (
          <div className="px-1">
            <IndeterminateCheckbox
              {...{
                checked: row.getIsSelected(),
                indeterminate: row.getIsSomeSelected(),
                onChange: row.getToggleSelectedHandler(),
              }}
            />
          </div>
        ),
        header: ({ table }) => (
          <IndeterminateCheckbox
            {...{
              checked: table.getIsAllRowsSelected(),
              indeterminate: table.getIsSomeRowsSelected(),
              onChange: table.getToggleAllRowsSelectedHandler(),
            }}
          />
        ),
        size: 1,
      },
      {
        accessorKey: 'id',
        // align: 'right',
        cell: (props) => experimentNameRenderer(props.getValue() as cellValue, props.row.original),
        header: 'ID',
        size: DEFAULT_COLUMN_WIDTHS['id'],
        // key: V1GetExperimentsRequestSortBy.ID,
        // onCell: onRightClickableCell,
        // sorter: true,
      },
      {
        accessorKey: 'name',
        cell: (props) => experimentNameRenderer(props.getValue() as cellValue, props.row.original),
        // filterDropdown: nameFilterSearch,
        // filterIcon: tableSearchIcon,
        header: 'Name',

        size: DEFAULT_COLUMN_WIDTHS['name'],
        // isFiltered: (settings: ExperimentListSettings) => !!settings.search,
        // key: V1GetExperimentsRequestSortBy.NAME,
        // onCell: onRightClickableCell,
        // sorter: true,
      },
      {
        accessorKey: 'description',
        cell: (props) => descriptionRenderer(props.getValue() as string, props.row.original),
        header: 'Description',
        size: DEFAULT_COLUMN_WIDTHS['description'],
        // onCell: onRightClickableCell,
      },
      {
        accessorKey: 'tags',
        cell: (props) => tagsRenderer(props.getValue() as string, props.row.original),
        // filterDropdown: labelFilterDropdown,
        // filters: labels.map((label) => ({ text: label, value: label })),
        header: 'Tags',

        size: DEFAULT_COLUMN_WIDTHS['tags'],
        // isFiltered: (settings: ExperimentListSettings) => !!settings.label,
        // key: 'labels',
      },
      {
        accessorKey: 'forkedFrom',
        // align: 'right',
        cell: (props) => forkedFromRenderer(props.getValue() as cellValue),
        header: 'Forked From',
        size: DEFAULT_COLUMN_WIDTHS['forkedFrom'],
        // key: V1GetExperimentsRequestSortBy.FORKEDFROM,
        // onCell: onRightClickableCell,
        // sorter: true,
      },
      {
        accessorKey: 'startTime',
        // align: 'right',
        cell: (props) => relativeTimeRenderer(new Date(props.getValue() as string | number | Date)),
        header: 'Start Time',
        size: DEFAULT_COLUMN_WIDTHS['startTime'],
        // key: V1GetExperimentsRequestSortBy.STARTTIME,
        // onCell: onRightClickableCell,
        // sorter: true,
      },
      {
        accessorKey: 'duration',
        // align: 'right',
        cell: (props) => experimentDurationRenderer(props.getValue() as string, props.row.original, props.row.index),
        header: 'Duration',
        size: DEFAULT_COLUMN_WIDTHS['duration'],
        // key: 'duration',
        // onCell: onRightClickableCell,
      },
      {
        accessorKey: 'numTrials',

        header: 'Trials',
        // align: 'right',
        size: DEFAULT_COLUMN_WIDTHS['numTrials'],
        // key: V1GetExperimentsRequestSortBy.NUMTRIALS,
        // onCell: onRightClickableCell,
        // sorter: true,
      },
      {
        accessorKey: 'state',
        cell: (props) => stateRenderer(props.getValue() as string, props.row.original, props.row.original.id),
        // filterDropdown: stateFilterDropdown,
        // filters: [
        //   RunState.Active,
        //   RunState.Paused,
        //   RunState.Canceled,
        //   RunState.Completed,
        //   RunState.Error,
        // ].map((value) => ({
        //   text: <Badge state={value} type={BadgeType.State} />,
        //   value,
        // })),
        header: () => {
          return (
            <>
              <span>State</span>
              <Dropdown overlay={filterDropdownState}>
                <div>F</div>
              </Dropdown>
            </>
          );
        },
        size: DEFAULT_COLUMN_WIDTHS['state'],
        // isFiltered: () => !!settings.state,
        // key: V1GetExperimentsRequestSortBy.STATE,
        // sorter: true,
      },
      {
        accessorKey: 'searcherType',
        header: 'Searcher Type',
        size: DEFAULT_COLUMN_WIDTHS['searcherType'],
        // key: 'searcherType',
        // onCell: onRightClickableCell,
      },
      {
        accessorKey: 'resourcePool',
        header: 'Resource Pool',
        size: DEFAULT_COLUMN_WIDTHS['resourcePool'],
        // key: V1GetExperimentsRequestSortBy.RESOURCEPOOL,
        // onCell: onRightClickableCell,
        // sorter: true,
      },
      {
        accessorKey: 'progress',
        // align: 'right',
        cell: (props) => experimentProgressRenderer(props.getValue() as string, props.row.original, props.row.index),
        header: 'Progress',
        size: DEFAULT_COLUMN_WIDTHS['progress'],
        // key: V1GetExperimentsRequestSortBy.PROGRESS,
        // sorter: true,
      },
      {
        accessorKey: 'archived',
        // align: 'right',
        cell: (props) => checkmarkRenderer(props.getValue() as boolean),
        header: 'Archived',
        size: DEFAULT_COLUMN_WIDTHS['archived'],
        // key: 'archived',
      },
      {
        accessorKey: 'user',
        cell: (props) => userRenderer(props.getValue() as string, props.row.original, props.row.index),
        // filterDropdown: userFilterDropdown,
        // filters: users.map((user) => ({ text: getDisplayName(user), value: user.id })),
        header: 'User',

        size: DEFAULT_COLUMN_WIDTHS['user'],
        // isFiltered: (settings: ExperimentListSettings) => !!settings.user,
        // key: V1GetExperimentsRequestSortBy.USER,
        // sorter: true,
      },
      {
        accessorKey: 'action',

        cell: (props) => actionRenderer(props.getValue() as string, props.row.original, props.row.index),

        header: '',
        // align: 'right',
        // className: 'fullCell',
        size: DEFAULT_COLUMN_WIDTHS['action'],
        // fixed: 'right',
        // key: 'action',
        // onCell: onRightClickableCell,
        // width: DEFAULT_COLUMN_WIDTHS['action'],
      },
    ] as ColumnDef<ExperimentItem>[];
  }, [
    ContextMenu,
    experimentTags,
    labelFilterDropdown,
    labels,
    nameFilterSearch,
    saveExperimentDescription,
    canEditExperiment,
    settings,
    project,
    stateFilterDropdown,
    tableSearchIcon,
    userFilterDropdown,
    users,
  ]);

  const [columnVisibility, setColumnVisibility] = React.useState({});
  const [columnOrder, setColumnOrder] = React.useState<ColumnOrderState>([]);
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [rowSelection, setRowSelection] = React.useState({});

  const table = useReactTable({
    columnResizeMode: 'onChange',
    columns,
    data: experiments,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onRowSelectionChange: setRowSelection,

    onSortingChange: setSorting,
    state: {
      columnOrder,

      columnVisibility,
      // columnSizing,
      rowSelection,
      sorting,
    },
  });

  useEffect(() => {
    updateSettings({
      row: Object.keys(rowSelection).map((i) => parseInt(i)),
    });
  }, [rowSelection]);

  // useLayoutEffect(() => {
  // This is the failsafe for when column settings get into a bad shape.
  // if (!settings.columns?.length || !settings.columnWidths?.length) {
  //   updateSettings({
  //     columns: DEFAULT_COLUMNS,
  //     columnWidths: DEFAULT_COLUMNS.map((columnName) => DEFAULT_COLUMN_WIDTHS[columnName]),
  //   });
  // } else {
  //   const columnNames = columns.map((column) => column.id as ExperimentColumnName);
  //   const actualColumns = settings.columns.filter((name) => columnNames.includes(name));
  //   const newSettings: Partial<ExperimentListSettings> = {};
  //   if (actualColumns.length < settings.columns.length) {
  //     newSettings.columns = actualColumns;
  //   }
  //   if (settings.columnWidths.length !== actualColumns.length) {
  //     newSettings.columnWidths = actualColumns.map((name) => DEFAULT_COLUMN_WIDTHS[name]);
  //   }
  //   if (Object.keys(newSettings).length !== 0) updateSettings(newSettings);
  // }
  // }, [settings.columns, settings.columnWidths, columns, resetSettings, updateSettings]);

  const transferColumns = useMemo(() => {
    return table.getVisibleLeafColumns()
      .filter(
        (column) => column.columnDef.header !== '' && column.columnDef.header !== 'Action' && column.columnDef.header !== 'Archived',
      )
      .map((column) => {
        return column.id?.toString() ?? '';
      });
  }, [table]);

  const { contextHolder: modalExperimentMoveContextHolder, modalOpen: openMoveModal } =
    useModalExperimentMove({ onClose: handleActionComplete, user });

  const sendBatchActions = useCallback(
    (action: Action): Promise<void[] | CommandTask> | void => {
      if (!settings.row) return;
      if (action === Action.OpenTensorBoard) {
        return openOrCreateTensorBoard({ experimentIds: settings.row });
      }
      if (action === Action.Move) {
        return openMoveModal({
          experimentIds: settings.row.filter(
            (id) =>
              canActionExperiment(Action.Move, experimentMap[id]) &&
              permissions.canMoveExperiment({ experiment: experimentMap[id] }),
          ),
          sourceProjectId: project?.id,
          sourceWorkspaceId: project?.workspaceId,
        });
      }

      return Promise.all(
        (settings.row || []).map((experimentId) => {
          switch (action) {
            case Action.Activate:
              return activateExperiment({ experimentId });
            case Action.Archive:
              return archiveExperiment({ experimentId });
            case Action.Cancel:
              return cancelExperiment({ experimentId });
            case Action.Delete:
              return deleteExperiment({ experimentId });
            case Action.Kill:
              return killExperiment({ experimentId });
            case Action.Pause:
              return pauseExperiment({ experimentId });
            case Action.Unarchive:
              return unarchiveExperiment({ experimentId });
            default:
              return Promise.resolve();
          }
        }),
      );
    },
    [settings.row, openMoveModal, project?.workspaceId, project?.id, experimentMap, permissions],
  );

  const submitBatchAction = useCallback(
    async (action: Action) => {
      try {
        const result = await sendBatchActions(action);
        if (action === Action.OpenTensorBoard && result) {
          openCommand(result as CommandTask);
        }

        /*
         * Deselect selected rows since their states may have changed where they
         * are no longer part of the filter criteria.
         */
        updateSettings({ row: undefined });

        // Refetch experiment list to get updates based on batch action.
        await fetchExperiments();
      } catch (e) {
        const publicSubject =
          action === Action.OpenTensorBoard
            ? 'Unable to View TensorBoard for Selected Experiments'
            : `Unable to ${action} Selected Experiments`;
        handleError(e, {
          isUserTriggered: true,
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
        });
      }
    },
    [fetchExperiments, sendBatchActions, updateSettings, settings.row],
  );

  const showConfirmation = useCallback(
    (action: Action) => {
      Modal.confirm({
        content: `
        Are you sure you want to ${action.toLocaleLowerCase()}
        all the eligible selected experiments?
      `,
        icon: <ExclamationCircleOutlined />,
        okText: /cancel/i.test(action) ? 'Confirm' : action,
        onOk: () => submitBatchAction(action),
        title: 'Confirm Batch Action',
      });
    },
    [submitBatchAction],
  );

  const handleBatchAction = useCallback(
    (action?: string) => {
      if (action === Action.OpenTensorBoard || action === Action.Move) {
        submitBatchAction(action);
      } else {
        showConfirmation(action as Action);
      }
    },
    [submitBatchAction, showConfirmation],
  );

  const handleTableRowSelect = useCallback(
    (rowKeys) => {
      updateSettings({ row: rowKeys });
    },
    [updateSettings],
  );

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [updateSettings]);

  const resetFilters = useCallback(() => {
    resetSettings([...filterKeys, 'tableOffset']);
    clearSelected();
  }, [clearSelected, resetSettings]);

  const handleUpdateColumns = useCallback(
    (columns: ExperimentColumnName[]) => {
      if (columns.length === 0) {
        updateSettings({
          columns: ['id', 'name'],
          // columnWidths: [DEFAULT_COLUMN_WIDTHS['id'], DEFAULT_COLUMN_WIDTHS['name']],
        });
      } else {
        updateSettings({
          columns: columns,
          // columnWidths: columns.map((col) => DEFAULT_COLUMN_WIDTHS[col]),
        });
      }
    },
    [updateSettings],
  );

  useEffect(() => {
    if (settings.columns.length) {
      const visibility = {
        action: true,
      };
      table.getAllLeafColumns().forEach((c) => {
        if (settings.columns.includes(c.id as ExperimentColumnName)) {
          visibility[c.id] = true;
        } else if (c.id !== 'action' && c.id !== 'selection') {
          visibility[c.id] = false;
        }
      });
      setColumnVisibility(visibility);
      setColumnOrder([
        'selection',
        ...settings.columns.filter((c) => visibility[c]),
      ]);
    }
  }, [settings.columns, table]);

  // useEffect(() => {
  //   const sizing = {};
  //   columnOrder.forEach((c, i) => {
  //     sizing[c] = settings.columnWidths[i];
  //   });
  //   setColumnSizing(sizing);
  // }, [settings.columnWidths, columnOrder]);

  const { contextHolder: modalColumnsCustomizeContextHolder, modalOpen: openCustomizeColumns } =
    useModalColumnsCustomize({
      columns: transferColumns,
      defaultVisibleColumns: DEFAULT_COLUMNS,
      initialVisibleColumns: settings.columns?.filter((col) => transferColumns.includes(col)),
      onSave: handleUpdateColumns as (columns: string[]) => void,
    });

  const handleCustomizeColumnsClick = useCallback(() => {
    openCustomizeColumns({});
  }, [openCustomizeColumns]);

  const switchShowArchived = useCallback(
    (showArchived: boolean) => {
      let newColumns: ExperimentColumnName[];
      let newColumnWidths: number[];

      if (showArchived) {
        if (settings.columns?.includes('archived')) {
          // just some defensive coding: don't add archived twice
          newColumns = settings.columns;
          newColumnWidths = settings.columnWidths;
        } else {
          newColumns = [...settings.columns, 'archived'];
          newColumnWidths = [...settings.columnWidths, DEFAULT_COLUMN_WIDTHS['archived']];
        }
      } else {
        const archivedIndex = settings.columns.indexOf('archived');
        if (archivedIndex !== -1) {
          newColumns = [...settings.columns];
          newColumnWidths = [...settings.columnWidths];
          newColumns.splice(archivedIndex, 1);
          newColumnWidths.splice(archivedIndex, 1);
        } else {
          newColumns = settings.columns;
          newColumnWidths = settings.columnWidths;
        }
      }
      updateSettings({
        archived: showArchived,
        columns: newColumns,
        columnWidths: newColumnWidths,
        row: undefined,
      });
    },
    [settings, updateSettings],
  );

  useEffect(() => {
    if (settings.tableOffset > total) {
      const newTotal = settings.tableOffset > total ? total : total - 1;
      const offset = settings.tableLimit * Math.floor(newTotal / settings.tableLimit);
      updateSettings({ tableOffset: offset });
    }
  }, [total, settings.tableOffset, settings.tableLimit, updateSettings]);

  /*
   * Get new experiments based on changes to the
   * filters, pagination, search and sorter.
   */
  useEffect(() => {
    setIsLoading(true);
    fetchExperiments();
  }, [
    fetchExperiments,
    settings.archived,
    labelsString,
    settings.search,
    settings.sortDesc,
    settings.sortKey,
    statesString,
    pinnedString,
    settings.tableLimit,
    settings.tableOffset,
    usersString,
  ]);

  // cleanup
  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();

      setExperiments([]);
      setLabels([]);
      setIsLoading(true);
      setTotal(0);
    };
  }, [canceler, stopPolling]);

  const tabBarContent = useMemo(() => {
    const getMenuProps = (): { items: MenuProps['items']; onClick: MenuProps['onClick'] } => {
      const MenuKey = {
        Columns: 'columns',
        ResultFilter: 'resetFilters',
        SwitchArchived: 'switchArchive',
      } as const;

      const funcs = {
        [MenuKey.SwitchArchived]: () => {
          switchShowArchived(!settings.archived);
        },
        [MenuKey.Columns]: () => {
          handleCustomizeColumnsClick();
        },
        [MenuKey.ResultFilter]: () => {
          resetFilters();
        },
      };

      const onItemClick: MenuProps['onClick'] = (e) => {
        funcs[e.key as ValueOf<typeof MenuKey>]();
      };

      const menuItems: MenuProps['items'] = [
        {
          key: MenuKey.SwitchArchived,
          label: settings.archived ? 'Hide Archived' : 'Show Archived',
        },
        { key: MenuKey.Columns, label: 'Columns' },
      ];
      if (filterCount > 0) {
        menuItems.push({ key: MenuKey.ResultFilter, label: `Clear Filters (${filterCount})` });
      }
      return { items: menuItems, onClick: onItemClick };
    };
    return (
      <div className={css.tabOptions}>
        <Space className={css.actionList}>
          <Toggle
            checked={settings.archived}
            prefixLabel="Show Archived"
            onChange={switchShowArchived}
          />
          <Button onClick={handleCustomizeColumnsClick}>Columns</Button>
          <FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />
        </Space>
        <div className={css.actionOverflow} title="Open actions menu">
          <Dropdown overlay={<Menu {...getMenuProps()} />} trigger={['click']}>
            <div>
              <Icon name="overflow-vertical" />
            </div>
          </Dropdown>
        </div>
      </div>
    );
  }, [
    filterCount,
    handleCustomizeColumnsClick,
    resetFilters,
    settings.archived,
    switchShowArchived,
  ]);

  useSetDynamicTabBar(tabBarContent);

  const DraggableColumnHeader: React.FC<{
    columnOrder,
    header, setColumnOrder
  }> = ({ header, columnOrder, setColumnOrder }) => {

    const reorderColumn = (
      draggedColumnId: string,
      targetColumnId: string,
      columnOrder: string[],
    ): ColumnOrderState => {
      columnOrder.splice(
        columnOrder.indexOf(targetColumnId),
        0,
        columnOrder.splice(columnOrder.indexOf(draggedColumnId), 1)[0] as string,
      );
      return [...columnOrder];
    };
    const { column } = header;

    const [, dropRef] = useDrop({
      accept: 'column',
      drop: (draggedColumn: Column<ExperimentItem>) => {
        const newColumnOrder = reorderColumn(
          draggedColumn.id,
          column.id,
          columnOrder,
        );
        setColumnOrder(newColumnOrder);
        updateSettings({
          columns: newColumnOrder as ExperimentColumnName[],
        });
      },
    });

    const [{ isDragging }, dragRef, previewRef] = useDrag({
      collect: (monitor) => ({
        isDragging: monitor.isDragging(),
      }),
      item: () => column,
      type: 'column',
    });

    return (
      <th
        className="ant-table-cell"
        colSpan={header.colSpan}
        key={header.id}
        ref={dropRef}
        style={{
          // minWidth: header.getSize(),
          opacity: isDragging ? 0.5 : 1,

          width: header.getSize(),
        }}>
        <div
          ref={previewRef}
          {...{
            className: header.column.getCanSort()
              ? 'cursor-pointer select-none'
              : '',
            onClick: header.column.getToggleSortingHandler(),
          }}>
          <div ref={dragRef}>
            {header.isPlaceholder
              ? null
              : flexRender(header.column.columnDef.header, header.getContext())}
            {{
              asc: ' 🔼',
              desc: ' 🔽',
            }[header.column.getIsSorted() as string] ?? null}
          </div>
          <div
            {...{
              className: `${css.resizer} ${header.column.getIsResizing() ? css.isResizing : ''
                }`,
              onMouseDown: header.getResizeHandler(),
              onTouchStart: header.getResizeHandler(),
              style: {
                transform: '',
              },
            }}
          />
        </div>
      </th>
    );
  };
  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      // for docTitle, when id is 1 that means Uncategorized from webui/react/src/routes/routes.ts
      docTitle={id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <div className={css.experimentTab}>
        <TableBatch
          actions={batchActions.map((action) => ({
            disabled: !availableBatchActions.includes(action),
            label: action,
            value: action,
          }))}
          selectedRowCount={(settings.row ?? []).length}
          onAction={handleBatchAction}
          onClear={clearSelected}
        />
        <div className="p-2">
          <table style={{ width: '100%' }}>
            <thead className="ant-table-thead">
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <DraggableColumnHeader
                      columnOrder={columnOrder}
                      header={header}
                      key={header.id}
                      setColumnOrder={setColumnOrder}
                    />
                  ))}
                </tr>
              ))}
            </thead>
            <tbody className="ant-table-tbody">
              {table.getRowModel().rows.map((row) => (
                <tr
                  className="ant-table-row ant-table-row-level-0"
                  key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <td
                      className="ant-table-cell"
                      key={cell.id}
                      style={{
                        height: 60,
                        overflow: 'hidden',
                        paddingBottom: 0,
                        paddingTop: 0,
                        width: cell.column.getSize(),
                      }}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      {modalColumnsCustomizeContextHolder}
      {modalExperimentMoveContextHolder}
    </Page>
  );
};

export default ExperimentList;
