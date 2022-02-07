import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Modal, Space, Switch } from 'antd';
import { ColumnsType, FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import FilterCounter from 'components/FilterCounter';
import Icon from 'components/Icon';
import InlineEditor from 'components/InlineEditor';
import Label, { LabelTypes } from 'components/Label';
import Link from 'components/Link';
import Page from 'components/Page';
import ResponsiveTable, { handleTableChange } from 'components/ResponsiveTable';
import tableCss from 'components/ResponsiveTable.module.scss';
import {
  checkmarkRenderer, defaultRowClassName, experimentNameRenderer, experimentProgressRenderer,
  ExperimentRenderer, expermentDurationRenderer, getFullPaginationConfig,
  relativeTimeRenderer, stateRenderer, userRenderer,
} from 'components/Table';
import TableBatch from 'components/TableBatch';
import TableFilterDropdown from 'components/TableFilterDropdown';
import TableFilterSearch from 'components/TableFilterSearch';
import TagList from 'components/TagList';
import TaskActionDropdown from 'components/TaskActionDropdown';
import { cancellableRunStates, deletableRunStates, pausableRunStates,
  terminalRunStates } from 'constants/states';
import { useStore } from 'contexts/Store';
import useExperimentTags from 'hooks/useExperimentTags';
import { useFetchUsers } from 'hooks/useFetch';
import useModalCustomizeColumns from 'hooks/useModal/useModalCustomizeColumns';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { paths } from 'routes/utils';
import {
  activateExperiment, archiveExperiment, cancelExperiment, deleteExperiment, getExperimentLabels,
  getExperiments, killExperiment, openOrCreateTensorBoard,
  patchExperiment, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { Determinedexperimentv1State, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { validateDetApiEnum, validateDetApiEnumList } from 'services/utils';
import {
  ExperimentAction as Action, CommandTask, ExperimentItem, RecordKey, RunState,
} from 'types';
import { isEqual } from 'utils/data';
import handleError, { ErrorLevel } from 'utils/error';
import { alphaNumericSorter } from 'utils/sort';
import { sentenceToCamelCase } from 'utils/string';
import { isTaskKillable, taskFromExperiment } from 'utils/task';
import { openCommand } from 'wait';

import settingsConfig, { DEFAULT_COLUMNS, Settings } from './ExperimentList.settings';

const filterKeys: Array<keyof Settings> = [ 'label', 'search', 'state', 'user' ];

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
    hasDeletable,
    hasKillable,
    hasPausable,
    hasUnarchivable,
  } = useMemo(() => {
    const tracker = {
      hasActivatable: false,
      hasArchivable: false,
      hasCancelable: false,
      hasDeletable: false,
      hasKillable: false,
      hasPausable: false,
      hasUnarchivable: false,
    };
    for (const id of settings.row || []) {
      const experiment = experimentMap[id];
      if (!experiment) continue;
      const isArchivable = !experiment.archived && terminalRunStates.has(experiment.state);
      const isCancelable = cancellableRunStates.has(experiment.state);
      const isDeletable = deletableRunStates.has(experiment.state) &&
        user && (user.isAdmin || user.username === experiment.username);
      const isKillable = isTaskKillable(experiment);
      const isActivatable = experiment.state === RunState.Paused;
      const isPausable = pausableRunStates.has(experiment.state);
      if (!tracker.hasArchivable && isArchivable) tracker.hasArchivable = true;
      if (!tracker.hasUnarchivable && experiment.archived) tracker.hasUnarchivable = true;
      if (!tracker.hasCancelable && isCancelable) tracker.hasCancelable = true;
      if (!tracker.hasDeletable && isDeletable) tracker.hasDeletable = true;
      if (!tracker.hasKillable && isKillable) tracker.hasKillable = true;
      if (!tracker.hasActivatable && isActivatable) tracker.hasActivatable = true;
      if (!tracker.hasPausable && isPausable) tracker.hasPausable = true;
    }
    return tracker;
  }, [ experimentMap, settings.row, user ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = (settings.state || []).map(state => encodeExperimentState(state as RunState));
      const response = await getExperiments(
        {
          archived: settings.archived ? undefined : false,
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
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments.' });
    } finally {
      setIsLoading(false);
    }
  }, [ canceler,
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
      const labels = await getExperimentLabels({ signal: canceler.signal });
      labels.sort((a, b) => alphaNumericSorter(a, b));
      setLabels(labels);
    } catch (e) { handleError(e); }
  }, [ canceler.signal ]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([ fetchExperiments(), fetchLabels(), fetchUsers() ]);
  }, [ fetchExperiments, fetchLabels, fetchUsers ]);

  usePolling(fetchAll);

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
        silent: true,
      });
      setIsLoading(false);
    }
  }, [ ]);

  const columns = useMemo(() => {
    const tagsRenderer = (value: string, record: ExperimentItem) => (
      <TagList
        compact
        tags={record.labels}
        onChange={experimentTags.handleTagListChange(record.id)}
      />
    );

    const actionRenderer: ExperimentRenderer = (_, record) => (
      <TaskActionDropdown
        curUser={user}
        task={taskFromExperiment(record)}
        onComplete={handleActionComplete}
      />
    );

    const descriptionRenderer = (value:string, record: ExperimentItem) => (
      <InlineEditor
        disabled={record.archived}
        placeholder="Add description..."
        value={value}
        onSave={(newDescription: string) => saveExperimentDescription(newDescription, record.id)}
      />
    );

    const forkedFromRenderer = (
      value: string | number | undefined,
    ): React.ReactNode => (
      value ? <Link path={paths.experimentDetails(value)}>{value}</Link> : null
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
      { dataIndex: 'description', render: descriptionRenderer, title: 'Description' },
      {
        dataIndex: 'labels',
        filterDropdown: labelFilterDropdown,
        filters: labels.map(label => ({ text: label, value: label })),
        key: 'labels',
        onHeaderCell: () => settings.label ? { className: tableCss.headerFilterOn } : {},
        render: tagsRenderer,
        title: 'Tags',
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
          .filter(value => [
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
        key: V1GetExperimentsRequestSortBy.STATE,
        onHeaderCell: () => settings.state ? { className: tableCss.headerFilterOn } : {},
        render: stateRenderer,
        sorter: true,
        title: 'State',
      },
      {
        dataIndex: 'searcherType',
        key: 'searcherType',
        title: 'Searcher Type',
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
        key: 'archived',
        render: checkmarkRenderer,
        title: 'Archived',
      },
      {
        dataIndex: 'forkedFrom',
        key: 'forkedFrom',
        render: forkedFromRenderer,
        title: 'Forked From',
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
    handleActionComplete,
    experimentTags,
    labelFilterDropdown,
    labels,
    nameFilterSearch,
    saveExperimentDescription,
    settings,
    stateFilterDropdown,
    tableSearchIcon,
    userFilterDropdown,
    users,
  ]);

  const visibleColumns = useMemo(() => {
    return columns.filter(column => {
      if (column.key === 'action') return true;
      if (column.key === 'archived') return settings.archived;
      return settings.columns?.includes(sentenceToCamelCase(column.title as string));
    });
  }, [ columns, settings.archived, settings.columns ]);

  const transferColumns = useMemo(() => {
    return columns.filter(column => column.title !== '' && column.title !== 'Archived')
      .map(column => sentenceToCamelCase(column.title as string));
  }, [ columns ]);

  const sendBatchActions = useCallback((action: Action): Promise<void[] | CommandTask> => {
    if (action === Action.OpenTensorBoard) {
      return openOrCreateTensorBoard({ experimentIds: settings.row });
    }
    return Promise.all((settings.row || []).map(experimentId => {
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
    if (action === Action.OpenTensorBoard) {
      submitBatchAction(action);
    } else {
      showConfirmation(action as Action);
    }
  }, [ submitBatchAction, showConfirmation ]);

  const handleTableRowSelect = useCallback(rowKeys => {
    updateSettings({ row: rowKeys });
  }, [ updateSettings ]);

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [ updateSettings ]);

  const resetFilters = useCallback(() => {
    resetSettings([ ...filterKeys, 'tableOffset' ]);
  }, [ resetSettings ]);

  const handleUpdateColumns = useCallback((columns: string[]) => {
    updateSettings({ columns: columns.length === 0 ? [ 'name' ] : columns });
  }, [ updateSettings ]);

  const { modalOpen } = useModalCustomizeColumns({
    columns: transferColumns,
    defaultVisibleColumns: DEFAULT_COLUMNS,
    onSave: handleUpdateColumns,
  });

  const openModal = useCallback(() => {
    modalOpen({ initialVisibleColumns: settings.columns });
  }, [ settings.columns, modalOpen ]);

  const switchShowArchived = useCallback((showArchived: boolean) => {
    updateSettings({ archived: showArchived, row: undefined });
  }, [ updateSettings ]);

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
      options={(
        <Space>
          <Switch checked={settings.archived} onChange={switchShowArchived} />
          <Label type={LabelTypes.TextOnly}>Show Archived</Label>
          <Button onClick={openModal}>Columns</Button>
          <FilterCounter activeFilterCount={filterCount} onReset={resetFilters} />
        </Space>
      )}
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
          { disabled: !hasDeletable, label: Action.Delete, value: Action.Delete },
        ]}
        selectedRowCount={(settings.row ?? []).length}
        onAction={handleBatchAction}
        onClear={clearSelected}
      />
      <ResponsiveTable<ExperimentItem>
        columns={visibleColumns}
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
        showSorterTooltip={false}
        size="small"
        onChange={handleTableChange(columns, settings, updateSettings)}
      />
    </Page>
  );
};

export default ExperimentList;
