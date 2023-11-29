import { Space } from 'antd';
import Button from 'hew/Button';
import Column from 'hew/Column';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon, { IconName } from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import { useToast } from 'hew/Toast';
import Tooltip from 'hew/Tooltip';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import ExperimentTensorBoardModal from 'components/ExperimentTensorBoardModal';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import TableFilter from 'components/FilterForm/TableFilter';
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
import { TableViewMode } from './GlideTable';
import MultiSortMenu, { Sort } from './MultiSortMenu';
import { OptionsMenu, RowHeight } from './OptionsMenu';
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
  compareViewOn?: boolean;
  excludedExperimentIds?: Set<number>;
  experiments: Loadable<ExperimentWithTrial>[];
  filters: V1BulkExperimentFilters;
  formStore: FilterFormStore;
  heatmapBtnVisible: boolean;
  heatmapOn: boolean;
  initialVisibleColumns: string[];
  isOpenFilter: boolean;
  onActionComplete?: () => Promise<void>;
  onActionSuccess?: (action: BatchAction, successfulIds: number[]) => void;
  onComparisonViewToggle?: () => void;
  onHeatmapToggle?: (heatmapOn: boolean) => void;
  onIsOpenFilterChange?: (value: boolean) => void;
  onRowHeightChange?: (rowHeight: RowHeight) => void;
  onTableViewModeChange?: (mode: TableViewMode) => void;
  onSortChange?: (sorts: Sort[]) => void;
  onVisibleColumnChange?: (newColumns: string[]) => void;
  project: Project;
  projectColumns: Loadable<ProjectColumn[]>;
  rowHeight: RowHeight;
  selectAll: boolean;
  selectedExperimentIds: Set<number>;
  sorts: Sort[];
  tableViewMode: TableViewMode;
  total: Loadable<number>;
}

const TableActionBar: React.FC<Props> = ({
  compareViewOn,
  excludedExperimentIds,
  experiments,
  filters,
  formStore,
  heatmapBtnVisible,
  heatmapOn,
  initialVisibleColumns,
  isOpenFilter,
  onActionComplete,
  onActionSuccess,
  onComparisonViewToggle,
  onHeatmapToggle,
  onIsOpenFilterChange,
  onRowHeightChange,
  onSortChange,
  onTableViewModeChange,
  onVisibleColumnChange,
  project,
  projectColumns,
  rowHeight,
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
  const { Component: ExperimentTensorBoardModalComponent, open: openExperimentTensorBoardModal } =
    useModal(ExperimentTensorBoardModal);
  const isMobile = useMobile();
  const { openToast } = useToast();
  const experimentIds = useMemo(() => Array.from(selectedExperimentIds), [selectedExperimentIds]);

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
      Array.from(selectedExperimentIds).flatMap((id) =>
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
        params.filters = { ...filters, excludedExperimentIds: Array.from(excludedExperimentIds) };
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
            description: `No selected experiments were eligible for ${action.toLowerCase()}`,
            title: 'No eligible experiments',
          });
        } else if (numFailures === 0) {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${results.successful.length} experiments`,
            title: `${action} Success`,
          });
        } else if (numSuccesses === 0) {
          openToast({
            description: `Unable to ${action.toLowerCase()} ${numFailures} experiments`,
            severity: 'Warning',
            title: `${action} Failure`,
          });
        } else {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${numSuccesses} out of ${
              numFailures + numSuccesses
            } eligible
            experiments`,
            severity: 'Warning',
            title: `Partial ${action} Failure`,
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
          'experiment',
        )}`;

        if (selectAll) {
          const all = !excludedExperimentIds?.size ? 'All ' : '';
          const totalSelected =
            (totalExperiments - (excludedExperimentIds?.size ?? 0)).toLocaleString() + ' ';
          label = `${all}${totalSelected}experiments selected`;
        } else if (selectedExperimentIds.size > 0) {
          label = `${selectedExperimentIds.size} of ${label} selected`;
        }

        return label;
      },
      NotLoaded: () => 'Loading experiments...',
    });
  }, [excludedExperimentIds, selectAll, selectedExperimentIds, total]);

  const handleAction = useCallback((key: string) => handleBatchAction(key), [handleBatchAction]);

  return (
    <>
      <Row>
        <Column>
          <Space className={css.base}>
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
          </Space>
        </Column>
        <Column align="right">
          <Row>
            {heatmapBtnVisible && (
              <Tooltip content={'Toggle Metric Heatmap'}>
                <Button
                  icon={<Icon name="heatmap" title="heatmap" />}
                  type={heatmapOn ? 'primary' : 'default'}
                  onClick={() => onHeatmapToggle?.(heatmapOn)}
                />
              </Tooltip>
            )}
            {!!onComparisonViewToggle && (
              <Button
                hideChildren={isMobile}
                icon={<Icon name={compareViewOn ? 'panel-on' : 'panel'} title="compare" />}
                onClick={onComparisonViewToggle}>
                Compare
              </Button>
            )}
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
      <ExperimentTensorBoardModalComponent
        filters={selectAll ? filters : undefined}
        selectedExperiments={selectedExperiments}
        workspaceId={project?.workspaceId}
      />
    </>
  );
};

export default TableActionBar;
