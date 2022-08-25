import { Dropdown, Menu } from 'antd';
import type { MenuProps } from 'antd';
import { FilterDropdownProps, SorterResult } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import HumanReadableNumber from 'components/HumanReadableNumber';
import InteractiveTable, { InteractiveTableSettings } from 'components/InteractiveTable';
import Link from 'components/Link';
import Section from 'components/Section';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table';
import { Renderer } from 'components/Table';
import TableBatch from 'components/TableBatch';
import TableFilterDropdown from 'components/TableFilterDropdown';
import { terminalRunStates } from 'constants/states';
import usePolling from 'hooks/usePolling';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getExperimentCheckpoints, openOrCreateTensorBoard } from 'services/api';
import { V1GetExperimentCheckpointsRequestSortBy } from 'services/api-ts-sdk';
import { encodeCheckpointState } from 'services/decoder';
import ActionDropdown from 'shared/components/ActionDropdown/ActionDropdown';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { routeToReactUrl } from 'shared/utils/routes';
import { validateDetApiEnum, validateDetApiEnumList } from 'shared/utils/service';
import {
  ExperimentAction as Action, CheckpointPagination, CheckpointState,
  CommandTask, CoreApiGenericCheckpoint, ExperimentBase,
} from 'types';
import handleError from 'utils/error';
import { getMetricValue } from 'utils/metric';
import { openCommand } from 'utils/wait';

import settingsConfig, { Settings } from './ExperimentCheckpoints.settings';
import { columns as defaultColumns } from './ExperimentCheckpoints.table';
import css from './ExperimentTrials.module.scss';

interface Props {
  experiment: ExperimentBase;
  pageRef: React.RefObject<HTMLElement>;
}

const ExperimentCheckpoints: React.FC<Props> = ({ experiment, pageRef }: Props) => {
  const [ total, setTotal ] = useState(0);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ checkpoints, setCheckpoints ] = useState<CoreApiGenericCheckpoint[]>();
  const [ canceler ] = useState(new AbortController());

  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [ updateSettings ]);

  const columns = useMemo(() => {

    const idRenderer: Renderer<CoreApiGenericCheckpoint> = (_, record) => (
      //<Link path={paths.trialDetails(record.id, experiment.id)}>
      <CheckpointModalTrigger
        checkpoint={record}
        experiment={experiment}
        title={`Checkpoint ${record.uuid}`}>
        <span>{record.uuid}</span>
      </CheckpointModalTrigger>
      //</Link>
    );

    const actionRenderer = (_: string, record: CoreApiGenericCheckpoint): React.ReactNode => (
      // <ActionDropdown<TrialAction>
      //   actionOrder={[
      //     TrialAction.OpenTensorBoard,
      //     TrialAction.HyperparameterSearch,
      //     TrialAction.ViewLogs,
      //   ]}
      //   id={experiment.id + ''}
      //   kind="experiment"
      //   onError={handleError}
      //   onTrigger={dropDownOnTrigger(record)}
      // />
      <></>
    );

    const newColumns = [ ...defaultColumns ].map((column) => {
      column.sortOrder = null;
      if (column.key === 'checkpoint') {
        //column.render = checkpointRenderer;
      } else if (column.key === V1GetExperimentCheckpointsRequestSortBy.UUID) {
        column.render = idRenderer;
      } else if (column.key === V1GetExperimentCheckpointsRequestSortBy.STATE) {
        //column.filterDropdown = stateFilterDropdown;
        column.isFiltered = (settings) => !!(settings as Settings).state;
        column.filters = ([ 'ACTIVE', 'COMPLETED', 'DELETED', 'ERROR' ] as CheckpointState[])
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
  }, [ experiment, settings.sortDesc, settings.sortKey ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, tableSorter) => {
    if (Array.isArray(tableSorter)) return;

    const { columnKey, order } = tableSorter as SorterResult<CoreApiGenericCheckpoint>;
    if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

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

  const fetchExperimentCheckpoints = useCallback(async () => {
    try {
      const states = (settings.state ?? []).map((state) => (
        encodeCheckpointState(state as CheckpointState)
      ));
      const response = await getExperimentCheckpoints(
        {
          id: experiment.id,
          limit: settings.tableLimit,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentCheckpointsRequestSortBy, settings.sortKey),
          states: validateDetApiEnumList(CheckpointState, states),
        },
        { signal: canceler.signal },
      );
      setTotal(response.pagination.total ?? 0);
      setCheckpoints(response.checkpoints);
    } catch (e) {
      handleError(e, {
        publicSubject: `Unable to fetch experiment ${experiment.id} checkpoints.`,
        silent: true,
        type: ErrorType.Api,
      });
    } finally {
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
    }
  }, [ settings.row ]);

  const submitBatchAction = useCallback(async (action: Action) => {
    try {
      const result = await sendBatchActions(action);
      if (action === Action.OpenTensorBoard && result) {
        openCommand(result as CommandTask);
      }

      // Refetch experiment list to get updates based on batch action.
      await fetchExperimentCheckpoints();
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
  }, [ fetchExperimentCheckpoints, sendBatchActions ]);

  const { stopPolling } = usePolling(fetchExperimentCheckpoints, { rerunOnNewFn: true });

  // Get new trials based on changes to the pagination, sorter and filters.
  useEffect(() => {
    fetchExperimentCheckpoints();
    setIsLoading(true);
  }, [
    fetchExperimentCheckpoints,
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

  const handleTableRowSelect = useCallback((rowKeys) => {
    updateSettings({ row: rowKeys });
  }, [ updateSettings ]);

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
        <InteractiveTable
          columns={columns}
          containerRef={pageRef}
          dataSource={checkpoints}
          loading={isLoading}
          pagination={getFullPaginationConfig({
            limit: settings.tableLimit,
            offset: settings.tableOffset,
          }, total)}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="uuid"
          rowSelection={{
            onChange: handleTableRowSelect,
            preserveSelectedRowKeys: true,
            selectedRowKeys: settings.row ?? [],
          }}
          settings={settings as InteractiveTableSettings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
          onChange={handleTableChange}
        />
      </Section>
    </div>
  );
};

export default ExperimentCheckpoints;
