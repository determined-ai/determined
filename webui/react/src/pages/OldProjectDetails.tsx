import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Dropdown, Input, Menu, Modal, Space, Typography } from 'antd';
import type { MenuProps } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import Badge, { BadgeType } from 'components/Badge';
import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import FilterCounter from 'components/FilterCounter';
import Link from 'components/Link';
import Page from 'components/Page';
import PageNotFound from 'components/PageNotFound';
import PaginatedNotesCard from 'components/PaginatedNotesCard';
import InteractiveTable, {
  ColumnDef,
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
  expStateRenderer,
  getFullPaginationConfig,
  relativeTimeRenderer,
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
import useModalProjectNoteDelete from 'hooks/useModal/Project/useModalProjectNoteDelete';
import usePermissions from 'hooks/usePermissions';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import NoPermissions from 'pages/NoPermissions';
import { paths } from 'routes/utils';
import {
  activateExperiment,
  addProjectNote,
  archiveExperiment,
  cancelExperiment,
  deleteExperiment,
  getExperimentLabels,
  getExperiments,
  getProject,
  killExperiment,
  openOrCreateTensorBoard,
  patchExperiment,
  pauseExperiment,
  setProjectNotes,
  unarchiveExperiment,
} from 'services/api';
import { Determinedexperimentv1State, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { GetExperimentsParams } from 'services/types';
import Icon from 'shared/components/Icon/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import usePolling from 'shared/hooks/usePolling';
import { RecordKey, ValueOf } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel } from 'shared/utils/error';
import { isNotFound } from 'shared/utils/service';
import { validateDetApiEnum, validateDetApiEnumList } from 'shared/utils/service';
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  ExperimentAction as Action,
  CommandTask,
  ExperimentItem,
  ExperimentPagination,
  Note,
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
  ProjectDetailsSettings,
} from './OldProjectDetails.settings';
import css from './ProjectDetails.module.scss';
import ProjectDetailsTabs, { TabInfo } from './ProjectDetails/ProjectDetailsTabs';

const filterKeys: Array<keyof ProjectDetailsSettings> = ['label', 'search', 'state', 'user'];

type Params = {
  projectId: string;
};

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

const ProjectDetails: React.FC = () => {
  const {
    users,
    auth: { user },
  } = useStore();
  const { projectId } = useParams<Params>();
  const [project, setProject] = useState<Project>();
  const [experiments, setExperiments] = useState<ExperimentItem[]>([]);
  const [labels, setLabels] = useState<string[]>([]);
  const [pageError, setPageError] = useState<Error>();
  const [isLoading, setIsLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);
  const expPermissions = usePermissions();

  const { updateSettings: updateDestinationSettings } = useSettings<MoveExperimentSettings>(
    moveExperimentSettingsConfig,
  );

  useEffect(() => {
    updateDestinationSettings({ projectId: undefined, workspaceId: project?.workspaceId });
  }, [updateDestinationSettings, project?.workspaceId]);

  const id = parseInt(projectId ?? '1');

  const { settings, updateSettings, resetSettings, activeSettings } =
    useSettings<ProjectDetailsSettings>(settingsConfig);

  const experimentMap = useMemo(() => {
    return (experiments || []).reduce((acc, experiment) => {
      acc[experiment.id] = getProjectExperimentForExperimentItem(experiment, project);
      return acc;
    }, {} as Record<RecordKey, ProjectExperiment>);
  }, [experiments, project]);

  const filterCount = useMemo(() => activeSettings(filterKeys).length, [activeSettings]);

  const availableBatchActions = useMemo(() => {
    const experiments = settings.row?.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, batchActions, expPermissions);
  }, [experimentMap, expPermissions, settings.row]);

  const fetchProject = useCallback(async () => {
    try {
      const response = await getProject({ id }, { signal: canceler.signal });
      setProject((prev) => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
      setPageError(undefined);
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    }
  }, [canceler.signal, id, pageError]);

  const statesString = useMemo(() => settings.state?.join('.'), [settings.state]);
  const pinnedString = useMemo(() => JSON.stringify(settings.pinned), [settings.pinned]);
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
    await Promise.allSettled([fetchProject(), fetchExperiments(), fetchUsers(), fetchLabels()]);
  }, [fetchProject, fetchExperiments, fetchUsers, fetchLabels]);

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
    expPermissions.canModifyExperimentMetadata({
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
    [project, settings, updateSettings, handleActionComplete],
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
        isFiltered: (settings: ProjectDetailsSettings) => !!settings.search,
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
        isFiltered: (settings: ProjectDetailsSettings) => !!settings.label,
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
        title: 'Forked From',
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
        title: 'Start Time',
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
        title: 'Searcher Type',
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
        align: 'center',
        dataIndex: 'progress',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['progress'],
        key: V1GetExperimentsRequestSortBy.PROGRESS,
        render: experimentProgressRenderer,
        sorter: true,
        title: 'Progress',
      },
      {
        align: 'center',
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
        isFiltered: (settings: ProjectDetailsSettings) => !!settings.user,
        key: V1GetExperimentsRequestSortBy.USER,
        render: userRenderer,
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
    ] as ColumnDef<ExperimentItem>[];
  }, [
    nameFilterSearch,
    tableSearchIcon,
    labelFilterDropdown,
    labels,
    stateFilterDropdown,
    userFilterDropdown,
    users,
    project,
    experimentTags,
    canEditExperiment,
    settings,
    saveExperimentDescription,
    ContextMenu,
  ]);

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
      const newSettings: Partial<ProjectDetailsSettings> = {};
      if (actualColumns.length < settings.columns.length) {
        newSettings.columns = actualColumns;
      }
      if (settings.columnWidths.length !== actualColumns.length) {
        newSettings.columnWidths = actualColumns.map((name) => DEFAULT_COLUMN_WIDTHS[name]);
      }
      if (Object.keys(newSettings).length !== 0) updateSettings(newSettings);
    }
  }, [settings.columns, settings.columnWidths, columns, updateSettings]);

  const transferColumns = useMemo(() => {
    return columns
      .filter(
        (column) => column.title !== '' && column.title !== 'Action' && column.title !== 'Archived',
      )
      .map((column) => column.dataIndex?.toString() ?? '');
  }, [columns]);

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
              expPermissions.canMoveExperiment({ experiment: experimentMap[id] }),
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
    [expPermissions, settings.row, openMoveModal, project?.workspaceId, project?.id, experimentMap],
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
    [fetchExperiments, sendBatchActions, updateSettings],
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

  const handleNewNotesPage = useCallback(async () => {
    if (!project?.id) return;
    try {
      await addProjectNote({ contents: '', id: project.id, name: 'Untitled' });
      await fetchProject();
    } catch (e) {
      handleError(e);
    }
  }, [fetchProject, project?.id]);

  const handleSaveNotes = useCallback(
    async (notes: Note[]) => {
      if (!project?.id) return;
      try {
        await setProjectNotes({ notes, projectId: project.id });
        await fetchProject();
      } catch (e) {
        handleError(e);
      }
    },
    [fetchProject, project?.id],
  );

  const { contextHolder: modalProjectNodeDeleteContextHolder, modalOpen: openNoteDelete } =
    useModalProjectNoteDelete({ onClose: fetchProject, project });

  const handleDeleteNote = useCallback(
    (pageNumber: number) => {
      if (!project?.id) return;
      try {
        openNoteDelete({ pageNumber });
      } catch (e) {
        handleError(e);
      }
    },
    [openNoteDelete, project?.id],
  );

  useEffect(() => {
    if (settings.tableOffset >= total && total) {
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

      setProject(undefined);
      setExperiments([]);
      setLabels([]);
      setIsLoading(true);
      setTotal(0);
    };
  }, [canceler, stopPolling]);

  const ExperimentTabOptions = useMemo(() => {
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
          <Dropdown
            overlay={<Menu {...getMenuProps()} />}
            placement="bottomRight"
            trigger={['click']}>
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

  const tabs: TabInfo[] = useMemo(() => {
    return [
      {
        body: (
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
            <InteractiveTable
              areRowsSelected={!!settings.row}
              columns={columns}
              containerRef={pageRef}
              ContextMenu={ContextMenu}
              dataSource={experiments}
              loading={isLoading}
              numOfPinned={(settings.pinned?.[id] ?? []).length}
              pagination={getFullPaginationConfig(
                {
                  limit: settings.tableLimit,
                  offset: settings.tableOffset,
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
              scroll={{
                y: `calc(100vh - ${availableBatchActions.length === 0 ? '230' : '280'}px)`,
              }}
              settings={settings as InteractiveTableSettings}
              showSorterTooltip={false}
              size="small"
              updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
            />
          </div>
        ),
        options: ExperimentTabOptions,
        title: 'Experiments',
      },
      {
        body: (
          <PaginatedNotesCard
            disabled={project?.archived}
            notes={project?.notes ?? []}
            onDelete={handleDeleteNote}
            onNewPage={handleNewNotesPage}
            onSave={handleSaveNotes}
          />
        ),
        options: (
          <div className={css.tabOptions}>
            <Button type="text" onClick={handleNewNotesPage}>
              + New Page
            </Button>
          </div>
        ),
        title: 'Notes',
      },
    ];
  }, [
    settings,
    handleBatchAction,
    clearSelected,
    columns,
    ContextMenu,
    experiments,
    isLoading,
    id,
    total,
    handleTableRowSelect,
    availableBatchActions,
    updateSettings,
    ExperimentTabOptions,
    project?.archived,
    project?.notes,
    handleDeleteNote,
    handleNewNotesPage,
    handleSaveNotes,
  ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Project ID ${projectId}`} />;
  } else if (!expPermissions.canViewWorkspaces) {
    return <NoPermissions />;
  } else if (pageError) {
    if (isNotFound(pageError)) return <PageNotFound />;
    const message = `Unable to fetch Project ${projectId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!project) {
    return (
      <Spinner tip={projectId === '1' ? 'Loading...' : `Loading project ${projectId} details...`} />
    );
  }

  if (project && !expPermissions.canViewWorkspace({ workspace: { id: project.workspaceId } })) {
    return <PageNotFound />;
  }

  return (
    <Page
      bodyNoPadding
      containerRef={pageRef}
      // for docTitle, when id is 1 that means Uncategorized from webui/react/src/routes/routes.ts
      docTitle={id === 1 ? 'Uncategorized Experiments' : 'Project Details'}
      id="projectDetails">
      <ProjectDetailsTabs
        curUser={user}
        fetchProject={fetchProject}
        project={project}
        tabs={tabs}
      />
      {modalColumnsCustomizeContextHolder}
      {modalExperimentMoveContextHolder}
      {modalProjectNodeDeleteContextHolder}
    </Page>
  );
};

export default ProjectDetails;
