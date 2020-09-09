import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Button, Input, Modal, Table } from 'antd';
import { SelectValue } from 'antd/es/select';
import { SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import Page from 'components/Page';
import { Indicator } from 'components/Spinner';
import StateSelectFilter from 'components/StateSelectFilter';
import {
  defaultRowClassName, ExperimentRenderer,
  getFullPaginationConfig, isAlternativeAction, MINIMUM_PAGE_SIZE,
} from 'components/Table';
import TableBatch from 'components/TableBatch';
import TagList from 'components/TagList';
import TaskActionDropdown from 'components/TaskActionDropdown';
import Toggle from 'components/Toggle';
import UserSelectFilter from 'components/UserSelectFilter';
import Auth from 'contexts/Auth';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import {
  archiveExperiment, createTensorboard, getExperimentList, killExperiment, setExperimentState,
} from 'services/api';
import { patchExperiment } from 'services/api';
import { V1GetExperimentsRequestSortBy } from 'services/api-ts-sdk';
import { ApiSorter } from 'services/types';
import {
  ALL_VALUE, Command, ExperimentFilters, ExperimentItem, Pagination, RunState, TBSourceType,
} from 'types';
import { handlePath, openBlank } from 'utils/routes';
import {
  cancellableRunStates, experimentToTask, isTaskKillable, terminalRunStates, waitPageUrl,
} from 'utils/types';

import css from './ExperimentList.module.scss';
import { columns as defaultColumns } from './ExperimentList.table';

enum Action {
  Activate = 'Activate',
  Archive = 'Archive',
  Cancel = 'Cancel',
  Kill = 'Kill',
  Pause = 'Pause',
  OpenTensorBoard = 'OpenTensorboard',
  Unarchive = 'Unarchive',
}

const defaultFilters: ExperimentFilters = {
  showArchived: false,
  states: [ ALL_VALUE ],
  username: undefined,
};

const defaultSorter: ApiSorter<V1GetExperimentsRequestSortBy> = {
  descend: true,
  key: V1GetExperimentsRequestSortBy.STARTTIME,
};

const STORAGE_PATH = 'experiment-list';
const STORAGE_FILTERS_KEY = 'filters';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const ExperimentList: React.FC = () => {
  const auth = Auth.useStateContext();
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initFilters = storage.getWithDefault(
    STORAGE_FILTERS_KEY,
    { ...defaultFilters, username: (auth.user || {}).username },
  );
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ pagination, setPagination ] = useState<Pagination>({ limit: initLimit, offset: 0 });
  const [ total, setTotal ] = useState(0);
  const [ filters, setFilters ] = useState<ExperimentFilters>(initFilters);
  const [ sorter, setSorter ] = useState(initSorter);
  const [ search, setSearch ] = useState('');
  const [ experiments, setExperiments ] = useState<ExperimentItem[]>();
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<string[]>([]);

  const showBatch = selectedRowKeys.length !== 0;

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
      const {
        experiments,
        pagination: responsePagination,
      } = await getExperimentList(sorter, pagination, filters, search);
      setTotal(responsePagination?.total || 0);
      setExperiments(experiments);
    } catch (e) {
      handleError({ message: 'Unable to fetch experiments.', silent: true, type: ErrorType.Api });
    }
  }, [ filters, pagination, search, sorter ]);

  usePolling(fetchExperiments);

  const updateTags = useCallback(async (id: number, labels: Record<string, boolean | null>) => {
    await patchExperiment({ body: { labels }, experimentId: id });
    await fetchExperiments();
  }, [ fetchExperiments ]);

  const handleTagListChange = useCallback((id: number) => (oldTag: string, newTag: string) => {
    updateTags(id, { [newTag]: true, [oldTag]: null });
  }, [ updateTags ]);

  const handleTagListCreate = useCallback((id: number) => (tag: string) => {
    updateTags(id, { [tag]: true });
  }, [ updateTags ]);

  const handleTagListDelete = useCallback((id: number) => (tag: string) => {
    updateTags(id, { [tag]: null });
  }, [ updateTags ]);

  const handleActionComplete = useCallback(() => fetchExperiments(), [ fetchExperiments ]);

  const columns = useMemo(() => {
    const nameRenderer = (_: string, record: ExperimentItem) => (
      <div className={css.nameColumn}>
        {record.name || ''}
        <TagList
          tags={record.labels || []}
          onChange={handleTagListChange(record.id)}
          onCreate={handleTagListCreate(record.id)}
          onDelete={handleTagListDelete(record.id)} />
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
    handleTagListChange,
    handleTagListCreate,
    handleTagListDelete,
    sorter,
  ]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleFilterChange = useCallback((filters: ExperimentFilters): void => {
    storage.set(STORAGE_FILTERS_KEY, filters);
    setFilters(filters);
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

  const sendBatchActions = useCallback((action: Action): Promise<void[] | Command> => {
    if (action === Action.OpenTensorBoard) {
      return createTensorboard({
        ids: selectedExperiments.map(experiment => experiment.id),
        type: TBSourceType.Experiment,
      });
    }
    return Promise.all(selectedExperiments
      .map(experiment => {
        switch (action) {
          case Action.Activate:
            return setExperimentState({ experimentId: experiment.id, state: RunState.Active });
          case Action.Archive:
            return archiveExperiment(experiment.id);
          case Action.Cancel:
            return setExperimentState({ experimentId: experiment.id, state: RunState.Canceled });
          case Action.Kill:
            return killExperiment({ experimentId: experiment.id });
          case Action.Pause:
            return setExperimentState({ experimentId: experiment.id, state: RunState.Paused });
          case Action.Unarchive:
            return archiveExperiment(experiment.id, false);
          default:
            return Promise.resolve();
        }
      }));
  }, [ selectedExperiments ]);

  const handleBatchAction = useCallback(async (action: Action) => {
    try {
      const result = await sendBatchActions(action);
      if (action === Action.OpenTensorBoard) {
        const url = waitPageUrl(result as Command);
        if (url) openBlank(url);
      }

      // Refetch experiment list to get updates based on batch action.
      await fetchExperiments();
    } catch (e) {
      const publicSubject = action === Action.OpenTensorBoard ?
        'Unable to Open TensorBoard for Selected Experiments' :
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
      okText: action,
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

    setPagination(prev => ({
      ...prev,
      limit: tablePagination.pageSize,
      offset: (tablePagination.current - 1) * tablePagination.pageSize,
    }));
    setSelectedRowKeys([]);
  }, [ columns, setSorter, storage ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  const handleTableRow = useCallback((record: ExperimentItem) => {
    const handleClick = (event: React.MouseEvent) => {
      if (isAlternativeAction(event)) return;
      handlePath(event, { path: record.url });
    };
    return { onAuxClick: handleClick, onClick: handleClick };
  }, []);

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
          <div className={css.filters}>
            <Toggle
              checked={filters.showArchived}
              prefixLabel="Show Archived"
              onChange={handleArchiveChange} />
            <StateSelectFilter
              showCommandStates={false}
              value={filters.states}
              onChange={handleStateChange} />
            <UserSelectFilter value={filters.username} onChange={handleUserChange} />
          </div>
        </div>
        <TableBatch message="Apply batch operations to multiple experiments." show={showBatch}>
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
        <Table
          columns={columns}
          dataSource={experiments}
          loading={{
            indicator: <Indicator />,
            spinning: !experiments,
          }}
          pagination={getFullPaginationConfig(pagination, total)}
          rowClassName={defaultRowClassName()}
          rowKey="id"
          rowSelection={{ onChange: handleTableRowSelect, selectedRowKeys }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange}
          onRow={handleTableRow}
        />
      </div>
    </Page>
  );
};

export default ExperimentList;
