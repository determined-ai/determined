import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Modal } from 'antd';
import { ColumnsType, FilterDropdownProps, SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Icon from 'components/Icon';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import tableCss from 'components/ResponsiveTable.module.scss';
import {
  archivedRenderer, defaultRowClassName, experimentNameRenderer, experimentProgressRenderer,
  ExperimentRenderer, expermentDurationRenderer, getFullPaginationConfig, MINIMUM_PAGE_SIZE,
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
import useStorage from 'hooks/useStorage';
import { parseUrl } from 'routes/utils';
import {
  activateExperiment, archiveExperiment, cancelExperiment, getExperimentLabels, getExperiments,
  killExperiment, openOrCreateTensorboard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { Determinedexperimentv1State, V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { ApiSorter } from 'services/types';
import { validateDetApiEnum, validateDetApiEnumList } from 'services/utils';
import {
  ArchiveFilter, CommandTask, ExperimentFilters, ExperimentItem, Pagination, RunState,
} from 'types';
import { isEqual } from 'utils/data';
import { alphanumericSorter } from 'utils/sort';
import { capitalize } from 'utils/string';
import {
  cancellableRunStates, experimentToTask, isTaskKillable, terminalRunStates,
} from 'utils/types';
import { openCommand } from 'wait';

import css from './ExperimentList.module.scss';

enum Action {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  Kill = 'Kill',
  Pause = 'Pause',
  OpenTensorBoard = 'OpenTensorboard',
  Unarchive = 'Unarchive',
}

const URL_ALL = 'all';

const STORAGE_PATH = 'experiment-list';
const STORAGE_FILTERS_KEY = 'filters';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const defaultFilters: ExperimentFilters = {
  archived: undefined,
  states: undefined,
  users: undefined,
};

const defaultSorter: ApiSorter<V1GetExperimentsRequestSortBy> = {
  descend: true,
  key: V1GetExperimentsRequestSortBy.STARTTIME,
};

const ExperimentList: React.FC = () => {
  const { auth, users } = useStore();
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initFilters = storage.getWithDefault(
    STORAGE_FILTERS_KEY,
    (!auth.user || auth.user?.isAdmin) ? defaultFilters : {
      ...defaultFilters,
      users: [ auth.user?.username ],
    },
  );
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ canceler ] = useState(new AbortController());
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>();
  const [ labels, setLabels ] = useState<string[]>([]);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ isUrlParsed, setIsUrlParsed ] = useState(false);
  const [ filters, setFilters ] = useState<ExperimentFilters>(initFilters);
  const [ pagination, setPagination ] = useState<Pagination>({ limit: initLimit, offset: 0 });
  const [ search, setSearch ] = useState('');
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);
  const [ sorter, setSorter ] = useState(initSorter);
  const [ total, setTotal ] = useState(0);

  /*
   * When filters changes update the page URL.
   */
  useEffect(() => {
    if (!isUrlParsed) return;

    const searchParams = new URLSearchParams;
    const url = parseUrl(window.location.href);

    // archived
    searchParams.append('archived', filters.archived ? filters.archived : URL_ALL);

    // labels
    if (filters.labels && filters.labels.length > 0) {
      filters.labels.forEach(label => searchParams.append('label', label));
    } else {
      searchParams.append('label', URL_ALL);
    }

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

    const urlSearchParams = parseUrl(window.location.href).searchParams;

    // archived
    const archived = urlSearchParams.get('archived');
    if (archived != null) {
      filters.archived = archived === URL_ALL ? undefined : archived as ArchiveFilter;
    }

    // labels
    const label = urlSearchParams.getAll('label');
    if (label!= null) {
      filters.labels = (label.includes(URL_ALL) ? undefined : label);
    }

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
    if (
      sortKey != null
      && Object.values(V1GetExperimentsRequestSortBy).includes(sortKey)
    ) {
      sorter.key = sortKey as unknown as V1GetExperimentsRequestSortBy;
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

  const experimentMap = useMemo(() => {
    return (experiments || []).reduce((acc, experiment) => {
      acc[experiment.id] = experiment;
      return acc;
    }, {} as Record<string, ExperimentItem>);
  }, [ experiments ]);

  const selectedExperiments = useMemo(() => {
    return selectedRowKeys.map(key => experimentMap[key]);
  }, [ experimentMap, selectedRowKeys ]);

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
    for (let i = 0; i < selectedExperiments.length; i++) {
      const experiment = selectedExperiments[i];
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
  }, [ selectedExperiments ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = (filters.states || []).map(state => encodeExperimentState(state as RunState));
      const response = await getExperiments(
        {
          archived: filters.archived ? filters.archived !== 'unarchived' : undefined,
          description: search,
          labels: filters.labels,
          limit: pagination.limit,
          offset: pagination.offset,
          orderBy: sorter.descend ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentsRequestSortBy, sorter.key),
          states: validateDetApiEnumList(Determinedexperimentv1State, states),
          users: filters.users,
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
  }, [ canceler, filters, pagination, search, sorter ]);

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

  const handleFilterChange = useCallback((filters: ExperimentFilters): void => {
    storage.set(STORAGE_FILTERS_KEY, filters);
    setSelectedRowKeys([]);
    setFilters(filters);
    setPagination(prev => ({ ...prev, offset: 0 }));
  }, [ setFilters, storage ]);

  const handleArchiveFilterApply = useCallback((archived: string[]) => {
    const archivedFilter = archived.length === 1 ? archived[0] as ArchiveFilter : undefined;
    handleFilterChange({ ...filters, archived: archivedFilter });
  }, [ handleFilterChange, filters ]);

  const handleArchiveFilterReset = useCallback(() => {
    handleFilterChange({ ...filters, archived: undefined });
  }, [ handleFilterChange, filters ]);

  const archiveFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      values={filters.archived ? [ filters.archived ] : undefined}
      onFilter={handleArchiveFilterApply}
      onReset={handleArchiveFilterReset}
    />
  ), [ filters.archived, handleArchiveFilterApply, handleArchiveFilterReset ]);

  const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" />, []);

  const handleNameSearchApply = useCallback((newSearch: string) => {
    setSearch(newSearch);
    setPagination(prev => ({ ...prev, offset: 0 }));
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

  const handleLabelFilterApply = useCallback((labels: string[]) => {
    handleFilterChange({ ...filters, labels: labels.length !== 0 ? labels : undefined });
  }, [ handleFilterChange, filters ]);

  const handleLabelFilterReset = useCallback(() => {
    handleFilterChange({ ...filters, labels: undefined });
  }, [ handleFilterChange, filters ]);

  const labelFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      searchable
      values={filters.labels}
      onFilter={handleLabelFilterApply}
      onReset={handleLabelFilterReset}
    />
  ), [ filters.labels, handleLabelFilterApply, handleLabelFilterReset ]);

  const handleStateFilterApply = useCallback((states: string[]) => {
    handleFilterChange({ ...filters, states: states.length !== 0 ? states : undefined });
  }, [ handleFilterChange, filters ]);

  const handleStateFilterReset = useCallback(() => {
    handleFilterChange({ ...filters, states: undefined });
  }, [ handleFilterChange, filters ]);

  const stateFilterDropdown = useCallback((filterProps: FilterDropdownProps) => (
    <TableFilterDropdown
      {...filterProps}
      multiple
      values={filters.states}
      onFilter={handleStateFilterApply}
      onReset={handleStateFilterReset} />
  ), [ filters.states, handleStateFilterApply, handleStateFilterReset ]);

  const handleUserFilterApply = useCallback((users: string[]) => {
    handleFilterChange({ ...filters, users: users.length !== 0 ? users : undefined });
  }, [ handleFilterChange, filters ]);

  const handleUserFilterReset = useCallback(() => {
    handleFilterChange({ ...filters, users: undefined });
  }, [ handleFilterChange, filters ]);

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
    const labelsRenderer = (value: string, record: ExperimentItem) => (
      <TagList
        compact
        tags={record.labels}
        onChange={experimentTags.handleTagListChange(record.id)}
      />
    );

    const actionRenderer: ExperimentRenderer = (_, record) => (
      <TaskActionDropdown task={experimentToTask(record)} onComplete={handleActionComplete} />
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
        key: V1GetExperimentsRequestSortBy.DESCRIPTION,
        onHeaderCell: () => search ? { className: tableCss.headerFilterOn } : {},
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
        onHeaderCell: () => filters.labels ? { className: tableCss.headerFilterOn } : {},
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
        onHeaderCell: () => filters.states ? { className: tableCss.headerFilterOn } : {},
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
        onHeaderCell: () => filters.archived ? { className: tableCss.headerFilterOn } : {},
        render: archivedRenderer,
        title: 'Archived',
      },
      {
        filterDropdown: userFilterDropdown,
        filters: users.map(user => ({ text: user.username, value: user.username })),
        key: V1GetExperimentsRequestSortBy.USER,
        onHeaderCell: () => filters.users ? { className: tableCss.headerFilterOn } : {},
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
      if (column.key === sorter.key) column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      return column;
    });
  }, [
    archiveFilterDropdown,
    handleActionComplete,
    experimentTags,
    filters,
    labelFilterDropdown,
    labels,
    nameFilterSearch,
    search,
    sorter,
    stateFilterDropdown,
    tableSearchIcon,
    userFilterDropdown,
    users,
  ]);

  const sendBatchActions = useCallback((action: Action): Promise<void[] | CommandTask> => {
    if (action === Action.OpenTensorBoard) {
      return openOrCreateTensorboard(
        { experimentIds: selectedExperiments.map(experiment => experiment.id) },
      );
    }
    return Promise.all(selectedExperiments
      .map(experiment => {
        switch (action) {
          case Action.Activate:
            return activateExperiment({ experimentId: experiment.id });
          case Action.Archive:
            return archiveExperiment({ experimentId: experiment.id });
          case Action.Cancel:
            return cancelExperiment({ experimentId: experiment.id });
          case Action.Kill:
            return killExperiment({ experimentId: experiment.id });
          case Action.Pause:
            return pauseExperiment({ experimentId: experiment.id });
          case Action.Unarchive:
            return unarchiveExperiment({ experimentId: experiment.id });
          default:
            return Promise.resolve();
        }
      }));
  }, [ selectedExperiments ]);

  const handleBatchAction = useCallback(async (action: Action) => {
    try {
      const result = await sendBatchActions(action);
      if (action === Action.OpenTensorBoard && result) {
        openCommand(result as CommandTask);
      }

      /*
       * Deselect selected rows since their states may have changed where they
       * are no longer part of the filter criteria.
       */
      setSelectedRowKeys([]);

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
  }, [ fetchExperiments, sendBatchActions ]);

  const handleConfirmation = useCallback((action: Action) => {
    Modal.confirm({
      content: `
        Are you sure you want to ${action.toLocaleLowerCase()}
        all the eligible selected experiments?
      `,
      icon: <ExclamationCircleOutlined />,
      okText: /cancel/i.test(action) ? 'Confirm' : action,
      onOk: () => handleBatchAction(action),
      title: 'Confirm Batch Action',
    });
  }, [ handleBatchAction ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<ExperimentItem>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    storage.set(STORAGE_SORTER_KEY, { descend: order === 'descend', key: columnKey as string });
    setSorter({ descend: order === 'descend', key: columnKey as V1GetExperimentsRequestSortBy });

    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPagination(prev => ({
      ...prev,
      limit: tablePagination.pageSize,
      offset: (tablePagination.current - 1) * tablePagination.pageSize,
    }));
    setSelectedRowKeys([]);
  }, [ columns, setSorter, storage ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  /*
   * Get new experiments based on changes to the
   * filters, pagination, search and sorter.
   */
  useEffect(() => {
    fetchExperiments();
    setIsLoading(true);
  }, [ fetchExperiments, filters, pagination, search, sorter ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <Page id="experiments" title="Experiments">
      <div className={css.base}>
        <TableBatch selectedRowCount={selectedRowKeys.length}>
          <Button onClick={(): Promise<void> => handleBatchAction(Action.OpenTensorBoard)}>
            View in TensorBoard
          </Button>
          <Button
            disabled={!hasActivatable}
            type="primary"
            onClick={(): void => handleConfirmation(Action.Activate)}>Activate</Button>
          <Button
            disabled={!hasPausable}
            onClick={(): void => handleConfirmation(Action.Pause)}>Pause</Button>
          <Button
            disabled={!hasArchivable}
            onClick={(): void => handleConfirmation(Action.Archive)}>Archive</Button>
          <Button
            disabled={!hasUnarchivable}
            onClick={(): void => handleConfirmation(Action.Unarchive)}>Unarchive</Button>
          <Button
            disabled={!hasCancelable}
            onClick={(): void => handleConfirmation(Action.Cancel)}>Cancel</Button>
          <Button
            danger
            disabled={!hasKillable}
            type="primary"
            onClick={(): void => handleConfirmation(Action.Kill)}>Kill</Button>
        </TableBatch>
        <ResponsiveTable<ExperimentItem>
          columns={columns}
          dataSource={experiments}
          loading={isLoading}
          pagination={getFullPaginationConfig(pagination, total)}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="id"
          rowSelection={{ onChange: handleTableRowSelect, selectedRowKeys }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange}
        />
      </div>
    </Page>
  );
};

export default ExperimentList;
