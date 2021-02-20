import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Input, Modal } from 'antd';
import { SelectValue } from 'antd/es/select';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import LabelSelectFilter from 'components/LabelSelectFilter';
import Page from 'components/Page';
import ResponsiveFilters from 'components/ResponsiveFilters';
import ResponsiveTable from 'components/ResponsiveTable';
import { Indicator } from 'components/Spinner';
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
import Auth from 'contexts/Auth';
import { useFetchUsers } from 'contexts/Users';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import useExperimentTags from 'hooks/useExperimentTags';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import {
  activateExperiment, archiveExperiment, cancelExperiment, getExperiments,
  killExperiment, openOrCreateTensorboard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import { V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { ApiSorter } from 'services/types';
import { validateDetApiEnum } from 'services/utils';
import {
  ALL_VALUE, CommandTask, ExperimentFilters, ExperimentItem, Pagination, RunState,
} from 'types';
import {
  cancellableRunStates, experimentToTask, isTaskKillable, terminalRunStates,
} from 'utils/types';
import { openCommand } from 'wait';

import css from './ExperimentList.module.scss';
import { columns as defaultColumns, idRenderer } from './ExperimentList.table';

enum Action {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  Kill = 'Kill',
  Pause = 'Pause',
  OpenTensorBoard = 'OpenTensorboard',
  Unarchive = 'Unarchive',
}

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
  const auth = Auth.useStateContext();
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
  const [ pagination, setPagination ] = useState<Pagination>({ limit: initLimit, offset: 0 });
  const [ total, setTotal ] = useState(0);
  const [ filters, setFilters ] = useState<ExperimentFilters>(initFilters);
  const [ sorter, setSorter ] = useState(initSorter);
  const [ search, setSearch ] = useState('');
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>();
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);
  const [ canceler ] = useState(new AbortController());

  const fetchUsers = useFetchUsers(canceler);

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
      setExperiments(response.experiments);
    } catch (e) {
      handleError({ message: 'Unable to fetch experiments.', silent: true, type: ErrorType.Api });
    }
  }, [ canceler, filters, pagination, search, sorter ]);

  const fetchAll = useCallback(() => {
    fetchExperiments();
    fetchUsers();
  }, [ fetchExperiments, fetchUsers ]);

  usePolling(fetchAll);

  const experimentTags = useExperimentTags(fetchExperiments);

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
            onChange={handleSearchChange} />
          <ResponsiveFilters hasFiltersApplied={hasFiltersApplied}>
            <Toggle
              checked={filters.showArchived}
              prefixLabel="Show Archived"
              onChange={handleArchiveChange} />
            <LabelSelectFilter
              value={filters.labels}
              onChange={handleLabelsChange} />
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
        </TableBatch>
        <ResponsiveTable<ExperimentItem>
          columns={columns}
          dataSource={experiments}
          loading={{
            indicator: <Indicator />,
            spinning: !experiments,
          }}
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
