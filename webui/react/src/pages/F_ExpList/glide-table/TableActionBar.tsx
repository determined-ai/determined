import { Space } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import TableFilter from 'components/FilterForm/TableFilter';
import Button from 'components/kit/Button';
import { Column, Columns } from 'components/kit/Columns';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon, { IconName } from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import useMobile from 'hooks/useMobile';
import usePermissions from 'hooks/usePermissions';
import {
  activateExperiments,
  archiveExperiments,
  cancelExperiments,
  deleteExperiments,
  killExperiments,
  openOrCreateTensorBoard,
  pauseExperiments,
  unarchiveExperiments,
} from 'services/api';
import { V1BulkExperimentFilters } from 'services/api-ts-sdk';
import { RecordKey } from 'types';
import {
  BulkActionResult,
  ExperimentAction,
  ExperimentWithTrial,
  Project,
  ProjectColumn,
  ProjectExperiment,
} from 'types';
import { notification } from 'utils/dialogApi';
import { ErrorLevel } from 'utils/error';
import handleError from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperimentsUnion,
  getProjectExperimentForExperimentItem,
} from 'utils/experiment';
import { Loadable } from 'utils/loadable';
import { openCommandResponse } from 'utils/wait';

import { ExpListView, RowHeight } from '../F_ExperimentList.settings';

import ColumnPickerMenu from './ColumnPickerMenu';
import MultiSortMenu, { Sort } from './MultiSortMenu';
import { OptionsMenu } from './OptionsMenu';
import css from './TableActionBar.module.scss';

const batchActions = [
  ExperimentAction.OpenTensorBoard,
  ExperimentAction.Move,
  ExperimentAction.Archive,
  ExperimentAction.Unarchive,
  ExperimentAction.Delete,
  ExperimentAction.Activate,
  ExperimentAction.Pause,
  ExperimentAction.Cancel,
  ExperimentAction.Kill,
] as const;

export type BatchAction = (typeof batchActions)[number];

const actionIcons: Record<BatchAction, IconName> = {
  [ExperimentAction.Activate]: 'play',
  [ExperimentAction.Pause]: 'pause',
  [ExperimentAction.Cancel]: 'stop',
  [ExperimentAction.Archive]: 'archive',
  [ExperimentAction.Unarchive]: 'document',
  [ExperimentAction.Move]: 'workspaces',
  [ExperimentAction.OpenTensorBoard]: 'tensor-board',
  [ExperimentAction.Kill]: 'cancelled',
  [ExperimentAction.Delete]: 'error',
} as const;

interface Props {
  experiments: Loadable<ExperimentWithTrial>[];
  filters: V1BulkExperimentFilters;
  initialVisibleColumns: string[];
  onAction: () => Promise<void>;
  sorts: Sort[];
  onSortChange: (sorts: Sort[]) => void;
  project: Project;
  projectColumns: Loadable<ProjectColumn[]>;
  selectAll: boolean;
  excludedExperimentIds?: Set<number>;
  selectedExperimentIds: number[];
  handleUpdateExperimentList: (action: BatchAction, successfulIds: number[]) => void;
  setVisibleColumns: (newColumns: string[]) => void;
  toggleComparisonView?: () => void;
  compareViewOn?: boolean;
  total: Loadable<number>;
  formStore: FilterFormStore;
  setIsOpenFilter: (value: boolean) => void;
  isOpenFilter: boolean;
  expListView: ExpListView;
  setExpListView: (view: ExpListView) => void;
  rowHeight: RowHeight;
  onRowHeightChange: (r: RowHeight) => void;
}

const TableActionBar: React.FC<Props> = ({
  experiments,
  excludedExperimentIds,
  filters,
  onAction,
  onSortChange,
  selectAll,
  selectedExperimentIds,
  handleUpdateExperimentList,
  sorts,
  project,
  projectColumns,
  total,
  initialVisibleColumns,
  setVisibleColumns,
  formStore,
  setIsOpenFilter,
  isOpenFilter,
  expListView,
  setExpListView,
  toggleComparisonView,
  rowHeight,
  onRowHeightChange,
  compareViewOn,
}) => {
  const permissions = usePermissions();
  const [batchAction, setBatchAction] = useState<BatchAction>();
  const BatchActionConfirmModal = useModal(BatchActionConfirmModalComponent);
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);
  const totalExperiments = Loadable.getOrElse(0, total);
  const isMobile = useMobile();

  const experimentMap = useMemo(() => {
    return experiments.filter(Loadable.isLoaded).reduce((acc, experiment) => {
      acc[experiment.data.experiment.id] = getProjectExperimentForExperimentItem(
        experiment.data.experiment,
        project,
      );
      return acc;
    }, {} as Record<RecordKey, ProjectExperiment>);
  }, [experiments, project]);

  const availableBatchActions = useMemo(() => {
    if (selectAll)
      return batchActions.filter((action) => action !== ExperimentAction.OpenTensorBoard);
    const experiments = selectedExperimentIds.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, [...batchActions], permissions);
    // Spreading batchActions is so TypeScript doesn't complain that it's readonly.
  }, [experimentMap, permissions, selectAll, selectedExperimentIds]);

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionResult | void> => {
      let requestFilters = selectAll ? filters : undefined;
      if (excludedExperimentIds?.size) {
        requestFilters = { ...filters, excludedExperimentIds: Array.from(excludedExperimentIds) };
      }
      switch (action) {
        case ExperimentAction.OpenTensorBoard:
          return openCommandResponse(
            await openOrCreateTensorBoard({
              experimentIds: selectedExperimentIds,
              filters: requestFilters,
              workspaceId: project?.workspaceId,
            }),
          );
        case ExperimentAction.Move:
          return ExperimentMoveModal.open();
        case ExperimentAction.Activate:
          return await activateExperiments({
            experimentIds: selectedExperimentIds,
            filters: requestFilters,
          });
        case ExperimentAction.Archive:
          return await archiveExperiments({
            experimentIds: selectedExperimentIds,
            filters: requestFilters,
          });
        case ExperimentAction.Cancel:
          return await cancelExperiments({
            experimentIds: selectedExperimentIds,
            filters: requestFilters,
          });
        case ExperimentAction.Kill:
          return await killExperiments({
            experimentIds: selectedExperimentIds,
            filters: requestFilters,
          });
        case ExperimentAction.Pause:
          return await pauseExperiments({
            experimentIds: selectedExperimentIds,
            filters: requestFilters,
          });
        case ExperimentAction.Unarchive:
          return await unarchiveExperiments({
            experimentIds: selectedExperimentIds,
            filters: requestFilters,
          });
        case ExperimentAction.Delete:
          return await deleteExperiments({
            experimentIds: selectedExperimentIds,
            filters: requestFilters,
          });
      }
    },
    [
      selectedExperimentIds,
      selectAll,
      excludedExperimentIds,
      filters,
      project?.workspaceId,
      ExperimentMoveModal,
    ],
  );

  const handleSubmitMove = useCallback(
    async (successfulIds?: number[]) => {
      if (!successfulIds) return;
      handleUpdateExperimentList(ExperimentAction.Move, successfulIds);
      await onAction();
    },
    [handleUpdateExperimentList, onAction],
  );

  const closeNotification = useCallback(() => notification.destroy(), []);

  const submitBatchAction = useCallback(
    async (action: BatchAction) => {
      try {
        const results = await sendBatchActions(action);
        if (results === undefined) return;

        handleUpdateExperimentList(action, results.successful);

        const numSuccesses = results.successful.length;
        const numFailures = results.failed.length;

        if (numSuccesses === 0 && numFailures === 0) {
          notification.open({
            description: `No selected experiments were eligible for ${action.toLowerCase()}`,
            message: 'No eligible experiments',
          });
        } else if (numFailures === 0) {
          notification.open({
            btn: null,
            description: (
              <div onClick={closeNotification}>
                <p>
                  {action} succeeded for {results.successful.length} experiments
                </p>
              </div>
            ),
            message: `${action} Success`,
          });
        } else if (numSuccesses === 0) {
          notification.warning({
            description: `Unable to ${action.toLowerCase()} ${numFailures} experiments`,
            message: `${action} Failure`,
          });
        } else {
          notification.warning({
            description: (
              <div onClick={closeNotification}>
                <p>
                  {action} succeeded for {numSuccesses} out of {numFailures + numSuccesses} eligible
                  experiments
                </p>
              </div>
            ),
            key: 'move-notification',
            message: `Partial ${action} Failure`,
          });
        }
      } catch (e) {
        const publicSubject =
          action === ExperimentAction.OpenTensorBoard
            ? 'Unable to View TensorBoard for Selected Experiments'
            : `Unable to ${action} Selected Experiments`;
        handleError(e, {
          isUserTriggered: true,
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
        });
      } finally {
        onAction();
      }
    },
    [sendBatchActions, closeNotification, onAction, handleUpdateExperimentList],
  );

  const handleBatchAction = useCallback(
    (action: string) => {
      if (action === ExperimentAction.OpenTensorBoard) {
        submitBatchAction(action);
      } else if (action === ExperimentAction.Move) {
        sendBatchActions(action);
      } else {
        setBatchAction(action as BatchAction);
        BatchActionConfirmModal.open();
      }
    },
    [BatchActionConfirmModal, submitBatchAction, sendBatchActions],
  );

  const editMenuItems: MenuItem[] = useMemo(() => {
    return batchActions.map((action) => ({
      danger: action === ExperimentAction.Delete,
      disabled: !availableBatchActions.includes(action),
      // The icon doesn't show up without being wrapped in a div.
      icon: (
        <div>
          <Icon name={actionIcons[action]} title={action} />
        </div>
      ),
      key: action,
      label: action,
    }));
  }, [availableBatchActions]);

  const selectionLabel = useMemo(() => {
    let label = `${totalExperiments.toLocaleString()} experiment${totalExperiments > 1 && 's'}`;

    if (selectAll) {
      const totalSelected = Loadable.isLoaded(total)
        ? (total.data - (excludedExperimentIds?.size ?? 0)).toLocaleString() + ' '
        : '';
      label = `All ${totalSelected}experiments selected`;
    } else if (selectedExperimentIds.length > 0) {
      label = `${selectedExperimentIds.length} of ${label} selected`;
    }

    return label;
  }, [excludedExperimentIds, selectAll, selectedExperimentIds, total, totalExperiments]);

  const handleAction = useCallback((key: string) => handleBatchAction(key), [handleBatchAction]);

  return (
    <Columns>
      <Column>
        <Space className={css.base}>
          <TableFilter
            formStore={formStore}
            isMobile={isMobile}
            isOpenFilter={isOpenFilter}
            loadableColumns={projectColumns}
            setIsOpenFilter={setIsOpenFilter}
          />
          <MultiSortMenu
            columns={projectColumns}
            isMobile={isMobile}
            sorts={sorts}
            onChange={onSortChange}
          />
          <ColumnPickerMenu
            initialVisibleColumns={initialVisibleColumns}
            isMobile={isMobile}
            projectColumns={projectColumns}
            projectId={project.id}
            setVisibleColumns={setVisibleColumns}
          />
          {(selectAll || selectedExperimentIds.length > 0) && (
            <Dropdown menu={editMenuItems} onClick={handleAction}>
              <Button hideChildren={isMobile} icon={<Icon decorative name="pencil" />}>
                Edit (
                {selectAll
                  ? Loadable.isLoaded(total)
                    ? (total.data - (excludedExperimentIds?.size ?? 0)).toLocaleString()
                    : 'All'
                  : selectedExperimentIds.length}
                )
              </Button>
            </Dropdown>
          )}
          {!isMobile && <span className={css.expNum}>{selectionLabel}</span>}
        </Space>
      </Column>
      <Column align="right">
        <Columns>
          <OptionsMenu
            expListView={expListView}
            rowHeight={rowHeight}
            setExpListView={setExpListView}
            onRowHeightChange={onRowHeightChange}
          />
          {!!toggleComparisonView && (
            <Button
              hideChildren={isMobile}
              icon={<Icon name={compareViewOn ? 'panel-on' : 'panel'} title="compare" />}
              onClick={toggleComparisonView}>
              Compare
            </Button>
          )}
        </Columns>
      </Column>
      {batchAction && (
        <BatchActionConfirmModal.Component
          batchAction={batchAction}
          onConfirm={() => submitBatchAction(batchAction)}
        />
      )}
      <ExperimentMoveModal.Component
        excludedExperimentIds={excludedExperimentIds}
        experimentIds={selectedExperimentIds.filter(
          (id) =>
            canActionExperiment(ExperimentAction.Move, experimentMap[id]) &&
            permissions.canMoveExperiment({ experiment: experimentMap[id] }),
        )}
        filters={selectAll ? filters : undefined}
        sourceProjectId={project.id}
        sourceWorkspaceId={project.workspaceId}
        onSubmit={handleSubmitMove}
      />
    </Columns>
  );
};

export default TableActionBar;
