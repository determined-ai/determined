import { Button, Tooltip } from 'antd';
import { FilterDropdownProps, SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import Link from 'components/Link';
import ResponsiveTable from 'components/ResponsiveTable';
import tableCss from 'components/ResponsiveTable.module.scss';
import Section from 'components/Section';
import {
  defaultRowClassName, getFullPaginationConfig, MINIMUM_PAGE_SIZE,
} from 'components/Table';
import { Renderer } from 'components/Table';
import TableBatch from 'components/TableBatch';
import TableFilterDropdown from 'components/TableFilterDropdown';
import TrialActionDropdown from 'components/TrialActionDropdown';
import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import useStorage from 'hooks/useStorage';
import { parseUrl } from 'routes/utils';
import { paths } from 'routes/utils';
import { getExpTrials, openOrCreateTensorboard } from 'services/api';
import {
  Determinedexperimentv1State, V1GetExperimentTrialsRequestSortBy,
} from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { ApiSorter } from 'services/types';
import { validateDetApiEnum, validateDetApiEnumList } from 'services/utils';
import {
  CheckpointWorkloadExtended, CommandTask, ExperimentBase,
  Pagination, RunState, TrialFilters, TrialItem,
} from 'types';
import { getMetricValue, terminalRunStates } from 'utils/types';
import { openCommand } from 'wait';

import { columns as defaultColumns } from './ExperimentTrials.table';

interface Props {
  experiment: ExperimentBase;
}

enum Action {
  OpenTensorBoard = 'OpenTensorboard',
}

const URL_ALL = 'all';

const STORAGE_PATH = 'experiment-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';
const STORAGE_FILTERS_KEY = 'filters';

const defaultFilters: TrialFilters = { states: undefined };

const defaultSorter: ApiSorter<V1GetExperimentTrialsRequestSortBy> = {
  descend: true,
  key: V1GetExperimentTrialsRequestSortBy.ID,
};

const ExperimentTrials: React.FC<Props> = ({ experiment }: Props) => {
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initFilters = storage.getWithDefault(STORAGE_FILTERS_KEY, { ...defaultFilters });
  const initSorter = storage.getWithDefault(STORAGE_SORTER_KEY, { ...defaultSorter });
  const [ filters, setFilters ] = useState<TrialFilters>(initFilters);
  const [ isUrlParsed, setIsUrlParsed ] = useState(false);
  const [ pagination, setPagination ] = useState<Pagination>({ limit: initLimit, offset: 0 });
  const [ total, setTotal ] = useState(0);
  const [ selectedRowKeys, setSelectedRowKeys ] = useState<number[]>([]);
  const [ sorter, setSorter ] = useState(initSorter);
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointWorkloadExtended>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ trials, setTrials ] = useState<TrialItem[]>();
  const [ canceler ] = useState(new AbortController());

  /*
   * When filters changes update the page URL.
   */
  useEffect(() => {
    if (!isUrlParsed) return;

    const searchParams = new URLSearchParams;
    const url = parseUrl(window.location.href);

    // limit
    searchParams.append('limit', pagination.limit.toString());

    // offset
    searchParams.append('offset', pagination.offset.toString());

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

    // selected rows
    if (selectedRowKeys && selectedRowKeys.length > 0) {
      selectedRowKeys.forEach(rowKey => searchParams.append('row', String(rowKey)));
    }

    window.history.pushState(
      {},
      '',
      url.origin + url.pathname + '?' + searchParams.toString(),
    );
  }, [ filters, isUrlParsed, pagination, selectedRowKeys, sorter ]);

  /*
   * On first load: if filters are specified in URL, override default.
   */
  useEffect(() => {
    if (isUrlParsed) return;

    // If search params are not set, we default to user preferences
    const url = parseUrl(window.location.href);
    if (url.search === '') {
      setIsUrlParsed(true);
      return;
    }

    const urlSearchParams = url.searchParams;

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

    // sortDesc
    const sortDesc = urlSearchParams.get('sortDesc');
    if (sortDesc != null) {
      sorter.descend = (sortDesc === '1');
    }

    // sortKey
    const sortKey = urlSearchParams.get('sortKey');
    if (sortKey != null &&
      Object.values(V1GetExperimentTrialsRequestSortBy).includes(sortKey)) {
      sorter.key = sortKey as unknown as V1GetExperimentTrialsRequestSortBy;
    }

    // states
    const state = urlSearchParams.getAll('state');
    if (state != null) {
      filters.states = (state.includes(URL_ALL) ? undefined : state);
    }

    // selected rows
    const rows = urlSearchParams.getAll('row');
    if (rows != null) {
      setSelectedRowKeys(rows.map(row => parseInt(row)));
    }

    setFilters(filters);
    setIsUrlParsed(true);
    setPagination(pagination);
    setSorter(sorter);
  }, [ filters, isUrlParsed, pagination, sorter ]);

  const clearSelected = useCallback(() => {
    setSelectedRowKeys([]);
  }, []);

  const handleFilterChange = useCallback((filters: TrialFilters): void => {
    storage.set(STORAGE_FILTERS_KEY, filters);
    setFilters(filters);
    setPagination(prev => ({ ...prev, offset: 0 }));
    clearSelected();
  }, [ clearSelected, setFilters, storage ]);

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

  const columns = useMemo(() => {
    const { metric } = experiment.config?.searcher || {};

    const idRenderer: Renderer<TrialItem> = (_, record) => (
      <Link path={paths.trialDetails(record.id, experiment.id)}>
        <span>Trial {record.id}</span>
      </Link>
    );

    const validationRenderer = (key: string) => {
      return function renderer (_: string, record: TrialItem): React.ReactNode {
        /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
        const value = getMetricValue((record as any)[key], metric);
        return value && <HumanReadableFloat num={value} />;
      };
    };

    const checkpointRenderer = (_: string, record: TrialItem): React.ReactNode => {
      if (!record.bestAvailableCheckpoint) return;
      const checkpoint: CheckpointWorkloadExtended = {
        ...record.bestAvailableCheckpoint,
        experimentId: experiment.id,
        trialId: record.id,
      };
      return (
        <Tooltip title="View Checkpoint">
          <Button
            aria-label="View Checkpoint"
            icon={<Icon name="checkpoint" />}
            onClick={e => handleCheckpointShow(e, checkpoint)} />
        </Tooltip>
      );
    };

    const actionRenderer = (_: string, record: TrialItem): React.ReactNode => {
      return <TrialActionDropdown experimentId={experiment.id} trial={record} />;
    };

    const newColumns = [ ...defaultColumns ].map(column => {
      column.sortOrder = null;
      if (column.key === 'checkpoint') {
        column.render = checkpointRenderer;
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.ID) {
        column.render = idRenderer;
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.BESTVALIDATIONMETRIC) {
        column.render = validationRenderer('bestValidationMetric');
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.LATESTVALIDATIONMETRIC) {
        column.render = validationRenderer('latestValidationMetric');
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.STATE) {
        column.filterDropdown = stateFilterDropdown;
        column.onHeaderCell = () => filters.states ? { className: tableCss.headerFilterOn } : {},
        column.filters = ([ 'ACTIVE', 'CANCELED', 'COMPLETED', 'ERROR' ] as RunState[])
          .map((value) => ({
            text: <Badge state={value} type={BadgeType.State} />,
            value,
          }));
      } else if (column.key === 'actions') {
        column.render = actionRenderer;
      }
      if (column.key === sorter.key) {
        column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      }
      return column;
    });

    return newColumns;
  }, [ experiment.config, experiment.id, sorter, stateFilterDropdown, filters ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, sorter) => {
    if (Array.isArray(sorter)) return;

    const { columnKey, order } = sorter as SorterResult<TrialItem>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    storage.set(STORAGE_SORTER_KEY, { descend: order === 'descend', key: columnKey as string });
    setSorter({
      descend: order === 'descend',
      key: columnKey as V1GetExperimentTrialsRequestSortBy,
    });

    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPagination(prev => ({
      ...prev,
      limit: tablePagination.pageSize,
      offset: (tablePagination.current - 1) * tablePagination.pageSize,
    }));
  }, [ columns, setSorter, storage ]);

  const handleCheckpointShow = (
    event: React.MouseEvent,
    checkpoint: CheckpointWorkloadExtended,
  ) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };

  const handleCheckpointDismiss = useCallback(() => setShowCheckpoint(false), []);

  const fetchExperimentTrials = useCallback(async () => {
    try {
      const states = (filters.states || []).map(state => encodeExperimentState(state as RunState));
      const { trials: experimentTrials, pagination: responsePagination } = await getExpTrials(
        {
          id: experiment.id,
          limit: pagination.limit,
          offset: pagination.offset,
          orderBy: sorter.descend ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentTrialsRequestSortBy, sorter.key),
          states: validateDetApiEnumList(Determinedexperimentv1State, states),
        },
        { signal: canceler.signal },
      );
      setTotal(responsePagination?.total || 0);
      setTrials(experimentTrials);
      setIsLoading(false);
    } catch (e) {
      handleError({
        message: `Unable to fetch experiments ${experiment.id} trials.`,
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [ experiment.id, canceler, pagination, sorter, filters ]);

  const sendBatchActions = useCallback((action: Action): Promise<void[] | CommandTask> => {
    if (action === Action.OpenTensorBoard) {
      return openOrCreateTensorboard(
        { trialIds: selectedRowKeys },
      );
    }
    return Promise.all([]);
  }, [ selectedRowKeys ]);

  const handleBatchAction = useCallback(async (action: Action) => {
    try {
      const result = await sendBatchActions(action);
      if (action === Action.OpenTensorBoard && result) {
        openCommand(result as CommandTask);
      }

      // Refetch experiment list to get updates based on batch action.
      await fetchExperimentTrials();
    } catch (e) {
      const publicSubject = action === Action.OpenTensorBoard ?
        'Unable to View TensorBoard for Selected Trials' :
        `Unable to ${action} Selected Trials`;
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
  }, [ fetchExperimentTrials, sendBatchActions ]);

  const { stopPolling } = usePolling(fetchExperimentTrials);

  // Get new trials based on changes to the pagination, sorter and filters.
  useEffect(() => {
    fetchExperimentTrials();
    setIsLoading(true);
  }, [ fetchExperimentTrials, filters, pagination, sorter ]);

  useEffect(() => {
    if (terminalRunStates.has(experiment.state)) stopPolling({ terminateGracefully: true });
  }, [ experiment.state, stopPolling ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const handleTableRowSelect = useCallback(rowKeys => setSelectedRowKeys(rowKeys), []);

  return (
    <>
      <Section>
        <TableBatch selectedRowCount={selectedRowKeys.length} onClear={clearSelected}>
          <Button onClick={(): Promise<void> => handleBatchAction(Action.OpenTensorBoard)}>
            View in TensorBoard
          </Button>
        </TableBatch>
        <ResponsiveTable
          columns={columns}
          dataSource={trials}
          loading={isLoading}
          pagination={getFullPaginationConfig(pagination, total)}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="id"
          rowSelection={{
            onChange: handleTableRowSelect,
            preserveSelectedRowKeys: true,
            selectedRowKeys,
          }}
          showSorterTooltip={false}
          size="small"
          onChange={handleTableChange} />
      </Section>
      {activeCheckpoint && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experiment.config}
        show={showCheckpoint}
        title={`Best Checkpoint for Trial ${activeCheckpoint.trialId}`}
        onHide={handleCheckpointDismiss} />}
    </>
  );
};

export default ExperimentTrials;
