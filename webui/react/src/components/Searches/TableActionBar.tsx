import Button from 'hew/Button';
import Column from 'hew/Column';
import { Sort } from 'hew/DataGrid/DataGrid';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon, { IconName } from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import { useToast } from 'hew/Toast';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import ExperimentRetainLogsModalComponent from 'components/ExperimentRetainLogsModal';
import ExperimentTensorBoardModal from 'components/ExperimentTensorBoardModal';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import TableFilter from 'components/FilterForm/TableFilter';
import MultiSortMenu from 'components/MultiSortMenu';
import { OptionsMenu, RowHeight, TableViewMode } from 'components/OptionsMenu';
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
import {
  BulkActionResult,
  ExperimentAction,
  ExperimentWithTrial,
  Project,
  ProjectColumn,
  ProjectExperiment,
} from 'types';
import handleError, { ErrorLevel } from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperimentsUnion,
  getProjectExperimentForExperimentItem,
} from 'utils/experiment';
import { pluralizer } from 'utils/string';
import { openCommandResponse } from 'utils/wait';

import ColumnPickerMenu from './ColumnPickerMenu';
import { SelectionType } from './Searches.settings';
import css from './TableActionBar.module.scss';

const batchActions = [
  ExperimentAction.OpenTensorBoard,
  ExperimentAction.Move,
  ExperimentAction.RetainLogs,
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
  [ExperimentAction.RetainLogs]: 'logs',
  [ExperimentAction.OpenTensorBoard]: 'tensor-board',
  [ExperimentAction.Kill]: 'cancelled',
  [ExperimentAction.Delete]: 'error',
} as const;

interface Props {
  excludedExperimentIds?: Map<number, unknown>;
  experiments: Loadable<ExperimentWithTrial>[];
  filters: V1BulkExperimentFilters;
  formStore: FilterFormStore;
  initialVisibleColumns: string[];
  isOpenFilter: boolean;
  onActionComplete?: () => Promise<void>;
  onActionSuccess?: (action: BatchAction, successfulIds: number[]) => void;
  onIsOpenFilterChange?: (value: boolean) => void;
  onRowHeightChange?: (rowHeight: RowHeight) => void;
  onTableViewModeChange?: (mode: TableViewMode) => void;
  onSortChange?: (sorts: Sort[]) => void;
  onVisibleColumnChange?: (newColumns: string[]) => void;
  project: Project;
  projectColumns: Loadable<ProjectColumn[]>;
  rowHeight: RowHeight;
  selectAll: boolean;
  selectedExperimentIds: Map<number, unknown>;
  selection: SelectionType;
  sorts: Sort[];
  tableViewMode: TableViewMode;
  total: Loadable<number>;
}

const TableActionBar: React.FC<Props> = ({
  excludedExperimentIds,
  experiments,
  filters,
  formStore,
  initialVisibleColumns,
  isOpenFilter,
  onActionComplete,
  onActionSuccess,
  onIsOpenFilterChange,
  onRowHeightChange,
  onSortChange,
  onTableViewModeChange,
  onVisibleColumnChange,
  project,
  projectColumns,
  rowHeight,
  selection,
  selectAll,
  selectedExperimentIds,
  sorts,
  tableViewMode,
  total,
}) => {
  const permissions = usePermissions();
  const [batchAction, setBatchAction] = useState<BatchAction>();
  const BatchActionConfirmModal = useModal(BatchActionConfirmModalComponent);
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);
  const ExperimentRetainLogsModal = useModal(ExperimentRetainLogsModalComponent);
  const { Component: ExperimentTensorBoardModalComponent, open: openExperimentTensorBoardModal } =
    useModal(ExperimentTensorBoardModal);
  const isMobile = useMobile();
  const { openToast } = useToast();
  const experimentIds = useMemo(
    () => Array.from(selectedExperimentIds.keys()),
    [selectedExperimentIds],
  );

  const experimentMap = useMemo(() => {
    return experiments.filter(Loadable.isLoaded).reduce(
      (acc, experiment) => {
        acc[experiment.data.experiment.id] = getProjectExperimentForExperimentItem(
          experiment.data.experiment,
          project,
        );
        return acc;
      },
      {} as Record<number, ProjectExperiment>,
    );
  }, [experiments, project]);

  const selectedExperiments = useMemo(
    () =>
      Array.from(selectedExperimentIds.keys()).flatMap((id) =>
        id in experimentMap ? [experimentMap[id]] : [],
      ),
    [experimentMap, selectedExperimentIds],
  );

  const availableBatchActions = useMemo(() => {
    if (selectAll)
      return batchActions.filter((action) => action !== ExperimentAction.OpenTensorBoard);
    const experiments = experimentIds.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, [...batchActions], permissions);
    // Spreading batchActions is so TypeScript doesn't complain that it's readonly.
  }, [experimentIds, experimentMap, permissions, selectAll]);

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionResult | void> => {
      const managedExperimentIds = selectedExperiments
        .filter((exp) => !exp.unmanaged)
        .map((exp) => exp.id);
      const params = {
        experimentIds: managedExperimentIds,
        filters: selectAll ? filters : undefined,
      };
      if (excludedExperimentIds?.size) {
        params.filters = {
          ...filters,
          excludedExperimentIds: Array.from(excludedExperimentIds.keys()),
        };
      }
      switch (action) {
        case ExperimentAction.OpenTensorBoard: {
          if (managedExperimentIds.length !== selectedExperiments.length) {
            // if unmanaged experiments are selected, open experimentTensorBoardModal
            openExperimentTensorBoardModal();
          } else {
            openCommandResponse(
              await openOrCreateTensorBoard({ ...params, workspaceId: project?.workspaceId }),
            );
          }
          return;
        }
        case ExperimentAction.Move:
          return ExperimentMoveModal.open();
        case ExperimentAction.RetainLogs:
          return ExperimentRetainLogsModal.open();
        case ExperimentAction.Activate:
          return await activateExperiments(params);
        case ExperimentAction.Archive:
          return await archiveExperiments(params);
        case ExperimentAction.Cancel:
          return await cancelExperiments(params);
        case ExperimentAction.Kill:
          return await killExperiments(params);
        case ExperimentAction.Pause:
          return await pauseExperiments(params);
        case ExperimentAction.Unarchive:
          return await unarchiveExperiments(params);
        case ExperimentAction.Delete:
          return await deleteExperiments(params);
      }
    },
    [
      selectedExperiments,
      selectAll,
      filters,
      excludedExperimentIds,
      ExperimentMoveModal,
      ExperimentRetainLogsModal,
      openExperimentTensorBoardModal,
      project?.workspaceId,
    ],
  );

  const handleSubmitMove = useCallback(
    async (successfulIds?: number[]) => {
      if (!successfulIds) return;
      onActionSuccess?.(ExperimentAction.Move, successfulIds);
      await onActionComplete?.();
    },
    [onActionComplete, onActionSuccess],
  );

  const handleSubmitRetainLogs = useCallback(
    async (successfulIds?: number[]) => {
      if (!successfulIds) return;
      onActionSuccess?.(ExperimentAction.RetainLogs, successfulIds);
      await onActionComplete?.();
    },
    [onActionComplete, onActionSuccess],
  );

  const submitBatchAction = useCallback(
    async (action: BatchAction) => {
      try {
        const results = await sendBatchActions(action);
        if (results === undefined) return;

        onActionSuccess?.(action, results.successful);

        const numSuccesses = results.successful.length;
        const numFailures = results.failed.length;

        if (numSuccesses === 0 && numFailures === 0) {
          openToast({
            description: `No selected searches were eligible for ${action.toLowerCase()}`,
            title: 'No eligible searches',
          });
        } else if (numFailures === 0) {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${results.successful.length} searches`,
            title: `${action} Success`,
          });
        } else if (numSuccesses === 0) {
          openToast({
            description: `Unable to ${action.toLowerCase()} ${numFailures} searches`,
            severity: 'Warning',
            title: `${action} Failure`,
          });
        } else {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${numSuccesses} out of ${numFailures + numSuccesses
              } eligible
            searches`,
            severity: 'Warning',
            title: `Partial ${action} Failure`,
          });
        }
      } catch (e) {
        const publicSubject =
          action === ExperimentAction.OpenTensorBoard
            ? 'Unable to View TensorBoard for Selected Searches'
            : `Unable to ${action} Selected Searches`;
        handleError(e, {
          isUserTriggered: true,
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
        });
      } finally {
        onActionComplete?.();
      }
    },
    [sendBatchActions, onActionComplete, onActionSuccess, openToast],
  );

  const handleBatchAction = useCallback(
    (action: string) => {
      if (action === ExperimentAction.OpenTensorBoard) {
        submitBatchAction(action);
      } else if (action === ExperimentAction.Move) {
        sendBatchActions(action);
      } else if (action === ExperimentAction.RetainLogs) {
        sendBatchActions(action);
      } else {
        setBatchAction(action as BatchAction);
        BatchActionConfirmModal.open();
      }
    },
    [BatchActionConfirmModal, submitBatchAction, sendBatchActions],
  );

  const editMenuItems = useMemo(() => {
    const groupedBatchActions = [
      batchActions.slice(0, 1), // View in TensorBoard
      batchActions.slice(1, 5), // Move, Archive, Unarchive, Delete
      batchActions.slice(5), // Resume, Pause, Cancel, Kill
    ];
    const groupSize = groupedBatchActions.length;
    return groupedBatchActions.reduce((acc, group, index) => {
      const isLastGroup = index === groupSize - 1;
      group.forEach((action) =>
        acc.push({
          danger: action === ExperimentAction.Delete,
          disabled: !availableBatchActions.includes(action),
          icon: <Icon name={actionIcons[action]} title={action} />,
          key: action,
          label: action,
        }),
      );
      if (!isLastGroup) acc.push({ type: 'divider' });
      return acc;
    }, [] as MenuItem[]);
  }, [availableBatchActions]);

  const selectionLabel = useMemo(() => {
    return Loadable.match(total, {
      Failed: () => null,
      Loaded: (totalExperiments) => {
        let label = `${totalExperiments.toLocaleString()} ${pluralizer(
          totalExperiments,
          'search',
        )}`;

        if (selection.type === 'ALL_EXCEPT') {
          const all = selection.exclusions.length === 0 ? 'All ' : '';
          const totalSelected =
            (totalExperiments - (selection.exclusions.length ?? 0)).toLocaleString() + ' ';
          label = `${all}${totalSelected}searches selected`;
        } else if (selection.selections.length > 0) {
          label = `${selection.selections.length} of ${label} selected`;
        }

        return label;
      },
      NotLoaded: () => 'Loading searches...',
    });
  }, [selection, total]);

  const handleAction = useCallback((key: string) => handleBatchAction(key), [handleBatchAction]);

  return (
    <div className={css.base}>
      <Row>
        <Column>
          <Row>
            <TableFilter
              formStore={formStore}
              isMobile={isMobile}
              isOpenFilter={isOpenFilter}
              loadableColumns={projectColumns}
              onIsOpenFilterChange={onIsOpenFilterChange}
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
              onVisibleColumnChange={onVisibleColumnChange}
            />
            <OptionsMenu
              rowHeight={rowHeight}
              tableViewMode={tableViewMode}
              onRowHeightChange={onRowHeightChange}
              onTableViewModeChange={onTableViewModeChange}
            />
            {(selectAll || selectedExperimentIds.size > 0) && (
              <Dropdown menu={editMenuItems} onClick={handleAction}>
                <Button hideChildren={isMobile}>Actions</Button>
              </Dropdown>
            )}
            {!isMobile && <span className={css.expNum}>{selectionLabel}</span>}
          </Row>
        </Column>
      </Row>
      {batchAction && (
        <BatchActionConfirmModal.Component
          batchAction={batchAction}
          isUnmanagedIncluded={selectedExperiments.some((exp) => exp.unmanaged)}
          onConfirm={() => submitBatchAction(batchAction)}
        />
      )}
      <ExperimentMoveModal.Component
        excludedExperimentIds={excludedExperimentIds}
        experimentIds={experimentIds.filter(
          (id) =>
            canActionExperiment(ExperimentAction.Move, experimentMap[id]) &&
            permissions.canMoveExperiment({ experiment: experimentMap[id] }),
        )}
        filters={selectAll ? filters : undefined}
        sourceProjectId={project.id}
        sourceWorkspaceId={project.workspaceId}
        onSubmit={handleSubmitMove}
      />
      <ExperimentRetainLogsModal.Component
        excludedExperimentIds={excludedExperimentIds}
        experimentIds={experimentIds.filter(
          (id) =>
            canActionExperiment(ExperimentAction.RetainLogs, experimentMap[id]) &&
            permissions.canModifyExperiment({
              workspace: { id: experimentMap[id].workspaceId },
            }),
        )}
        filters={selectAll ? filters : undefined}
        onSubmit={handleSubmitRetainLogs}
      />
      <ExperimentTensorBoardModalComponent
        filters={selectAll ? filters : undefined}
        selectedExperiments={selectedExperiments}
        workspaceId={project?.workspaceId}
      />
    </div>
  );
};

export default TableActionBar;
