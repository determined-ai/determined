import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Modal } from 'antd';
import { ColumnsType, FilterDropdownProps, SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import FilterCounter from 'components/FilterCounter';
import Icon from 'components/Icon';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import tableCss from 'components/ResponsiveTable.module.scss';
import {
  archivedRenderer, defaultRowClassName, experimentNameRenderer, experimentProgressRenderer,
  ExperimentRenderer, expermentDurationRenderer, getFullPaginationConfig,
  relativeTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import TableBatch from 'components/TableBatch';
import TableFilterDropdown from 'components/TableFilterDropdown';
import TableFilterSearch from 'components/TableFilterSearch';
import TagList from 'components/TagList';
import TaskActionDropdown from 'components/TaskActionDropdown';
import { useStore } from 'contexts/Store';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import useExperimentTags from 'hooks/useExperimentTags';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import {
  activateExperiment, archiveExperiment, cancelExperiment, getExperimentLabels, getExperiments,
  killExperiment, openOrCreateTensorboard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { Determinedexperimentv1State, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { validateDetApiEnum, validateDetApiEnumList } from 'services/utils';
import {
  ExperimentAction as Action, ArchiveFilter, CommandTask, ExperimentItem, RecordKey, RunState,
} from 'types';
import { isBoolean, isEqual } from 'utils/data';
import { alphanumericSorter } from 'utils/sort';
import { capitalize } from 'utils/string';
import {
  cancellableRunStates, experimentToTask, isTaskKillable, terminalRunStates,
} from 'utils/types';
import { openCommand } from 'wait';

import settingsConfig, { Settings } from './ExperimentList.settings';

const filterKeys: Array<keyof Settings> = [ 'archived', 'label', 'search', 'state', 'user' ];

const ExperimentList: React.FC = () => {
  const { users, auth: { user } } = useStore();
  const [ canceler ] = useState(new AbortController());
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>();
  const [ labels, setLabels ] = useState<string[]>([]);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ total, setTotal ] = useState(0);

  const {
    activeSettings,
    resetSettings,
    settings,
    updateSettings,
  } = useSettings<Settings>(settingsConfig);

  const experimentMap = useMemo(() => {
    return (experiments || []).reduce((acc, experiment) => {
      acc[experiment.id] = experiment;
      return acc;
    }, {} as Record<RecordKey, ExperimentItem>);
  }, [ experiments ]);

  const filterCount = useMemo(() => activeSettings(filterKeys).length, [ activeSettings ]);

  const {
    hasActivatable,
    hasArchivable,
    hasCancelable,
    hasKillable,
    hasPausable,
    hasUnarchivable,
  } = useMemo(() => {
    const tracker = {
      hasActivatable: false,
      hasArchivable: false,
      hasCancelable: false,
      hasKillable: false,
      hasPausable: false,
      hasUnarchivable: false,
    };
    for (const id of settings.row || []) {
      const experiment = experimentMap[id];
      if (!experiment) continue;
      const isArchivable = !experiment.archived && terminalRunStates.has(experiment.state);
      const isCancelable = cancellableRunStates.includes(experiment.state);
      const isKillable = isTaskKillable(experiment);
      const isActivatable = experiment.state === RunState.Paused;
      const isPausable = experiment.state === RunState.Active;
      if (!tracker.hasArchivable && isArchivable) tracker.hasArchivable = true;
      if (!tracker.hasUnarchivable && experiment.archived) tracker.hasUnarchivable = true;
      if (!tracker.hasCancelable && isCancelable) tracker.hasCancelable = true;
      if (!tracker.hasKillable && isKillable) tracker.hasKillable = true;
      if (!tracker.hasActivatable && isActivatable) tracker.hasActivatable = true;
      if (!tracker.hasPausable && isPausable) tracker.hasPausable = true;
    }
    return tracker;
  }, [ experimentMap, settings.row ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = (settings.state || []).map(state => encodeExperimentState(state as RunState));
      const response = await getExperiments(
        {
          archived: settings.archived,
          labels: settings.label,
          limit: settings.tableLimit,
          name: settings.search,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentsRequestSortBy, settings.sortKey),
          states: validateDetApiEnumList(Determinedexperimentv1State, states),
          users: settings.user,
        },
        { signal: canceler.signal },
      );
      setTotal(response.pagination.total || 0);
      setExperiments(prev => {
        if (isEqual(prev, response.experiments)) return prev;
        return response.experiments;
      });
      setIsLoading(false);
    } catch (e) {
      handleError({ message: 'Unable to fetch experiments.', silent: true, type: ErrorType.Api });
      setIsLoading(false);
    }
  }, [ canceler, settings ]);

  const fetchLabels = useCallback(async () => {
    try {
      const labels = await getExperimentLabels({ signal: canceler.signal });
      labels.sort((a, b) => alphanumericSorter(a, b));
      setLabels(labels);
    } catch (e) {}
  }, [ canceler.signal ]);

  const fetchAll = useCallback(() => {
    fetchExperiments();
    fetchLabels();
    fetchUsers();
  }, [ fetchExperiments, fetchLabels, fetchUsers ]);

  usePolling(fetchAll);

  const experimentTags = useExperimentTags(fetchAll);

  const handleActionComplete = useCallback(() => fetchExperiments(), [ fetchExperiments ]);

  const handleArchiveFilterApply = useCallback((archived: string[]) => {
    const archivedFilter = archived.length === 1
      ? archived[0] === ArchiveFilter.Archived : undefined;
    updateSettings({ archived: archivedFilter, row: undefined });
  }, [ updateSettings ]);

  const handleArchiveFilterReset = useCallback(() => {
    updateSettings({ archived: undefined, row: undefined });
  }, [ updateSettings ]);

  const archiveFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      values={isBoolean(settings.archived)
        ? [ settings.archived ? ArchiveFilter.Archived : ArchiveFilter.Unarchived ]
        : undefined}
      onFilter={handleArchiveFilterApply}
      onReset={handleArchiveFilterReset}
    />
  ), [ handleArchiveFilterApply, handleArchiveFilterReset, settings.archived ]);

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
    const labelsRenderer = (value: string, record: ExperimentItem) => (
      <TagList
        compact
        tags={record.labels}
        onChange={experimentTags.handleTagListChange(record.id)}
      />
    );

    const actionRenderer: ExperimentRenderer = (_, record) => (
      <TaskActionDropdown
        curUser={user}
        task={experimentToTask(record)}
        onComplete={handleActionComplete} />
    );

    const tableColumns: ColumnsType<ExperimentItem> = [
      {
        dataIndex: 'id',
        key: V1GetExperimentsRequestSortBy.ID,
        render: experimentNameRenderer,
        sorter: true,
        title: 'ID',
      },
      {
        dataIndex: 'name',
        filterDropdown: nameFilterSearch,
        filterIcon: tableSearchIcon,
        key: V1GetExperimentsRequestSortBy.NAME,
        onHeaderCell: () => settings.search ? { className: tableCss.headerFilterOn } : {},
        render: experimentNameRenderer,
        sorter: true,
        title: 'Name',
        width: 240,
      },
      {
        dataIndex: 'labels',
        filterDropdown: labelFilterDropdown,
        filters: labels.map(label => ({ text: label, value: label })),
        key: 'labels',
        onHeaderCell: () => settings.label ? { className: tableCss.headerFilterOn } : {},
        render: labelsRenderer,
        title: 'Labels',
        width: 120,
      },
      {
        key: V1GetExperimentsRequestSortBy.STARTTIME,
        render: (_: number, record: ExperimentItem): React.ReactNode =>
          relativeTimeRenderer(new Date(record.startTime)),
        sorter: true,
        title: 'Start Time',
      },
      {
        key: 'duration',
        render: expermentDurationRenderer,
        title: 'Duration',
      },
      {
        dataIndex: 'numTrials',
        key: V1GetExperimentsRequestSortBy.NUMTRIALS,
        sorter: true,
        title: 'Trials',
      },
      {
        filterDropdown: stateFilterDropdown,
        filters: Object.values(RunState)
          .filter(value => value !== RunState.Unspecified)
          .map((value) => ({
            text: <Badge state={value} type={BadgeType.State} />,
            value,
          })),
        key: V1GetExperimentsRequestSortBy.STATE,
        onHeaderCell: () => settings.state ? { className: tableCss.headerFilterOn } : {},
        render: stateRenderer,
        sorter: true,
        title: 'State',
      },
      {
        dataIndex: 'resourcePool',
        key: 'resourcePool',
        sorter: true,
        title: 'Resource Pool',
      },
      {
        key: V1GetExperimentsRequestSortBy.PROGRESS,
        render: experimentProgressRenderer,
        sorter: true,
        title: 'Progress',
      },
      {
        dataIndex: 'archived',
        filterDropdown: archiveFilterDropdown,
        filters: [
          { text: capitalize(ArchiveFilter.Archived), value: ArchiveFilter.Archived },
          { text: capitalize(ArchiveFilter.Unarchived), value: ArchiveFilter.Unarchived },
        ],
        key: 'archived',
        onHeaderCell: () => settings.archived != null ? { className: tableCss.headerFilterOn } : {},
        render: archivedRenderer,
        title: 'Archived',
      },
      {
        filterDropdown: userFilterDropdown,
        filters: users.map(user => ({ text: user.username, value: user.username })),
        key: V1GetExperimentsRequestSortBy.USER,
        onHeaderCell: () => settings.user ? { className: tableCss.headerFilterOn } : {},
        render: userRenderer,
        sorter: true,
        title: 'User',
      },
      {
        align: 'right',
        className: 'fullCell',
        fixed: 'right',
        key: 'action',
        render: actionRenderer,
        title: '',
        width: 40,
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
    user,
    archiveFilterDropdown,
    handleActionComplete,
    experimentTags,
    labelFilterDropdown,
    labels,
    nameFilterSearch,
    settings,
    stateFilterDropdown,
    tableSearchIcon,
    userFilterDropdown,
    users,
  ]);

  const sendBatchActions = useCallback((action: Action): Promise<void[] | CommandTask> => {
    if (action === Action.OpenTensorBoard) {
      return openOrCreateTensorboard({ experimentIds: settings.row });
    }
    return Promise.all((settings.row || []).map(experimentId => {
      switch (action) {
        case Action.Activate:
          return activateExperiment({ experimentId });
        case Action.Archive:
          return archiveExperiment({ experimentId });
        case Action.Cancel:
          return cancelExperiment({ experimentId });
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
  }, [ settings.row ]);

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
      handleError({
        error: e,
        level: ErrorLevel.Error,
        message: e.message,
        publicMessage: 'Please try again later.',
        publicSubject,
        silent: false,
        type: ErrorType.Server,
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
    if (action === Action.OpenTensorBoard) {
      submitBatchAction(action);
    } else {
      showConfirmation(action as Action);
    }
  }, [ submitBatchAction, showConfirmation ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<ExperimentItem>;
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
  }, [ columns, settings.tableOffset, updateSettings ]);

  const handleTableRowSelect = useCallback(rowKeys => {
    updateSettings({ row: rowKeys });
  }, [ updateSettings ]);

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [ updateSettings ]);

  const resetFilters = useCallback(() => {
    resetSettings([ ...filterKeys, 'tableOffset' ]);
  }, [ resetSettings ]);

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

  return (
    <Page
      id="experiments"
      options={<FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />}
      title="Experiments">
      <TableBatch
        actions={[
          { label: Action.OpenTensorBoard, value: Action.OpenTensorBoard },
          { disabled: !hasActivatable, label: Action.Activate, value: Action.Activate },
          { disabled: !hasPausable, label: Action.Pause, value: Action.Pause },
          { disabled: !hasArchivable, label: Action.Archive, value: Action.Archive },
          { disabled: !hasUnarchivable, label: Action.Unarchive, value: Action.Unarchive },
          { disabled: !hasCancelable, label: Action.Cancel, value: Action.Cancel },
          { disabled: !hasKillable, label: Action.Kill, value: Action.Kill },
        ]}
        selectedRowCount={(settings.row || []).length}
        onAction={handleBatchAction}
        onClear={clearSelected}
      />
      <ResponsiveTable<ExperimentItem>
        columns={columns}
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
          selectedRowKeys: settings.row,
        }}
        showSorterTooltip={false}
        size="small"
        onChange={handleTableChange}
      />
    </Page>
  );
};

export default ExperimentList;
