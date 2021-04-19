import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Input, Modal, Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import MultiSelect from 'components/MultiSelect';
import Page from 'components/Page';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import StateSelectFilter from 'components/StateSelectFilter';
import {
  defaultRowClassName, ExperimentRenderer,
  getFullPaginationConfig, MINIMUM_PAGE_SIZE,
} from 'components/Table';
import TableBatch from 'components/TableBatch';
import TagList from 'components/TagList';
import TaskActionDropdown from 'components/TaskActionDropdown';
import Toggle from 'components/Toggle';
import UserSelectFilter from 'components/UserSelectFilter';
import { useStore } from 'contexts/Store';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import useExperimentTags from 'hooks/useExperimentTags';
import { useFetchUsers } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { parseUrl } from 'routes/utils';
import {
  activateExperiment, archiveExperiment, cancelExperiment, deleteExperiment, getExperimentLabels,
  getExperiments, killExperiment, openOrCreateTensorboard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { ApiSorter } from 'services/types';
import { validateDetApiEnum } from 'services/utils';
import {
  ALL_VALUE, CommandTask, ExperimentFilters, ExperimentItem, Pagination, RunState,
} from 'types';
import { isEqual } from 'utils/data';
import { alphanumericSorter } from 'utils/sort';
import {
  cancellableRunStates, experimentToTask, isTaskKillable, terminalRunStates,
} from 'utils/types';
import { openCommand } from 'wait';

import css from './ExperimentList.module.scss';
import { columns as defaultColumns, idRenderer } from './ExperimentList.table';

const { Option } = Select;

enum Action {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  Delete = 'Delete',
  Kill = 'Kill',
  Pause = 'Pause',
  OpenTensorBoard = 'OpenTensorboard',
  Unarchive = 'Unarchive',
}

const URL_ALL = '';

const STORAGE_PATH = 'experiment-list';
const STORAGE_FILTERS_KEY = 'filters';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const defaultFilters: ExperimentFilters = {
  showArchived: false,
  states: [ ALL_VALUE ],
  username: undefined,
};

const defaultSorter: ApiSorter<V1GetExperimentsRequestSortBy> = {
  descend: true,
  key: V1GetExperimentsRequestSortBy.STARTTIME,
};

const ExperimentList: React.FC = () => {
  const { auth } = useStore();
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initFilters = storage.getWithDefault(
    STORAGE_FILTERS_KEY,
    (!auth.user || auth.user?.isAdmin) ? defaultFilters : {
      ...defaultFilters,
      username: auth.user?.username,
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
    searchParams.append('archived', filters.showArchived ? '1' : '0');

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

    // username
    searchParams.append('username', filters.username || URL_ALL);

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
      filters.showArchived = (archived === '1');
    }

    // labels
    if (urlSearchParams.get('label') != null) {
      const labels = urlSearchParams.getAll('label');
      filters.labels = (labels.includes(URL_ALL) ? [] : labels);
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
    if (urlSearchParams.get('state') != null) {
      const states = urlSearchParams.getAll('state');
      filters.states = (states.includes(URL_ALL) ? [ ALL_VALUE ] : states);
    }

    // username
    const username = urlSearchParams.get('username');
    if (username != null) {
      filters.username = (username === URL_ALL ? '' : username);
    }

    setFilters(filters);
    setIsUrlParsed(true);
    setPagination(pagination);
    setSorter(sorter);
  }, [ filters, isUrlParsed, pagination, search, sorter ]);

  const hasFiltersApplied = useMemo(() => {
    return filters.showArchived || !filters.states.includes(ALL_VALUE) || !!filters.username;
  }, [ filters ]);

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
    hasDeletable,
    hasKillable,
    hasPausable,
    hasUnarchivable,
  } = useMemo(() => {
    const tracker = {
      hasActivatable: false,
      hasArchivable: false,
      hasCancelable: false,
      hasDeletable: selectedExperiments.length > 0,
      hasKillable: false,
      hasPausable: false,
      hasUnarchivable: false,
    };
    for (let i = 0; i < selectedExperiments.length; i++) {
      const experiment = selectedExperiments[i];
      const isArchivable = !experiment.archived && terminalRunStates.has(experiment.state);
      const isCancelable = cancellableRunStates.includes(experiment.state);
      const isDeletable = terminalRunStates.has(experiment.state);
      const isKillable = isTaskKillable(experiment);
      const isActivatable = experiment.state === RunState.Paused;
      const isPausable = experiment.state === RunState.Active;
      if (!tracker.hasArchivable && isArchivable) tracker.hasArchivable = true;
      if (!tracker.hasUnarchivable && experiment.archived) tracker.hasUnarchivable = true;
      if (!tracker.hasCancelable && isCancelable) tracker.hasCancelable = true;
      if (!tracker.hasDeletable || !isDeletable) tracker.hasDeletable = false;
      if (!tracker.hasKillable && isKillable) tracker.hasKillable = true;
      if (!tracker.hasActivatable && isActivatable) tracker.hasActivatable = true;
      if (!tracker.hasPausable && isPausable) tracker.hasPausable = true;
    }
    return tracker;
  }, [ selectedExperiments ]);

  const fetchUsers = useFetchUsers(canceler);

  const fetchExperiments = useCallback(async (): Promise<void> => {
    try {
      const states = filters.states.includes(ALL_VALUE) ? undefined : filters.states.map(state => {
        /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
        return encodeExperimentState(state as RunState) as any;
      });
      const response = await getExperiments(
        {
          archived: filters.showArchived ? undefined : false,
          description: search,
          labels: filters.labels?.length === 0 ? undefined : filters.labels,
          limit: pagination.limit,
          offset: pagination.offset,
          orderBy: sorter.descend ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentsRequestSortBy, sorter.key),
          states,
          users: filters.username ? [ filters.username ] : undefined,
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

  const columns = useMemo(() => {
    const nameRenderer = (value: string, record: ExperimentItem) => (
      <div className={css.nameColumn}>
        {idRenderer(value, record)}
        <TagList
          tags={record.labels || []}
          onChange={experimentTags.handleTagListChange(record.id)}
        />
      </div>
    );

    const actionRenderer: ExperimentRenderer = (_, record) => (
      <TaskActionDropdown task={experimentToTask(record)} onComplete={handleActionComplete} />
    );

    const newColumns = [ ...defaultColumns ].map(column => {
      column.sortOrder = null;
      if (column.key === sorter.key) column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      if (column.key === V1GetExperimentsRequestSortBy.DESCRIPTION) column.render = nameRenderer;
      if (column.key === 'action') column.render = actionRenderer;
      return column;
    });

    return newColumns;
  }, [
    handleActionComplete,
    experimentTags,
    sorter,
  ]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
    setPagination(prev => ({ ...prev, offset: 0 }));
  }, []);

  const handleFilterChange = useCallback((filters: ExperimentFilters): void => {
    storage.set(STORAGE_FILTERS_KEY, filters);
    setSelectedRowKeys([]);
    setFilters(filters);
    setPagination(prev => ({ ...prev, offset: 0 }));
  }, [ setFilters, storage ]);

  const handleArchiveChange = useCallback((value: boolean): void => {
    handleFilterChange({ ...filters, showArchived: value });
  }, [ filters, handleFilterChange ]);

  const handleStateChange = useCallback((value: SelectValue): void => {
    if (typeof value !== 'string') return;
    handleFilterChange({ ...filters, states: [ value ] });
  }, [ filters, handleFilterChange ]);

  const handleUserChange = useCallback((value: SelectValue) => {
    const username = value === ALL_VALUE ? undefined : value as string;
    handleFilterChange({ ...filters, username });
  }, [ filters, handleFilterChange ]);

  const handleLabelsChange = useCallback((newValue: SelectValue) => {
    handleFilterChange({
      ...filters,
      labels: (newValue as Array<string>).map(label => label.toString()),
    });
  }, [ filters, handleFilterChange ]);

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
          case Action.Delete:
            return deleteExperiment({ experimentId: experiment.id });
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
  }, [ fetchExperiments ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  return (
    <Page id="experiments" title="Experiments">
      <div className={css.base}>
        <div className={css.header}>
          <Input
            allowClear
            className={css.search}
            placeholder="name"
            prefix={<Icon name="search" size="small" />}
            value={search}
            onChange={handleSearchChange}
          />
          <ResponsiveFilters hasFiltersApplied={hasFiltersApplied}>
            <Toggle
              checked={filters.showArchived}
              prefixLabel="Show Archived"
              onChange={handleArchiveChange} />
            <MultiSelect label="Labels" value={filters.labels} onChange={handleLabelsChange}>
              {labels.map((label) => <Option key={label} value={label}>{label}</Option>)}
            </MultiSelect>
            <StateSelectFilter
              showCommandStates={false}
              value={filters.states}
              onChange={handleStateChange} />
            <UserSelectFilter value={filters.username} onChange={handleUserChange} />
          </ResponsiveFilters>
        </div>
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
          <Button
            danger
            disabled={!hasDeletable}
            onClick={(): void => handleConfirmation(Action.Delete)}>Delete</Button>
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
