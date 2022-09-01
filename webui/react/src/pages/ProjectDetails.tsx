import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Dropdown, Menu, Modal, Space } from 'antd';
import type { MenuProps } from 'antd';
import { FilterDropdownProps } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';

import Badge, { BadgeType } from 'components/Badge';
import ExperimentActionDropdown from 'components/ExperimentActionDropdown';
import FilterCounter from 'components/FilterCounter';
import InlineEditor from 'components/InlineEditor';
import InteractiveTable, { ColumnDef,
  InteractiveTableSettings,
  onRightClickableCell } from 'components/InteractiveTable';
import Link from 'components/Link';
import Page from 'components/Page';
import PaginatedNotesCard from 'components/PaginatedNotesCard';
import { checkmarkRenderer, defaultRowClassName, experimentDurationRenderer,
  experimentNameRenderer, experimentProgressRenderer, ExperimentRenderer,
  getFullPaginationConfig, relativeTimeRenderer, stateRenderer,
  userRenderer } from 'components/Table';
import TableBatch from 'components/TableBatch';
import TableFilterDropdown from 'components/TableFilterDropdown';
import TableFilterSearch from 'components/TableFilterSearch';
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
import usePolling from 'hooks/usePolling';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import {
  activateExperiment, addProjectNote, archiveExperiment, cancelExperiment, deleteExperiment,
  getExperimentLabels, getExperiments, getProject, killExperiment, openOrCreateTensorBoard,
  patchExperiment, pauseExperiment, setProjectNotes, unarchiveExperiment,
} from 'services/api';
import { Determinedexperimentv1State, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import Icon from 'shared/components/Icon/Icon';
import Message, { MessageType } from 'shared/components/Message';
import Spinner from 'shared/components/Spinner';
import { RecordKey } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { ErrorLevel } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { isNotFound } from 'shared/utils/service';
import { validateDetApiEnum, validateDetApiEnumList } from 'shared/utils/service';
import { alphaNumericSorter } from 'shared/utils/sort';
import {
  ExperimentAction as Action,
  CommandTask,
  ExperimentItem,
  Note,
  Project,
  ProjectExperiment,
  RunState,
} from 'types';
import handleError from 'utils/error';
import {
  canUserActionExperiment,
  getActionsForExperimentsUnion,
  getProjectExperimentForExperimentItem,
} from 'utils/experiment';
import { getDisplayName } from 'utils/user';
import { openCommand } from 'utils/wait';

import css from './ProjectDetails.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, DEFAULT_COLUMNS,
  ExperimentColumnName, ProjectDetailsSettings } from './ProjectDetails.settings';
import ProjectDetailsTabs, { TabInfo } from './ProjectDetails/ProjectDetailsTabs';

const filterKeys: Array<keyof ProjectDetailsSettings> = [ 'label', 'search', 'state', 'user' ];

interface Params {
  projectId: string;
}

const batchActions = [
  Action.CompareExperiments,
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
  const { users, auth: { user } } = useStore();
  const { projectId } = useParams<Params>();
  const [ project, setProject ] = useState<Project>();
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>([]);
  const [ labels, setLabels ] = useState<string[]>([]);
  const [ pageError, setPageError ] = useState<Error>();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ total, setTotal ] = useState(0);
  const [ canceler ] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const { updateSettings: updateDestinationSettings } = useSettings<MoveExperimentSettings>(
    moveExperimentSettingsConfig,
  );

  useEffect(() => {
    updateDestinationSettings({ projectId: undefined, workspaceId: project?.workspaceId });
  }, [ updateDestinationSettings, project?.workspaceId ]);

  const id = parseInt(projectId);

  const {
    settings,
    updateSettings,
    resetSettings,
    activeSettings,
  } = useSettings<ProjectDetailsSettings>(settingsConfig);

  const experimentMap = useMemo(() => {
    return (experiments || []).reduce((acc, experiment) => {
      acc[experiment.id] = getProjectExperimentForExperimentItem(experiment, project);
      return acc;
    }, {} as Record<RecordKey, ProjectExperiment>);
  }, [
    experiments,
    project,
  ]);

  const filterCount = useMemo(() => activeSettings(filterKeys).length, [ activeSettings ]);

  const availableBatchActions = useMemo(() => {
    const experiments = settings.row?.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, batchActions, user);
  }, [ experimentMap, settings.row, user ]);

  const fetchProject = useCallback(async () => {
    try {
      const response = await getProject({ id }, { signal: canceler.signal });
      setProject((prev) => {
        if (isEqual(prev, response)) return prev;
        return response;
      });
    } catch (e) {
      if (!pageError) setPageError(e as Error);
    } finally {
      setIsLoading(false);
    }
  }, [ canceler.signal, id, pageError ]);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = (settings.state || []).map((state) => (
        encodeExperimentState(state as RunState)
      ));
      const response = await getExperiments(
        {
          archived: settings.archived ? undefined : false,
          labels: settings.label,
          limit: settings.tableLimit,
          name: settings.search,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          projectId: id,
          sortBy: validateDetApiEnum(V1GetExperimentsRequestSortBy, settings.sortKey),
          states: validateDetApiEnumList(Determinedexperimentv1State, states),
          users: settings.user,
        },
        { signal: canceler.signal },
      );
      setTotal(response.pagination.total ?? 0);
      setExperiments((prev) => {
        if (isEqual(prev, response.experiments)) return prev;
        return response.experiments;
      });
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
  }, [ canceler.signal,
    id,
    settings.archived,
    settings.label,
    settings.search,
    settings.sortDesc,
    settings.sortKey,
    settings.state,
    settings.tableLimit,
    settings.tableOffset,
    settings.user ]);

  const fetchLabels = useCallback(async () => {
    try {
      const labels = await getExperimentLabels({ project_id: id }, { signal: canceler.signal });
      labels.sort((a, b) => alphaNumericSorter(a, b));
      setLabels(labels);
    } catch (e) { handleError(e); }
  }, [ canceler.signal, id ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([
      fetchProject(), fetchExperiments(), fetchUsers(), fetchLabels() ]);
  }, [ fetchProject, fetchExperiments, fetchUsers, fetchLabels ]);

  usePolling(fetchAll, { rerunOnNewFn: true });

  const experimentTags = useExperimentTags(fetchAll);

  const handleActionComplete = useCallback(() => fetchExperiments(), [ fetchExperiments ]);

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

  const handleLabelFilterApply = useCallback((labels: string[]) => {
    updateSettings({
      label: labels.length !== 0 ? labels : undefined,
      row: undefined,
    });
  }, [ updateSettings ]);

  const handleLabelFilterReset = useCallback(() => {
    updateSettings({ label: undefined, row: undefined });
  }, [ updateSettings ]);

  const labelFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      searchable
      values={settings.label}
      onFilter={handleLabelFilterApply}
      onReset={handleLabelFilterReset}
    />
  ), [ handleLabelFilterApply, handleLabelFilterReset, settings.label ]);

  const handleStateFilterApply = useCallback((states: string[]) => {
    updateSettings({
      row: undefined,
      state: states.length !== 0 ? states as RunState[] : undefined,
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
      onReset={handleStateFilterReset}
    />
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
      onReset={handleUserFilterReset}
    />
  ), [ handleUserFilterApply, handleUserFilterReset, settings.user ]);

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
      setIsLoading(false);
      return e as Error;
    }
  }, [ ]);

  const columns = useMemo(() => {
    const tagsRenderer = (value: string, record: ExperimentItem) => (
      <TagList
        disabled={record.archived || project?.archived}
        tags={record.labels}
        onChange={experimentTags.handleTagListChange(record.id)}
      />
    );

    const actionRenderer: ExperimentRenderer = (_, record) => {
      return (
        <ExperimentActionDropdown
          curUser={user}
          experiment={getProjectExperimentForExperimentItem(record, project)}
          onComplete={handleActionComplete}
        />
      );
    };

    const descriptionRenderer = (value:string, record: ExperimentItem) => (
      <InlineEditor
        disabled={record.archived}
        maxLength={500}
        placeholder={record.archived ? 'Archived' : 'Add description...'}
        value={value}
        onSave={(newDescription: string) => saveExperimentDescription(newDescription, record.id)}
      />
    );

    const forkedFromRenderer = (
      value: string | number | undefined,
    ): React.ReactNode => (
      value ? <Link path={paths.experimentDetails(value)}>{value}</Link> : null
    );

    return [
      {
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
        onCell: onRightClickableCell,
        render: tagsRenderer,
        title: 'Tags',
      },
      {
        dataIndex: 'forkedFrom',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['forkedFrom'],
        key: V1GetExperimentsRequestSortBy.FORKEDFROM,
        onCell: onRightClickableCell,
        render: forkedFromRenderer,
        sorter: true,
        title: 'Forked From',
      },
      {
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
        dataIndex: 'duration',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['duration'],
        key: 'duration',
        onCell: onRightClickableCell,
        render: experimentDurationRenderer,
        title: 'Duration',
      },
      {
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
        filters: Object.values(RunState)
          .filter((value) => [
            RunState.Active,
            RunState.Paused,
            RunState.Canceled,
            RunState.Completed,
            RunState.Errored,
          ].includes(value))
          .map((value) => ({
            text: <Badge state={value} type={BadgeType.State} />,
            value,
          })),
        isFiltered: () => !!settings.state,
        key: V1GetExperimentsRequestSortBy.STATE,
        render: stateRenderer,
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
        dataIndex: 'progress',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['progress'],
        key: V1GetExperimentsRequestSortBy.PROGRESS,
        render: experimentProgressRenderer,
        sorter: true,
        title: 'Progress',
      },
      {
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
    user,
    handleActionComplete,
    experimentTags,
    labelFilterDropdown,
    labels,
    nameFilterSearch,
    saveExperimentDescription,
    settings,
    project,
    stateFilterDropdown,
    tableSearchIcon,
    userFilterDropdown,
    users,
  ]);

  useEffect(() => {
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
  }, [ settings.columns, settings.columnWidths, columns, resetSettings, updateSettings ]);

  const transferColumns = useMemo(() => {
    return columns
      .filter(
        (column) => column.title !== '' && column.title !== 'Action' && column.title !== 'Archived',
      )
      .map((column) => column.dataIndex?.toString() ?? '');
  }, [ columns ]);

  const {
    contextHolder: modalExperimentMoveContextHolder,
    modalOpen: openMoveModal,
  } = useModalExperimentMove({ onClose: handleActionComplete, user });

  const sendBatchActions = useCallback((action: Action): Promise<void[] | CommandTask> | void => {
    if (!settings.row) return;
    if (action === Action.OpenTensorBoard) {
      return openOrCreateTensorBoard({ experimentIds: settings.row });
    }
    if (action === Action.Move) {
      return openMoveModal({
        experimentIds: settings.row.filter((id) =>
          canUserActionExperiment(user, Action.Move, experimentMap[id])),
        sourceProjectId: project?.id,
        sourceWorkspaceId: project?.workspaceId,
      });
    }
    if (action === Action.CompareExperiments) {
      if (settings.row?.length)
        return routeToReactUrl(paths.experimentComparison(settings.row.map((id) => id.toString())));
    }

    return Promise.all((settings.row || []).map((experimentId) => {
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
    }));
  }, [ settings.row, openMoveModal, project?.workspaceId, project?.id, experimentMap, user ]);

  const submitBatchAction = useCallback(async (action: Action) => {
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
      const publicSubject = action === Action.OpenTensorBoard ?
        'Unable to View TensorBoard for Selected Experiments' :
        `Unable to ${action} Selected Experiments`;
      handleError(e, {
        isUserTriggered: true,
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject,
        silent: false,
      });
    }
  }, [ fetchExperiments, sendBatchActions, updateSettings ]);

  const showConfirmation = useCallback((action: Action) => {
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
  }, [ submitBatchAction ]);

  const handleBatchAction = useCallback((action?: string) => {
    if (action === Action.OpenTensorBoard || action === Action.Move) {
      submitBatchAction(action);
    } else {
      showConfirmation(action as Action);
    }
  }, [ submitBatchAction, showConfirmation ]);

  const handleTableRowSelect = useCallback((rowKeys) => {
    updateSettings({ row: rowKeys });
  }, [ updateSettings ]);

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [ updateSettings ]);

  const resetFilters = useCallback(() => {
    resetSettings([ ...filterKeys, 'tableOffset' ]);
    clearSelected();
  }, [ clearSelected, resetSettings ]);

  const handleUpdateColumns = useCallback((columns: ExperimentColumnName[]) => {
    if (columns.length === 0) {
      updateSettings({
        columns: [ 'id', 'name' ],
        columnWidths: [
          DEFAULT_COLUMN_WIDTHS['id'],
          DEFAULT_COLUMN_WIDTHS['name'],
        ],
      });
    } else {
      updateSettings({
        columns: columns,
        columnWidths: columns.map((col) => DEFAULT_COLUMN_WIDTHS[col]),
      });
    }
  }, [ updateSettings ]);

  const {
    contextHolder: modalColumnsCustomizeContextHolder,
    modalOpen: openCustomizeColumns,
  } = useModalColumnsCustomize({
    columns: transferColumns,
    defaultVisibleColumns: DEFAULT_COLUMNS,
    onSave: (handleUpdateColumns as (columns: string[]) => void),
  });

  const handleCustomizeColumnsClick = useCallback(() => {
    openCustomizeColumns(
      { initialVisibleColumns: settings.columns?.filter((col) => transferColumns.includes(col)) },
    );
  }, [ openCustomizeColumns, settings.columns, transferColumns ]);

  const switchShowArchived = useCallback((showArchived: boolean) => {
    let newColumns: ExperimentColumnName[];
    let newColumnWidths: number[];

    if (showArchived) {
      if (settings.columns?.includes('archived')) {
        // just some defensive coding: don't add archived twice
        newColumns = settings.columns;
        newColumnWidths = settings.columnWidths;
      } else {
        newColumns = [ ...settings.columns, 'archived' ];
        newColumnWidths = [ ...settings.columnWidths, DEFAULT_COLUMN_WIDTHS['archived'] ];
      }
    } else {
      const archivedIndex = settings.columns.indexOf('archived');
      if (archivedIndex !== -1) {
        newColumns = [ ...settings.columns ];
        newColumnWidths = [ ...settings.columnWidths ];
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

  }, [ settings, updateSettings ]);

  const handleNewNotesPage = useCallback(async () => {
    if (!project?.id) return;
    try {
      await addProjectNote({ contents: '', id: project.id, name: 'Untitled' });
      await fetchProject();
    } catch (e) { handleError(e); }
  }, [ fetchProject, project?.id ]);

  const handleSaveNotes = useCallback(async (notes: Note[]) => {
    if (!project?.id) return;
    try {
      await setProjectNotes({ notes, projectId: project.id });
      await fetchProject();
    } catch (e) { handleError(e); }
  }, [ fetchProject, project?.id ]);

  const {
    contextHolder: modalProjectNodeDeleteContextHolder,
    modalOpen: openNoteDelete,
  } = useModalProjectNoteDelete({ onClose: fetchProject, project });

  const handleDeleteNote = useCallback((pageNumber: number) => {
    if (!project?.id) return;
    try {
      openNoteDelete({ pageNumber });
    } catch (e) { handleError(e); }
  }, [ openNoteDelete, project?.id ]);

  useEffect(() => {
    if (settings.tableOffset >= total && total){
      const newTotal = settings.tableOffset > total ? total : total - 1;
      const offset = settings.tableLimit * Math.floor(newTotal / settings.tableLimit);
      updateSettings({ tableOffset: offset });
    }
  }, [ total, settings.tableOffset, settings.tableLimit, updateSettings ]);

  /*
   * Get new experiments based on changes to the
   * filters, pagination, search and sorter.
   */
  useEffect(() => {
    fetchExperiments();
    setIsLoading(true);
  }, [
    fetchExperiments,
    settings.archived,
    settings.label,
    settings.search,
    settings.sortDesc,
    settings.sortKey,
    settings.state,
    settings.tableLimit,
    settings.tableOffset,
    settings.user,
  ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const ContextMenu = useCallback(
    ({ record, onVisibleChange, children }) => {
      return (
        <ExperimentActionDropdown
          curUser={user}
          experiment={getProjectExperimentForExperimentItem(record, project)}
          onComplete={handleActionComplete}
          onVisibleChange={onVisibleChange}>
          {children}
        </ExperimentActionDropdown>
      );
    },
    [ user, handleActionComplete, project ],
  );

  const ExperimentTabOptions = useMemo(() => {
    const getMenuProps = ():{items: MenuProps['items'], onClick: MenuProps['onClick']} => {
      enum MenuKey {
        SWITCH_ARCHIVED = 'switchArchive',
        COLUMNS = 'columns',
        RESULT_FILTER = 'resetFilters',
      }

      const funcs = {
        [MenuKey.SWITCH_ARCHIVED]: () => { switchShowArchived(!settings.archived); },
        [MenuKey.COLUMNS]: () => { handleCustomizeColumnsClick(); },
        [MenuKey.RESULT_FILTER]: () => { resetFilters(); },
      };

      const onItemClick: MenuProps['onClick'] = (e) => {
        funcs[e.key as MenuKey]();
      };

      const menuItems: MenuProps['items'] = [
        {
          key: MenuKey.SWITCH_ARCHIVED,
          label: settings.archived ? 'Hide Archived' : 'Show Archived',
        },
        { key: MenuKey.COLUMNS, label: 'Columns' },
      ];
      if (filterCount > 0) {
        menuItems.push({ key: MenuKey.RESULT_FILTER, label: `Clear Filters (${filterCount})` });
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
            trigger={[ 'click' ]}>
            <div>
              <Icon name="overflow-vertical" />
            </div>
          </Dropdown>
        </div>
      </div>
    );
  }, [ filterCount,
    handleCustomizeColumnsClick,
    resetFilters,
    settings.archived,
    switchShowArchived ]);

  const tabs: TabInfo[] = useMemo(() => {
    return ([ {
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
            pagination={getFullPaginationConfig({
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            }, total)}
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
          />
        </div>
      ),
      options: ExperimentTabOptions,
      title: 'Experiments',
    }, {
      body: (
        <PaginatedNotesCard
          disabled={project?.archived}
          notes={project?.notes ?? []}
          onDelete={handleDeleteNote}
          onNewPage={handleNewNotesPage}
          onSave={handleSaveNotes}
        />),
      options: (
        <div className={css.tabOptions}>
          <Button type="text" onClick={handleNewNotesPage}>+ New Page</Button>
        </div>),
      title: 'Notes',
    } ]);
  }, [ ContextMenu,
    ExperimentTabOptions,
    clearSelected,
    columns,
    experiments,
    handleBatchAction,
    handleDeleteNote,
    handleNewNotesPage,
    handleSaveNotes,
    handleTableRowSelect,
    availableBatchActions,
    isLoading,
    project?.notes,
    project?.archived,
    settings,
    total,
    updateSettings ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Project ID ${projectId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find Project ${projectId}` :
      `Unable to fetch Project ${projectId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!project) {
    return (
      <Spinner
        tip={projectId === '1' ? 'Loading...' : `Loading project ${projectId} details...`}
      />
    );
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
