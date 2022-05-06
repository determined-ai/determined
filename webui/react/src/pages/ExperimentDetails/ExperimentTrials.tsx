import { FilterDropdownProps, SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import ActionDropdown from 'components/ActionDropdown';
import Badge, { BadgeType } from 'components/Badge';
import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import ResponsiveTable from 'components/ResponsiveTable';
import tableCss from 'components/ResponsiveTable.module.scss';
import Section from 'components/Section';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table';
import { Renderer } from 'components/Table';
import TableBatch from 'components/TableBatch';
import TableFilterDropdown from 'components/TableFilterDropdown';
import { terminalRunStates } from 'constants/states';
import usePolling from 'hooks/usePolling';
import useSettings from 'hooks/useSettings';
import { paths, routeToReactUrl } from 'routes/utils';
import { getExpTrials, openOrCreateTensorBoard } from 'services/api';
import {
  Determinedexperimentv1State, V1GetExperimentTrialsRequestSortBy,
} from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { validateDetApiEnum, validateDetApiEnumList } from 'services/utils';
import {
  ExperimentAction as Action, CheckpointWorkloadExtended, CommandTask, ExperimentBase,
  RunState, TrialItem,
} from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { getMetricValue } from 'utils/metric';
import { openCommand } from 'wait';

import css from './ExperimentTrials.module.scss';
import settingsConfig, { Settings } from './ExperimentTrials.settings';
import { columns as defaultColumns } from './ExperimentTrials.table';
import TrialsComparisonModal from './TrialsComparisonModal';

interface Props {
  experiment: ExperimentBase;
}

enum TrialAction {
  OpenTensorBoard = 'Open Tensorboard',
  ViewLogs = 'View Logs',
}

const ExperimentTrials: React.FC<Props> = ({ experiment }: Props) => {
  const [ total, setTotal ] = useState(0);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ trials, setTrials ] = useState<TrialItem[]>();
  const [ canceler ] = useState(new AbortController());

  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [ updateSettings ]);

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

  const dropDownOnTrigger = useCallback((trial: TrialItem) => {
    return {
      [TrialAction.OpenTensorBoard]: async () => {
        openCommand(await openOrCreateTensorBoard({ trialIds: [ trial.id ] }));
      },
      [TrialAction.ViewLogs]: () => {
        routeToReactUrl(paths.trialLogs(trial.id, experiment.id));
      },
    };
  }, [ experiment.id ]);

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
        return <HumanReadableNumber num={value} />;
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
        <CheckpointModalTrigger
          checkpoint={checkpoint}
          experiment={experiment}
          title={`Best Checkpoint for Trial ${checkpoint.trialId}`}
        />
      );
    };

    const actionRenderer = (_: string, record: TrialItem): React.ReactNode => (
      <ActionDropdown<TrialAction>
        actionOrder={[
          TrialAction.OpenTensorBoard,
          TrialAction.ViewLogs,
        ]}
        id={experiment.id + ''}
        kind="experiment"
        onTrigger={dropDownOnTrigger(record)}
      />
    );

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
        column.onHeaderCell = () => settings.state ? { className: tableCss.headerFilterOn } : {},
        column.filters = ([ 'ACTIVE', 'CANCELED', 'COMPLETED', 'ERROR' ] as RunState[])
          .map((value) => ({
            text: <Badge state={value} type={BadgeType.State} />,
            value,
          }));
      } else if (column.key === 'actions') {
        column.render = actionRenderer;
      }
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });

    return newColumns;
  }, [ experiment, settings, stateFilterDropdown, dropDownOnTrigger ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<TrialItem>;
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

  const fetchExperimentTrials = useCallback(async () => {
    try {
      const states = (settings.state || []).map(state => encodeExperimentState(state as RunState));
      const { trials: experimentTrials, pagination: responsePagination } = await getExpTrials(
        {
          id: experiment.id,
          limit: settings.tableLimit,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentTrialsRequestSortBy, settings.sortKey),
          states: validateDetApiEnumList(Determinedexperimentv1State, states),
        },
        { signal: canceler.signal },
      );
      setTotal(responsePagination?.total || 0);
      setTrials(experimentTrials);
      setIsLoading(false);
    } catch (e) {
      handleError(e, {
        publicSubject: `Unable to fetch experiments ${experiment.id} trials.`,
        silent: true,
        type: ErrorType.Api,
      });
      setIsLoading(false);
    }
  }, [
    experiment.id,
    canceler,
    settings.sortDesc,
    settings.sortKey,
    settings.state,
    settings.tableLimit,
    settings.tableOffset,
  ]);

  const sendBatchActions = useCallback(async (action: Action) => {
    if (action === Action.OpenTensorBoard) {
      return await openOrCreateTensorBoard({ trialIds: settings.row });
    } else if (action === Action.CompareTrials) {
      return updateSettings({ compare: true });
    }
  }, [ settings.row, updateSettings ]);

  const submitBatchAction = useCallback(async (action: Action) => {
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
      handleError(e, {
        level: ErrorLevel.Error,
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
  }, [
    fetchExperimentTrials,
    settings.sortDesc,
    settings.sortKey,
    settings.state,
    settings.tableLimit,
    settings.tableOffset,
  ]);

  useEffect(() => {
    if (terminalRunStates.has(experiment.state)) stopPolling({ terminateGracefully: true });
  }, [ experiment.state, stopPolling ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  const handleTableRowSelect = useCallback(rowKeys => {
    updateSettings({ row: rowKeys });
  }, [ updateSettings ]);

  const handleTrialCompareCancel = useCallback(() => {
    updateSettings({ compare: false });
  }, [ updateSettings ]);

  const handleTrialUnselect = useCallback((trialId: number) => {
    const trialIds = settings.row ? settings.row.filter(id => id !== trialId) : undefined;
    updateSettings({ row: trialIds });
  }, [ settings.row, updateSettings ]);

  return (
    <div className={css.base}>
      <Section>
        <TableBatch
          actions={[
            { label: Action.OpenTensorBoard, value: Action.OpenTensorBoard },
            { label: Action.CompareTrials, value: Action.CompareTrials },
          ]}
          selectedRowCount={(settings.row ?? []).length}
          onAction={action => submitBatchAction(action as Action)}
          onClear={clearSelected}
        />
        <ResponsiveTable
          columns={columns}
          dataSource={trials}
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
          onChange={handleTableChange}
        />
      </Section>
      {settings.compare && (
        <TrialsComparisonModal
          experiment={experiment}
          trials={settings.row ?? []}
          visible={settings.compare}
          onCancel={handleTrialCompareCancel}
          onUnselect={handleTrialUnselect}
        />
      )}
    </div>
  );
};

export default ExperimentTrials;
