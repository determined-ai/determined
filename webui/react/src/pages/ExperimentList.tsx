import { Space, Typography } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import ColumnsCustomizeModalComponent from 'components/ColumnsCustomizeModal';
import { useSetDynamicTabBar } from 'components/DynamicTabs';
import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import FilterCounter from 'components/FilterCounter';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import Input from 'components/kit/Input';
import { useModal } from 'components/kit/Modal';
import Tags from 'components/kit/Tags';
import Toggle from 'components/kit/Toggle';
import Link from 'components/Link';
import Spinner from 'components/Spinner';
import InteractiveTable, {
  ColumnDef,
  onRightClickableCell,
} from 'components/Table/InteractiveTable';
import {
  checkmarkRenderer,
  defaultRowClassName,
  experimentDurationRenderer,
  experimentNameRenderer,
  experimentProgressRenderer,
  ExperimentRenderer,
  expStateRenderer,
  getFullPaginationConfig,
  relativeTimeRenderer,
  userRenderer,
} from 'components/Table/Table';
import TableBatch from 'components/Table/TableBatch';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import TableFilterSearch from 'components/Table/TableFilterSearch';
import useExperimentTags from 'hooks/useExperimentTags';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
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
import { Experimentv1State, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { GetExperimentsParams } from 'services/types';
import userStore from 'stores/users';
import { RecordKey } from 'types';
import {
  ExperimentAction as Action,
  CommandResponse,
  CommandTask,
  ExperimentItem,
  ExperimentPagination,
  Project,
  ProjectExperiment,
  RunState,
} from 'types';
import { ErrorLevel } from 'utils/error';
import handleError from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperimentsUnion,
  getProjectExperimentForExperimentItem,
} from 'utils/experiment';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';
import { validateDetApiEnum, validateDetApiEnumList } from 'utils/service';
import { alphaNumericSorter } from 'utils/sort';
import { humanReadableBytes } from 'utils/string';
import { getDisplayName } from 'utils/user';
import { openCommandResponse } from 'utils/wait';

import BatchActionConfirmModalComponent from '../components/BatchActionConfirmModal';

import {
  DEFAULT_COLUMN_WIDTHS,
  DEFAULT_COLUMNS,
  ExperimentColumnName,
  ExperimentListSettings,
  settingsConfigForProject,
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

const MenuKey = {
  Columns: 'columns',
  ResultFilter: 'resetFilters',
  SwitchArchived: 'switchArchive',
} as const;

const ExperimentList: React.FC<Props> = ({ project }) => {
  const [experiments, setExperiments] = useState<ExperimentItem[]>([]);
  const [labels, setLabels] = useState<string[]>([]);
  const [batchMovingExperimentIds, setBatchMovingExperimentIds] = useState<number[]>();
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [batchAction, setBatchAction] = useState<Action>();
  const canceler = useRef(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const users = Loadable.getOrElse([], useObservable(userStore.getUsers()));
  const permissions = usePermissions();

  const id = project?.id;

  const settingsConfig = useMemo(() => settingsConfigForProject(id), [id]);

  const {
    settings,
    isLoading: isLoadingSettings,
    updateSettings,
    resetSettings,
    activeSettings,
  } = useSettings<ExperimentListSettings>(settingsConfig);

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
    if (!settings || isLoadingSettings) return;
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
        states: validateDetApiEnumList(Experimentv1State, states),
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
          { signal: canceler.current.signal },
        );
      }

      const pageNumber = settings.tableOffset / settings.tableLimit;
      const rowsTakenUpByPins = pageNumber * pinnedIds.length;
      const otherExpResponse = await getExperiments(
        {
          ...baseParams,
          experimentIdFilter: { notIn: pinnedIds },
          limit: settings.tableLimit - pinnedIds.length,
          offset: settings.tableOffset - rowsTakenUpByPins,
        },
        { signal: canceler.current.signal },
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
  }, [id, isLoadingSettings, settings, labelsString, pinnedString, statesString, usersString]);

  const fetchLabels = useCallback(async () => {
    try {
      const labels = await getExperimentLabels(
        { project_id: id },
        { signal: canceler.current.signal },
      );
      labels.sort((a, b) => alphaNumericSorter(a, b));
      setLabels(labels);
    } catch (e) {
      handleError(e);
    }
  }, [id]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchExperiments(), fetchLabels()]);
  }, [fetchExperiments, fetchLabels]);

  const { stopPolling } = usePolling(fetchAll, { rerunOnNewFn: true });

  const experimentTags = useExperimentTags(fetchAll);

  const handleActionComplete = useCallback(() => fetchExperiments(), [fetchExperiments]);

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
      if (!settings) return <Spinner spinning />;
      return (
        <ExperimentActionDropdown
          experiment={getProjectExperimentForExperimentItem(record, project)}
          isContextMenu
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
            tooltip: <Tags disabled tags={record.labels} />,
          }}>
          <div>
            <Tags
              compact
              disabled={record.archived || project?.archived || !canEditExperiment}
              tags={record.labels}
              onAction={experimentTags.handleTagListChange(record.id, record.labels)}
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
        onBlur={(e) => {
          const newDesc = e.currentTarget.value;
          saveExperimentDescription(newDesc, record.id);
        }}
        onPressEnter={(e) => {
          // when enter is pressed,
          // input box gets blurred and then value will be saved in onBlur
          e.currentTarget.blur();
        }}
      />
    );

    const forkedFromRenderer = (value: string | number | undefined): React.ReactNode =>
      value ? <Link path={paths.experimentDetails(value)}>{value}</Link> : null;

    const checkpointSizeRenderer = (value: number) => (value ? humanReadableBytes(value) : '');

    return [
      {
        align: 'right',
        dataIndex: 'id',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['id'],
        key: V1GetExperimentsRequestSortBy.ID,
        onCell: onRightClickableCell,
        render: experimentNameRenderer,
        sorter: true,
        title: 'ID',
      },
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        isFiltered: (settings: ExperimentListSettings) => !!settings.search,
        key: V1GetExperimentsRequestSortBy.NAME,
        onCell: onRightClickableCell,
        render: experimentNameRenderer,
        sorter: true,
        title: 'Name',
      },
      {
        dataIndex: 'description',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['description'],
        onCell: onRightClickableCell,
        render: descriptionRenderer,
        title: 'Description',
      },
      {
        dataIndex: 'tags',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['tags'],
        filterDropdown: labelFilterDropdown,
        filters: labels.map((label) => ({ text: label, value: label })),
        isFiltered: (settings: ExperimentListSettings) => !!settings.label,
        key: 'labels',
        render: tagsRenderer,
        title: 'Tags',
      },
      {
        align: 'right',
        dataIndex: 'forkedFrom',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['forkedFrom'],
        key: V1GetExperimentsRequestSortBy.FORKEDFROM,
        onCell: onRightClickableCell,
        render: forkedFromRenderer,
        sorter: true,
        title: 'Forked',
      },
      {
        align: 'right',
        dataIndex: 'startTime',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['startTime'],
        key: V1GetExperimentsRequestSortBy.STARTTIME,
        onCell: onRightClickableCell,
        render: (_: number, record: ExperimentItem): React.ReactNode =>
          relativeTimeRenderer(new Date(record.startTime)),
        sorter: true,
        title: 'Started',
      },
      {
        align: 'right',
        dataIndex: 'duration',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['duration'],
        key: 'duration',
        onCell: onRightClickableCell,
        render: experimentDurationRenderer,
        title: 'Duration',
      },
      {
        align: 'right',
        dataIndex: 'numTrials',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['numTrials'],
        key: V1GetExperimentsRequestSortBy.NUMTRIALS,
        onCell: onRightClickableCell,
        sorter: true,
        title: 'Trials',
      },
      {
        dataIndex: 'state',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['state'],
        filterDropdown: stateFilterDropdown,
        filters: [
          RunState.Active,
          RunState.Paused,
          RunState.Canceled,
          RunState.Completed,
          RunState.Error,
        ].map((value) => ({
          text: <Badge state={value} type={BadgeType.State} />,
          value,
        })),
        isFiltered: () => !!settings.state,
        key: V1GetExperimentsRequestSortBy.STATE,
        render: expStateRenderer,
        sorter: true,
        title: 'State',
      },
      {
        dataIndex: 'searcherType',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['searcherType'],
        key: 'searcherType',
        onCell: onRightClickableCell,
        title: 'Searcher',
      },
      {
        dataIndex: 'resourcePool',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['resourcePool'],
        key: V1GetExperimentsRequestSortBy.RESOURCEPOOL,
        onCell: onRightClickableCell,
        sorter: true,
        title: 'Resource Pool',
      },
      {
        align: 'right',
        dataIndex: 'progress',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['progress'],
        key: V1GetExperimentsRequestSortBy.PROGRESS,
        render: experimentProgressRenderer,
        sorter: true,
        title: 'Progress',
      },
      {
        align: 'right',
        dataIndex: 'checkpointSize',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['checkpointSize'],
        key: V1GetExperimentsRequestSortBy.CHECKPOINTSIZE,
        render: checkpointSizeRenderer,
        sorter: true,
        title: 'Checkpoint Size',
      },
      {
        align: 'right',
        dataIndex: 'checkpointCount',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['checkpointCount'],
        key: V1GetExperimentsRequestSortBy.CHECKPOINTCOUNT,
        sorter: true,
        title: 'Checkpoint Count',
      },
      {
        align: 'right',
        dataIndex: 'archived',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['archived'],
        key: 'archived',
        render: checkmarkRenderer,
        title: 'Archived',
      },
      {
        dataIndex: 'user',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['user'],
        filterDropdown: userFilterDropdown,
        filters: users.map((user) => ({ text: getDisplayName(user), value: user.id })),
        isFiltered: (settings: ExperimentListSettings) => !!settings.user,
        key: V1GetExperimentsRequestSortBy.USER,
        render: (_, r) => userRenderer(users.find((u) => u.id === r.userId)),
        sorter: true,
        title: 'User',
      },
      {
        align: 'right',
        className: 'fullCell',
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        fixed: 'right',
        key: 'action',
        onCell: onRightClickableCell,
        render: actionRenderer,
        title: '',
        width: DEFAULT_COLUMN_WIDTHS['action'],
      },
      {
        dataIndex: 'searcherMetricValue',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['searcherMetricValue'],
        key: V1GetExperimentsRequestSortBy.SEARCHERMETRICVAL,
        render: (_: string, record: ExperimentItem) => (
          <HumanReadableNumber num={record.searcherMetricValue} />
        ),
        sorter: true,
        title: 'Searcher Metric Value',
        width: DEFAULT_COLUMN_WIDTHS['searcherMetricValue'],
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

  const ColumnsCustomizeModal = useModal(ColumnsCustomizeModalComponent);

  useLayoutEffect(() => {
    // This is the failsafe for when column settings get into a bad shape.
    if (!settings.columns?.length || !settings.columnWidths?.length) {
      updateSettings({
        columns: DEFAULT_COLUMNS,
        columnWidths: DEFAULT_COLUMNS.map((columnName) => DEFAULT_COLUMN_WIDTHS[columnName]),
      });
    } else {
      const columnNames = columns.map((column) => column.dataIndex as ExperimentColumnName);
      const actualColumns = settings.columns.filter((name) => columnNames.includes(name));
      const newSettings: Partial<ExperimentListSettings> = {};
      if (actualColumns.length < settings.columns.length) {
        newSettings.columns = actualColumns;
      }
      if (settings.columnWidths.length !== actualColumns.length) {
        newSettings.columnWidths = actualColumns.map((name) => DEFAULT_COLUMN_WIDTHS[name]);
      }
      if (Object.keys(newSettings).length !== 0) updateSettings(newSettings);
    }
  }, [settings.columns, settings.columnWidths, columns, resetSettings, updateSettings]);

  const transferColumns = useMemo(() => {
    return columns
      .filter(
        (column) => column.title !== '' && column.title !== 'Action' && column.title !== 'Archived',
      )
      .map((column) => column.dataIndex?.toString() ?? '');
  }, [columns]);

  const initialVisibleColumns = useMemo(
    () => settings.columns?.filter((col) => transferColumns.includes(col)),
    [settings.columns, transferColumns],
  );

  const BatchActionConfirmModal = useModal(BatchActionConfirmModalComponent);
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);

  const sendBatchActions = useCallback(
    (action: Action): Promise<void[] | CommandTask | CommandResponse> | void => {
      if (!settings.row) return;
      if (action === Action.OpenTensorBoard) {
        return openOrCreateTensorBoard({
          experimentIds: settings.row,
          workspaceId: project?.workspaceId,
        });
      }
      if (action === Action.Move) {
        if (!settings?.row?.length) return;
        setBatchMovingExperimentIds(
          settings.row.filter(
            (id) =>
              canActionExperiment(Action.Move, experimentMap[id]) &&
              permissions.canMoveExperiment({ experiment: experimentMap[id] }),
          ),
        );
        ExperimentMoveModal.open();
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
    [settings.row, experimentMap, permissions, ExperimentMoveModal, project?.workspaceId],
  );

  const submitBatchAction = useCallback(
    async (action: Action) => {
      try {
        const result = await sendBatchActions(action);
        if (action === Action.OpenTensorBoard && result) {
          openCommandResponse(result as CommandResponse);
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
    [fetchExperiments, sendBatchActions, updateSettings],
  );

  const handleBatchAction = useCallback(
    (action?: string) => {
      if (action === Action.OpenTensorBoard || action === Action.Move) {
        submitBatchAction(action);
      } else {
        setBatchAction(action as Action);
        BatchActionConfirmModal.open();
      }
    },
    [BatchActionConfirmModal, submitBatchAction],
  );

  const handleTableRowSelect = useCallback(
    (rowKeys: unknown) => {
      updateSettings({ row: rowKeys as number[] });
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
          columnWidths: [DEFAULT_COLUMN_WIDTHS['id'], DEFAULT_COLUMN_WIDTHS['name']],
        });
      } else {
        updateSettings({
          columns: columns,
          columnWidths: columns.map((col) => DEFAULT_COLUMN_WIDTHS[col]),
        });
      }
    },
    [updateSettings],
  );

  const switchShowArchived = useCallback(
    (showArchived: boolean) => {
      if (!settings) return;
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
    setIsLoading(true);
    fetchExperiments();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    return () => stopPolling();
  }, [stopPolling]);

  useEffect(() => {
    const currentCanceler = canceler.current;
    return () => currentCanceler.abort();
  }, []);

  const tabBarContent = useMemo(() => {
    const menuItems: MenuItem[] = [
      {
        key: MenuKey.SwitchArchived,
        label: settings.archived ? 'Hide Archived' : 'Show Archived',
      },
      { key: MenuKey.Columns, label: 'Columns' },
    ];
    if (filterCount > 0) {
      menuItems.push({ key: MenuKey.ResultFilter, label: `Clear Filters (${filterCount})` });
    }
    const handleDropdown = (key: string) => {
      switch (key) {
        case MenuKey.Columns:
          ColumnsCustomizeModal.open();
          break;
        case MenuKey.ResultFilter:
          resetFilters();
          break;
        case MenuKey.SwitchArchived:
          switchShowArchived(!settings.archived);
          break;
      }
    };

    return (
      <div className={css.tabOptions}>
        <Space className={css.actionList}>
          <Toggle checked={settings.archived} label="Show Archived" onChange={switchShowArchived} />
          <Button onClick={ColumnsCustomizeModal.open}>Columns</Button>
          <FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />
        </Space>
        <div className={css.actionOverflow} title="Open actions menu">
          <Dropdown menu={menuItems} onClick={handleDropdown}>
            <div>
              <Icon name="overflow-vertical" title="Action menu" />
            </div>
          </Dropdown>
        </div>
      </div>
    );
  }, [filterCount, ColumnsCustomizeModal, resetFilters, settings.archived, switchShowArchived]);

  useSetDynamicTabBar(tabBarContent);
  return (
    <>
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
      <InteractiveTable<ExperimentItem, ExperimentListSettings>
        areRowsSelected={!!settings.row}
        columns={columns}
        containerRef={pageRef}
        ContextMenu={ContextMenu}
        dataSource={experiments}
        loading={isLoading}
        numOfPinned={(settings.pinned?.[id] ?? []).length}
        pagination={getFullPaginationConfig(
          {
            limit: settings.tableLimit || 0,
            offset: settings.tableOffset || 0,
          },
          total,
        )}
        rowClassName={defaultRowClassName({ clickable: false })}
        rowKey="id"
        rowSelection={{
          onChange: handleTableRowSelect,
          preserveSelectedRowKeys: true,
          selectedRowKeys: settings.row ?? [],
        }}
        scroll={{ y: `calc(100vh - ${availableBatchActions.length === 0 ? '230' : '280'}px)` }}
        settings={settings}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings}
      />
      {batchAction && (
        <BatchActionConfirmModal.Component
          batchAction={batchAction}
          onConfirm={() => submitBatchAction(batchAction)}
        />
      )}
      <ColumnsCustomizeModal.Component
        columns={transferColumns}
        defaultVisibleColumns={DEFAULT_COLUMNS}
        initialVisibleColumns={initialVisibleColumns}
        onSave={handleUpdateColumns as (columns: string[]) => void}
      />
      <ExperimentMoveModal.Component
        experimentIds={batchMovingExperimentIds ?? []}
        sourceProjectId={project?.id}
        sourceWorkspaceId={project?.workspaceId}
        onSubmit={handleActionComplete}
      />
    </>
  );
};

export default ExperimentList;
