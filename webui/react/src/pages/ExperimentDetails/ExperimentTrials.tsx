import { TablePaginationConfig } from 'antd';
import { FilterDropdownProps, FilterValue, SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import ActionDropdown from 'components/ActionDropdown/ActionDropdown';
import Badge, { BadgeType } from 'components/Badge';
import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Dropdown from 'components/kit/Dropdown';
import Link from 'components/Link';
import Section from 'components/Section';
import InteractiveTable, { onRightClickableCell } from 'components/Table/InteractiveTable';
import { Renderer } from 'components/Table/Table';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table/Table';
import TableBatch from 'components/Table/TableBatch';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import { terminalRunStates } from 'constants/states';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getExpTrials, openOrCreateTensorBoard } from 'services/api';
import { Experimentv1State, V1GetExperimentTrialsRequestSortBy } from 'services/api-ts-sdk';
import { encodeExperimentState } from 'services/decoder';
import { ValueOf } from 'types';
import {
  ExperimentAction as Action,
  CheckpointWorkloadExtended,
  CommandResponse,
  ExperimentBase,
  MetricsWorkload,
  RunState,
  TrialItem,
} from 'types';
import { ErrorLevel, ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { getMetricValue } from 'utils/metric';
import { routeToReactUrl } from 'utils/routes';
import { validateDetApiEnum, validateDetApiEnumList } from 'utils/service';
import { humanReadableBytes } from 'utils/string';
import { openCommandResponse } from 'utils/wait';

import css from './ExperimentTrials.module.scss';
import { configForExperiment, isOfSortKey, Settings } from './ExperimentTrials.settings';
import { columns as defaultColumns } from './ExperimentTrials.table';
import TrialsComparisonModal from './TrialsComparisonModal';

interface Props {
  experiment: ExperimentBase;
  pageRef: React.RefObject<HTMLElement>;
}

const TrialAction = {
  HyperparameterSearch: 'Hyperparameter Search',
  OpenTensorBoard: 'Open Tensorboard',
  ViewLogs: 'View Logs',
} as const;

type TrialAction = ValueOf<typeof TrialAction>;

const ExperimentTrials: React.FC<Props> = ({ experiment, pageRef }: Props) => {
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [trials, setTrials] = useState<TrialItem[]>();
  const [canceler] = useState(new AbortController());

  const config = useMemo(() => configForExperiment(experiment.id), [experiment.id]);
  const { settings, updateSettings } = useSettings<Settings>(config);

  const workspace = { id: experiment.workspaceId };
  const { canCreateExperiment, canViewExperimentArtifacts } = usePermissions();
  const canHparam = canCreateExperiment({ workspace }) && canViewExperimentArtifacts({ workspace });

  const {
    contextHolder: modalHyperparameterSearchContextHolder,
    modalOpen: openModalHyperparameterSearch,
  } = useModalHyperparameterSearch({ experiment });

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [updateSettings]);

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
    (filterProps: FilterDropdownProps) => {
      return (
        <TableFilterDropdown
          {...filterProps}
          multiple
          values={settings.state}
          onFilter={handleStateFilterApply}
          onReset={handleStateFilterReset}
        />
      );
    },
    [handleStateFilterApply, handleStateFilterReset, settings.state],
  );

  const handleOpenTensorBoard = useCallback(
    async (trial: TrialItem) => {
      openCommandResponse(
        await openOrCreateTensorBoard({ trialIds: [trial.id], workspaceId: workspace.id }),
      );
    },
    [workspace.id],
  );

  const handleViewLogs = useCallback(
    (trial: TrialItem) => {
      routeToReactUrl(paths.trialLogs(trial.id, experiment.id));
    },
    [experiment.id],
  );

  const handleHyperparameterSearch = useCallback(
    (trial: TrialItem) => {
      openModalHyperparameterSearch({ trial });
    },
    [openModalHyperparameterSearch],
  );

  const dropDownOnTrigger = useCallback(
    (trial: TrialItem) => {
      const opts: Partial<Record<TrialAction, () => Promise<void> | void>> = {
        [TrialAction.OpenTensorBoard]: () => handleOpenTensorBoard(trial),
        [TrialAction.ViewLogs]: () => handleViewLogs(trial),
        [TrialAction.HyperparameterSearch]: () => handleHyperparameterSearch(trial),
      };
      if (!canHparam) {
        delete opts[TrialAction.HyperparameterSearch];
      }
      return opts;
    },
    [canHparam, handleHyperparameterSearch, handleOpenTensorBoard, handleViewLogs],
  );

  const columns = useMemo(() => {
    const { metric } = experiment.config?.searcher || {};

    const idRenderer: Renderer<TrialItem> = (_, record) => (
      <Link path={paths.trialDetails(record.id, experiment.id)}>
        <span>Trial {record.id}</span>
      </Link>
    );

    const autoRestartsRenderer = (_: string, record: TrialItem): React.ReactNode => {
      const maxRestarts = experiment.config.maxRestarts ?? 0;
      const className = record.autoRestarts ? css.hasRestarts : undefined;
      return (
        <span className={className}>
          {record.autoRestarts}
          {maxRestarts ? `/${maxRestarts}` : ''}
        </span>
      );
    };

    const validationRenderer = (key: keyof TrialItem) => {
      return function renderer(_: string, record: TrialItem): React.ReactNode {
        const hasMetric = (obj: TrialItem[keyof TrialItem]): obj is MetricsWorkload => {
          return !!obj && typeof obj === 'object' && 'metrics' in obj;
        };

        const item: TrialItem[keyof TrialItem] = record[key];
        const value = getMetricValue(hasMetric(item) ? item : undefined, metric);
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
          TrialAction.HyperparameterSearch,
          TrialAction.ViewLogs,
        ]}
        id={experiment.id + ''}
        kind="experiment"
        onError={handleError}
        onTrigger={dropDownOnTrigger(record)}
      />
    );

    const newColumns = [...defaultColumns].map((column) => {
      column.sortOrder = null;
      if (column.key === 'checkpoint') {
        column.render = checkpointRenderer;
        column.onCell = onRightClickableCell;
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.ID) {
        column.render = idRenderer;
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.BESTVALIDATIONMETRIC) {
        column.render = validationRenderer('bestValidationMetric');
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.LATESTVALIDATIONMETRIC) {
        column.render = validationRenderer('latestValidationMetric');
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.STATE) {
        column.filterDropdown = stateFilterDropdown;
        column.isFiltered = (settings) => !!(settings as Settings).state;
        column.filters = (['ACTIVE', 'CANCELED', 'COMPLETED', 'ERROR'] as RunState[]).map(
          (value) => ({
            text: <Badge state={value} type={BadgeType.State} />,
            value,
          }),
        );
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.RESTARTS) {
        column.render = autoRestartsRenderer;
      } else if (column.key === 'actions') {
        column.render = actionRenderer;
      } else if (column.key === V1GetExperimentTrialsRequestSortBy.CHECKPOINTSIZE) {
        column.render = (value: number) => (value ? humanReadableBytes(value) : '');
      }
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });

    return newColumns;
  }, [experiment, settings, stateFilterDropdown, dropDownOnTrigger]);

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      tableFilters: Record<string, FilterValue | null>,
      tableSorter: SorterResult<TrialItem> | SorterResult<TrialItem>[],
    ) => {
      if (Array.isArray(tableSorter) || !settings) return;

      const { columnKey, order } = tableSorter as SorterResult<TrialItem>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

      const newSettings = {
        sortDesc: order === 'descend',
        sortKey: isOfSortKey(columnKey)
          ? columnKey
          : V1GetExperimentTrialsRequestSortBy.UNSPECIFIED,
        tableLimit: tablePagination.pageSize,
        tableOffset: ((tablePagination.current ?? 1) - 1) * (tablePagination.pageSize ?? 0),
      };
      updateSettings(newSettings);
    },
    [columns, settings, updateSettings],
  );

  const stateString = useMemo(() => settings.state?.join('.'), [settings.state]);
  const fetchExperimentTrials = useCallback(async () => {
    if (!settings) return;

    try {
      const states = stateString
        ?.split('.')
        .map((state) => encodeExperimentState(state as RunState));
      const { trials: experimentTrials, pagination: responsePagination } = await getExpTrials(
        {
          id: experiment.id,
          limit: settings.tableLimit,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentTrialsRequestSortBy, settings.sortKey),
          states: validateDetApiEnumList(Experimentv1State, states),
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
  }, [experiment.id, canceler, settings, stateString]);

  const sendBatchActions = useCallback(
    async (action: Action) => {
      if (!settings.row) return;

      if (action === Action.OpenTensorBoard) {
        return await openOrCreateTensorBoard({ trialIds: settings.row, workspaceId: workspace.id });
      } else if (action === Action.CompareTrials) {
        return updateSettings({ compare: true });
      }
    },
    [settings.row, updateSettings, workspace.id],
  );

  const submitBatchAction = useCallback(
    async (action: Action) => {
      try {
        const result = await sendBatchActions(action);
        if (action === Action.OpenTensorBoard && result) {
          openCommandResponse(result as CommandResponse);
        }

        // Refetch experiment list to get updates based on batch action.
        await fetchExperimentTrials();
      } catch (e) {
        const publicSubject =
          action === Action.OpenTensorBoard
            ? 'Unable to View TensorBoard for Selected Trials'
            : `Unable to ${action} Selected Trials`;
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [fetchExperimentTrials, sendBatchActions],
  );

  const { stopPolling } = usePolling(fetchExperimentTrials, { rerunOnNewFn: true });

  // Get new trials based on changes to the pagination, sorter and filters.
  useEffect(() => {
    setIsLoading(true);
    fetchExperimentTrials();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (terminalRunStates.has(experiment.state)) stopPolling({ terminateGracefully: true });
  }, [experiment.state, stopPolling]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  const handleTableRowSelect = useCallback(
    (rowKeys: React.Key[]) => {
      updateSettings({ row: rowKeys.map(Number) });
    },
    [updateSettings],
  );

  const handleTrialCompareCancel = useCallback(() => {
    updateSettings({ compare: false });
  }, [updateSettings]);

  const handleTrialUnselect = useCallback(
    (trialId: number) => {
      const trialIds = settings.row ? settings.row.filter((id) => id !== trialId) : undefined;
      updateSettings({ row: trialIds });
    },
    [settings.row, updateSettings],
  );

  const TrialActionDropdown = useCallback(
    ({
      record,
      children,
    }: {
      children: React.ReactNode;
      onVisibleChange?: (visible: boolean) => void;
      record: TrialItem;
    }) => {
      const MenuKey = {
        HyperparameterSearch: 'hyperparameter-search',
        OpenTensorboard: 'open-tensorboard',
        ViewLogs: 'view-logs',
      } as const;

      const menuItems = [
        { key: MenuKey.OpenTensorboard, label: TrialAction.OpenTensorBoard },
        { key: MenuKey.HyperparameterSearch, label: TrialAction.HyperparameterSearch },
        { key: MenuKey.ViewLogs, label: TrialAction.ViewLogs },
      ];

      const handleDropdown = (key: string) => {
        switch (key) {
          case MenuKey.HyperparameterSearch:
            handleHyperparameterSearch(record);
            break;
          case MenuKey.OpenTensorboard:
            handleOpenTensorBoard(record);
            break;
          case MenuKey.ViewLogs:
            handleViewLogs(record);
            break;
        }
      };

      return (
        <Dropdown isContextMenu menu={menuItems} onClick={handleDropdown}>
          {children}
        </Dropdown>
      );
    },
    [handleHyperparameterSearch, handleOpenTensorBoard, handleViewLogs],
  );

  return (
    <div className={css.base}>
      <Section>
        <TableBatch
          actions={[
            { label: Action.OpenTensorBoard, value: Action.OpenTensorBoard },
            { label: Action.CompareTrials, value: Action.CompareTrials },
          ]}
          selectedRowCount={(settings.row ?? []).length}
          onAction={(action) => submitBatchAction(action as Action)}
          onClear={clearSelected}
        />
        <InteractiveTable<TrialItem, Settings>
          columns={columns}
          containerRef={pageRef}
          ContextMenu={TrialActionDropdown}
          dataSource={trials}
          loading={isLoading}
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
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings}
          onChange={handleTableChange}
        />
      </Section>
      {settings.compare && (
        <TrialsComparisonModal
          experiment={experiment}
          trialIds={settings.row ?? []}
          visible={settings.compare}
          onCancel={handleTrialCompareCancel}
          onUnselect={handleTrialUnselect}
        />
      )}
      {modalHyperparameterSearchContextHolder}
    </div>
  );
};

export default ExperimentTrials;
