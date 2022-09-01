import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import InteractiveTable, { InteractiveTableSettings } from 'components/InteractiveTable';
import Section from 'components/Section';
import { defaultRowClassName, getFullPaginationConfig,
  HumanReadableNumberRenderer } from 'components/Table';
import TableBatch from 'components/TableBatch';
import TableFilterDropdown from 'components/TableFilterDropdown';
import useModalCheckpointRegister from 'hooks/useModal/Checkpoint/useModalCheckpointRegister';
import useModalModelCreate from 'hooks/useModal/Model/useModalModelCreate';
import usePolling from 'hooks/usePolling';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
import { getExperimentCheckpoints } from 'services/api';
import { Determinedcheckpointv1State,
  V1GetExperimentCheckpointsRequestSortBy } from 'services/api-ts-sdk';
import { encodeCheckpointState } from 'services/decoder';
import ActionDropdown from 'shared/components/ActionDropdown/ActionDropdown';
import { ModalCloseReason } from 'shared/hooks/useModal/useModal';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { validateDetApiEnum, validateDetApiEnumList } from 'shared/utils/service';
import {
  CheckpointState, CoreApiGenericCheckpoint, ExperimentBase,
} from 'types';
import handleError from 'utils/error';

import settingsConfig, { Settings } from './ExperimentCheckpoints.settings';
import { columns as defaultColumns } from './ExperimentCheckpoints.table';
import css from './ExperimentTrials.module.scss';

interface Props {
  experiment: ExperimentBase;
  pageRef: React.RefObject<HTMLElement>;
}

const checkpointAction = {
  Delete: 'Delete',
  Register: 'Register',
} as const;

type CheckpointAction = typeof checkpointAction[keyof typeof checkpointAction];

const ExperimentCheckpoints: React.FC<Props> = ({ experiment, pageRef }: Props) => {
  const [ total, setTotal ] = useState(0);
  const [ isLoading, setIsLoading ] = useState(true);
  const [ checkpoints, setCheckpoints ] = useState<CoreApiGenericCheckpoint[]>();
  const [ canceler ] = useState(new AbortController());

  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

  const {
    contextHolder: modalModelCreateContextHolder,
    modalOpen: openModalCreateModel,
  } = useModalModelCreate();

  const handleOnCloseCheckpointRegister = useCallback((
    reason?: ModalCloseReason,
    checkpointUuid?: string,
  ) => {
    if (checkpointUuid) openModalCreateModel({ checkpointUuid });
  }, [ openModalCreateModel ]);

  const {
    contextHolder: modalCheckpointRegisterContextHolder,
    modalOpen: openModalCheckpointRegister,
  } = useModalCheckpointRegister({ onClose: handleOnCloseCheckpointRegister });

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [ updateSettings ]);

  const handleStateFilterApply = useCallback((states: string[]) => {
    updateSettings({
      row: undefined,
      state: states.length !== 0 ? states as CheckpointState[] : undefined,
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

  const handleRegisterCheckpoint = useCallback((checkpoint: CoreApiGenericCheckpoint) => {
    openModalCheckpointRegister({ checkpointUuid: checkpoint.uuid });
  }, [ openModalCheckpointRegister ]);

  const dropDownOnTrigger = useCallback((checkpoint: CoreApiGenericCheckpoint) => {
    return { [checkpointAction.Register]: () => handleRegisterCheckpoint(checkpoint) };
  }, [ handleRegisterCheckpoint ]);

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: CoreApiGenericCheckpoint): React.ReactNode => (
      <ActionDropdown<CheckpointAction>
        actionOrder={[
          'Register',
        ]}
        id={record.uuid}
        kind="checkpoint"
        onError={handleError}
        onTrigger={dropDownOnTrigger(record)}
      />
    );

    const checkpointRenderer = (_: string, record: CoreApiGenericCheckpoint): React.ReactNode => {
      return (
        <CheckpointModalTrigger
          checkpoint={record}
          experiment={experiment}
          title={`Checkpoint ${record.uuid}`}
        />
      );
    };

    const newColumns = [ ...defaultColumns ].map((column) => {
      column.sortOrder = null;
      if (column.key === 'checkpoint') {
        column.render = checkpointRenderer;
      } else if (column.key === V1GetExperimentCheckpointsRequestSortBy.STATE) {
        column.filterDropdown = stateFilterDropdown;
        column.isFiltered = (settings) => !!(settings as Settings).state;
        column.filters = (Object.values(CheckpointState))
          .filter((value) => value !== CheckpointState.Unspecified)
          .map((value) => ({
            text: <Badge state={value} type={BadgeType.State} />,
            value,
          }));
      } else if (column.key === V1GetExperimentCheckpointsRequestSortBy.SEARCHERMETRIC) {
        column.render = HumanReadableNumberRenderer;
        column.title = `Searcher Metric (${experiment.config.searcher.metric})`;
      } else if (column.key === 'actions') {
        column.render = actionRenderer;
      }
      if (column.key === settings.sortKey) {
        column.sortOrder = settings.sortDesc ? 'descend' : 'ascend';
      }
      return column;
    });

    return newColumns;
  }, [ dropDownOnTrigger, experiment, settings.sortDesc, settings.sortKey, stateFilterDropdown ]);

  const fetchExperimentCheckpoints = useCallback(async () => {
    try {
      const states = (settings.state ?? []).map((state) => (
        encodeCheckpointState(state)
      ));
      const response = await getExperimentCheckpoints(
        {
          id: experiment.id,
          limit: settings.tableLimit,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(V1GetExperimentCheckpointsRequestSortBy, settings.sortKey),
          states: validateDetApiEnumList(Determinedcheckpointv1State, states),
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

  const submitBatchAction = useCallback(async (action: CheckpointAction) => {
    try {
      // TODO: Actions

      // Refetch experiment list to get updates based on batch action.
      await fetchExperimentCheckpoints();
    } catch (e) {
      const publicSubject = `Unable to ${action} Selected Checkpoints`;
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: 'Please try again later.',
        publicSubject,
        silent: false,
        type: ErrorType.Server,
      });
    }
  }, [ fetchExperimentCheckpoints ]);

  usePolling(fetchExperimentCheckpoints, { rerunOnNewFn: true });

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
    return () => canceler.abort();
  }, [ canceler ]);

  // const handleTableRowSelect = useCallback((rowKeys) => {
  //   updateSettings({ row: rowKeys });
  // }, [ updateSettings ]);

  return (
    <div className={css.base}>
      <Section>
        <TableBatch
          actions={[
            { label: checkpointAction.Register, value: checkpointAction.Register },
            { label: checkpointAction.Delete, value: checkpointAction.Delete },
          ]}
          selectedRowCount={(settings.row ?? []).length}
          onAction={(action) => submitBatchAction(action)}
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
          // rowSelection={{
          //   onChange: handleTableRowSelect,
          //   preserveSelectedRowKeys: true,
          //   selectedRowKeys: settings.row ?? [],
          // }}
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
        />
      </Section>
      {modalModelCreateContextHolder}
      {modalCheckpointRegisterContextHolder}
    </div>
  );
};

export default ExperimentCheckpoints;
