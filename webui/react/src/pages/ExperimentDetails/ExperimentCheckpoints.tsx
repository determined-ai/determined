import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { Key, useCallback, useEffect, useMemo, useState } from 'react';

import ActionDropdown from 'components/ActionDropdown';
import Badge, { BadgeType } from 'components/Badge';
import CheckpointModalTrigger from 'components/CheckpointModalTrigger';
import { useModal } from 'components/kit/Modal';
import useConfirm from 'components/kit/useConfirm';
import ModelCreateModal from 'components/ModelCreateModal';
import Section from 'components/Section';
import InteractiveTable, { ContextMenuProps } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  HumanReadableNumberRenderer,
} from 'components/Table/Table';
import TableBatch from 'components/Table/TableBatch';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import useModalCheckpointRegister from 'hooks/useModal/Checkpoint/useModalCheckpointRegister';
import { ModalCloseReason } from 'hooks/useModal/useModal';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { getExperimentCheckpoints } from 'services/api';
import { Checkpointv1SortBy, Checkpointv1State } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { encodeCheckpointState } from 'services/decoder';
import { readStream } from 'services/utils';
import {
  checkpointAction,
  CheckpointAction,
  CheckpointState,
  CoreApiGenericCheckpoint,
  ExperimentBase,
  RecordKey,
} from 'types';
import { canActionCheckpoint, getActionsForCheckpointsUnion } from 'utils/checkpoint';
import { ensureArray } from 'utils/data';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { validateDetApiEnum, validateDetApiEnumList } from 'utils/service';
import { pluralizer } from 'utils/string';

import { configForExperiment, Settings } from './ExperimentCheckpoints.settings';
import { columns as defaultColumns } from './ExperimentCheckpoints.table';

interface Props {
  experiment: ExperimentBase;
  pageRef: React.RefObject<HTMLElement>;
}

const batchActions = [checkpointAction.Register, checkpointAction.Delete];

const ExperimentCheckpoints: React.FC<Props> = ({ experiment, pageRef }: Props) => {
  const confirm = useConfirm();
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [checkpoints, setCheckpoints] = useState<CoreApiGenericCheckpoint[]>();
  const [canceler] = useState(new AbortController());

  const config = useMemo(() => configForExperiment(experiment.id), [experiment.id]);
  const { settings, updateSettings } = useSettings<Settings>(config);

  const modelCreateModal = useModal(ModelCreateModal);

  const {
    contextHolder: modalCheckpointRegisterContextHolder,
    modalOpen: openModalCheckpointRegister,
  } = useModalCheckpointRegister({
    onClose: (reason?: ModalCloseReason, checkpoints?: string[]) => {
      // TODO: fix the behavior along with checkpoint modal migration
      // It used to open checkpoint modal again after creating a model,
      // but it doesn't with new create model modal since we don't use context holder anymore.
      // This should be able to fix it along with checkpoint modal migration.
      if (checkpoints) modelCreateModal.open();
    },
  });

  const handleOnCloseCreateModel = useCallback(
    (reason?: ModalCloseReason, checkpoints?: string[], modelName?: string) => {
      if (checkpoints) openModalCheckpointRegister({ checkpoints, selectedModelName: modelName });
    },
    [openModalCheckpointRegister],
  );

  const clearSelected = useCallback(() => {
    updateSettings({ row: undefined });
  }, [updateSettings]);

  const handleStateFilterApply = useCallback(
    (states: string[]) => {
      updateSettings({
        row: undefined,
        state: states.length !== 0 ? (states as CheckpointState[]) : undefined,
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

  const handleRegisterCheckpoint = useCallback(
    (checkpoints: string[]) => {
      openModalCheckpointRegister({ checkpoints });
    },
    [openModalCheckpointRegister],
  );

  const handleDelete = useCallback((checkpoints: string[]) => {
    readStream(
      detApi.Checkpoint.deleteCheckpoints({
        checkpointUuids: checkpoints,
      }),
    );
  }, []);

  const handleDeleteCheckpoint = useCallback(
    (checkpoints: string[]) => {
      const content = `Are you sure you want to request checkpoint deletion for ${
        checkpoints.length
      }
      ${pluralizer(
        checkpoints.length,
        'checkpoint',
      )}? This action may complete or fail without further notification.`;

      confirm({
        content,
        danger: true,
        okText: 'Request Delete',
        onConfirm: () => handleDelete(checkpoints),
        onError: handleError,
        title: 'Confirm Checkpoint Deletion',
      });
    },
    [confirm, handleDelete],
  );

  const dropDownOnTrigger = useCallback(
    (checkpoints: string | string[]) => {
      const checkpointsArr = ensureArray(checkpoints);
      return {
        [checkpointAction.Register]: () => handleRegisterCheckpoint(checkpointsArr),
        [checkpointAction.Delete]: () => handleDeleteCheckpoint(checkpointsArr),
      };
    },
    [handleDeleteCheckpoint, handleRegisterCheckpoint],
  );

  const CheckpointActionDropdown: React.FC<ContextMenuProps<CoreApiGenericCheckpoint>> =
    useCallback(
      ({ record, children }) => {
        return (
          <ActionDropdown<CheckpointAction>
            actionOrder={batchActions}
            danger={{ [checkpointAction.Delete]: true }}
            disabled={{
              [checkpointAction.Register]: !canActionCheckpoint(checkpointAction.Register, record),
              [checkpointAction.Delete]: !canActionCheckpoint(checkpointAction.Delete, record),
            }}
            id={record.uuid}
            isContextMenu
            kind="checkpoint"
            onError={handleError}
            onTrigger={dropDownOnTrigger(record.uuid)}>
            {children}
          </ActionDropdown>
        );
      },
      [dropDownOnTrigger],
    );

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: CoreApiGenericCheckpoint): React.ReactNode => (
      <ActionDropdown<CheckpointAction>
        actionOrder={batchActions}
        danger={{ [checkpointAction.Delete]: true }}
        disabled={{
          [checkpointAction.Register]: !canActionCheckpoint(checkpointAction.Register, record),
          [checkpointAction.Delete]: !canActionCheckpoint(checkpointAction.Delete, record),
        }}
        id={record.uuid}
        kind="checkpoint"
        onError={handleError}
        onTrigger={dropDownOnTrigger(record.uuid)}
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

    const newColumns = [...defaultColumns].map((column) => {
      column.sortOrder = null;
      if (column.key === 'checkpoint') {
        column.render = checkpointRenderer;
      } else if (column.key === Checkpointv1SortBy.STATE) {
        column.filterDropdown = stateFilterDropdown;
        column.isFiltered = (settings) => !!(settings as Settings).state;
        column.filters = Object.values(CheckpointState)
          .filter((value) => value !== CheckpointState.Unspecified)
          .map((value) => ({
            text: <Badge state={value} type={BadgeType.State} />,
            value,
          }));
      } else if (column.key === Checkpointv1SortBy.SEARCHERMETRIC) {
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
  }, [dropDownOnTrigger, experiment, settings.sortDesc, settings.sortKey, stateFilterDropdown]);

  const stateString = settings.state?.join('.');
  const fetchExperimentCheckpoints = useCallback(async () => {
    if (!settings) return;
    try {
      const states = stateString
        ?.split('.')
        .map((state) => encodeCheckpointState(state as CheckpointState));
      const response = await getExperimentCheckpoints(
        {
          id: experiment.id,
          limit: settings.tableLimit,
          offset: settings.tableOffset,
          orderBy: settings.sortDesc ? 'ORDER_BY_DESC' : 'ORDER_BY_ASC',
          sortBy: validateDetApiEnum(Checkpointv1SortBy, settings.sortKey),
          states: validateDetApiEnumList(Checkpointv1State, states),
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
  }, [experiment.id, canceler, settings, stateString]);

  const submitBatchAction = useCallback(
    async (action: CheckpointAction) => {
      if (!settings.row) return;
      try {
        dropDownOnTrigger(settings.row)[action]();

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
    },
    [dropDownOnTrigger, fetchExperimentCheckpoints, settings.row],
  );

  const { stopPolling } = usePolling(fetchExperimentCheckpoints, { rerunOnNewFn: true });

  // Get new trials based on changes to the pagination, sorter and filters.
  useEffect(() => {
    setIsLoading(true);
    fetchExperimentCheckpoints();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    return () => {
      canceler.abort();
      stopPolling();
    };
  }, [canceler, stopPolling]);

  const handleTableRowSelect = useCallback(
    (rowKeys?: Key[]) => {
      updateSettings({ row: rowKeys?.map(String) });
    },
    [updateSettings],
  );

  const checkpointMap = useMemo(() => {
    return (checkpoints ?? []).reduce((acc, checkpoint) => {
      acc[checkpoint.uuid] = checkpoint;
      return acc;
    }, {} as Record<RecordKey, CoreApiGenericCheckpoint>);
  }, [checkpoints]);

  const availableBatchActions = useMemo(() => {
    const checkpoints = settings.row?.map((uuid) => checkpointMap[uuid]) ?? [];
    return getActionsForCheckpointsUnion(checkpoints, batchActions);
  }, [checkpointMap, settings.row]);

  return (
    <>
      <Section>
        <TableBatch
          actions={batchActions.map((action) => ({
            disabled: !availableBatchActions.includes(action),
            label: action,
            value: action,
          }))}
          selectedRowCount={(settings.row ?? []).length}
          onAction={(action) => submitBatchAction(action)}
          onClear={clearSelected}
        />
        {settings ? (
          <InteractiveTable<CoreApiGenericCheckpoint, Settings>
            columns={columns}
            containerRef={pageRef}
            ContextMenu={CheckpointActionDropdown}
            dataSource={checkpoints}
            loading={isLoading}
            pagination={getFullPaginationConfig(
              {
                limit: settings.tableLimit,
                offset: settings.tableOffset,
              },
              total,
            )}
            rowClassName={defaultRowClassName({ clickable: false })}
            rowKey="uuid"
            rowSelection={{
              onChange: handleTableRowSelect,
              preserveSelectedRowKeys: true,
              selectedRowKeys: settings.row ?? [],
            }}
            settings={settings}
            showSorterTooltip={false}
            size="small"
            updateSettings={updateSettings}
          />
        ) : (
          <SkeletonTable columns={columns.length} />
        )}
      </Section>
      <modelCreateModal.Component onClose={handleOnCloseCreateModel} />
      {modalCheckpointRegisterContextHolder}
    </>
  );
};

export default ExperimentCheckpoints;
