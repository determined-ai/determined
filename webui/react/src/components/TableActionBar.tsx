import Button from 'hew/Button';
import Column from 'hew/Column';
import { Sort } from 'hew/DataGrid/DataGrid';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon, { IconName } from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import { useToast } from 'hew/Toast';
import Tooltip from 'hew/Tooltip';
import { Loadable, Loaded } from 'hew/utils/loadable';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import ColumnPickerMenu from 'components/ColumnPickerMenu';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import ExperimentRetainLogsModalComponent from 'components/ExperimentRetainLogsModal';
import ExperimentTensorBoardModal from 'components/ExperimentTensorBoardModal';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import TableFilter from 'components/FilterForm/TableFilter';
import MultiSortMenu from 'components/MultiSortMenu';
import { OptionsMenu, RowHeight } from 'components/OptionsMenu';
import { defaultProjectSettings } from 'components/Searches/Searches.settings';
import useMobile from 'hooks/useMobile';
import usePermissions from 'hooks/usePermissions';
import { defaultExperimentColumns } from 'pages/F_ExpList/expListColumns';
import {
  activateExperiments,
  archiveExperiments,
  cancelExperiments,
  deleteExperiments,
  getExperiments,
  killExperiments,
  openOrCreateTensorBoard,
  pauseExperiments,
  unarchiveExperiments,
} from 'services/api';
import { V1LocationType } from 'services/api-ts-sdk';
import {
  BulkActionResult,
  BulkExperimentItem,
  ExperimentAction,
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
import { capitalizeWord } from 'utils/string';
import { openCommandResponse } from 'utils/wait';

import LoadableCount from './LoadableCount';
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
  compareViewOn?: boolean;
  formStore: FilterFormStore;
  heatmapBtnVisible?: boolean;
  heatmapOn?: boolean;
  initialVisibleColumns: string[];
  isOpenFilter: boolean;
  onActionComplete?: () => Promise<void>;
  onActionSuccess?: (action: BatchAction, successfulIds: number[]) => void;
  onActualSelectAll?: () => void;
  onComparisonViewToggle?: () => void;
  onHeatmapToggle?: (heatmapOn: boolean) => void;
  onIsOpenFilterChange?: (value: boolean) => void;
  onRowHeightChange?: (rowHeight: RowHeight) => void;
  onSortChange?: (sorts: Sort[]) => void;
  onVisibleColumnChange?: (newColumns: string[], pinnedCount?: number) => void;
  onHeatmapSelectionRemove?: (id: string) => void;
  pageSize?: number;
  project: Project;
  projectColumns: Loadable<ProjectColumn[]>;
  rowHeight: RowHeight;
  selectedExperimentIds: number[];
  sorts: Sort[];
  pinnedColumnsCount?: number;
  total: Loadable<number>;
  labelSingular: string;
  labelPlural: string;
  columnGroups: (V1LocationType | V1LocationType[])[];
  bannedFilterColumns?: Set<string>;
  bannedSortColumns?: Set<string>;
  entityCopy?: string;
}

const TableActionBar: React.FC<Props> = ({
  compareViewOn,
  formStore,
  heatmapBtnVisible,
  heatmapOn,
  initialVisibleColumns,
  isOpenFilter,
  onActionComplete,
  onActionSuccess,
  onActualSelectAll,
  onComparisonViewToggle,
  onHeatmapToggle,
  onIsOpenFilterChange,
  onRowHeightChange,
  onSortChange,
  onHeatmapSelectionRemove,
  onVisibleColumnChange,
  pageSize,
  project,
  projectColumns,
  rowHeight,
  selectedExperimentIds,
  sorts,
  pinnedColumnsCount = 0,
  total,
  labelSingular,
  labelPlural,
  columnGroups,
  bannedFilterColumns,
  bannedSortColumns,
  entityCopy,
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

  const [experiments, setExperiments] = useState<Loadable<BulkExperimentItem>[]>([]);

  const fetchExperimentsByIds = useCallback(async (selectedIds: number[]) => {
    const CHUNK_SIZE = 80; // match largest pageSizeOption used for Pagination components
    const chunkedExperimentIds = _.chunk(selectedIds, CHUNK_SIZE);
    try {
      const responses = await Promise.all(
        chunkedExperimentIds.map(async (ids) => {
          const response = await getExperiments({
            experimentIdFilter: {
              incl: ids,
            },
          });
          return response.experiments;
        }),
      );
      const fetchedExperiments = responses.reduce((acc, experiments) => {
        acc.push(...experiments);
        return acc;
      }, []);
      setExperiments(fetchedExperiments.map((experiment) => Loaded(experiment)));
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch experiments for selected experiment ids.' });
    }
  }, []);

  useEffect(() => {
    fetchExperimentsByIds(selectedExperimentIds);
  }, [selectedExperimentIds, fetchExperimentsByIds]);

  const experimentMap = useMemo(() => {
    return experiments.filter(Loadable.isLoaded).reduce(
      (acc, experiment) => {
        acc[experiment.data.id] = getProjectExperimentForExperimentItem(experiment.data, project);
        return acc;
      },
      {} as Record<number, ProjectExperiment>,
    );
  }, [experiments, project]);

  const selectedExperiments = useMemo(
    () => selectedExperimentIds.flatMap((id) => (id in experimentMap ? [experimentMap[id]] : [])),
    [experimentMap, selectedExperimentIds],
  );

  const availableBatchActions = useMemo(() => {
    const experiments = selectedExperimentIds.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, [...batchActions], permissions);
    // Spreading batchActions is so TypeScript doesn't complain that it's readonly.
  }, [selectedExperimentIds, experimentMap, permissions]);

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionResult | void> => {
      const validExperimentIds = selectedExperiments
        .filter((exp) => !exp.unmanaged && canActionExperiment(action, exp))
        .map((exp) => exp.id);
      const params = {
        experimentIds: validExperimentIds,
        projectId: project.id,
      };
      switch (action) {
        case ExperimentAction.OpenTensorBoard: {
          if (validExperimentIds.length !== selectedExperiments.length) {
            // if unmanaged experiments are selected, open experimentTensorBoardModal
            openExperimentTensorBoardModal();
          } else {
            openCommandResponse(
              await openOrCreateTensorBoard({
                experimentIds: params.experimentIds,
                workspaceId: project?.workspaceId,
              }),
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
      ExperimentMoveModal,
      ExperimentRetainLogsModal,
      openExperimentTensorBoardModal,
      project,
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
            description: `No selected ${labelPlural.toLowerCase()} were eligible for ${action.toLowerCase()}`,
            title: `No eligible ${labelPlural.toLowerCase()}`,
          });
        } else if (numFailures === 0) {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${
              results.successful.length
            } ${labelPlural.toLowerCase()}`,
            title: `${action} Success`,
          });
        } else if (numSuccesses === 0) {
          openToast({
            description: `Unable to ${action.toLowerCase()} ${numFailures} ${labelPlural.toLowerCase()}`,
            severity: 'Warning',
            title: `${action} Failure`,
          });
        } else {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${numSuccesses} out of ${
              numFailures + numSuccesses
            } eligible
            ${labelPlural.toLowerCase()}`,
            severity: 'Warning',
            title: `Partial ${action} Failure`,
          });
        }
      } catch (e) {
        const publicSubject =
          action === ExperimentAction.OpenTensorBoard
            ? `Unable to View TensorBoard for Selected ${capitalizeWord(labelPlural)}`
            : `Unable to ${action} Selected ${capitalizeWord(labelPlural)}`;
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
    [sendBatchActions, onActionComplete, onActionSuccess, openToast, labelPlural],
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

  const handleAction = useCallback((key: string) => handleBatchAction(key), [handleBatchAction]);

  return (
    <div className={css.base} data-test-component="tableActionBar">
      <Row>
        <Column>
          <Row>
            <TableFilter
              bannedFilterColumns={bannedFilterColumns}
              entityCopy={entityCopy}
              formStore={formStore}
              isMobile={isMobile}
              isOpenFilter={isOpenFilter}
              loadableColumns={projectColumns}
              onIsOpenFilterChange={onIsOpenFilterChange}
            />
            <MultiSortMenu
              bannedSortColumns={bannedSortColumns}
              columns={projectColumns}
              isMobile={isMobile}
              sorts={sorts}
              onChange={onSortChange}
            />
            <ColumnPickerMenu
              compare={compareViewOn}
              defaultPinnedCount={defaultProjectSettings.pinnedColumnsCount}
              defaultVisibleColumns={defaultExperimentColumns}
              initialVisibleColumns={initialVisibleColumns}
              isMobile={isMobile}
              pinnedColumnsCount={pinnedColumnsCount}
              projectColumns={projectColumns}
              projectId={project.id}
              tabs={columnGroups}
              onHeatmapSelectionRemove={onHeatmapSelectionRemove}
              onVisibleColumnChange={onVisibleColumnChange}
            />
            <OptionsMenu rowHeight={rowHeight} onRowHeightChange={onRowHeightChange} />
            {selectedExperimentIds.length > 0 && (
              <Dropdown menu={editMenuItems} onClick={handleAction}>
                <Button data-test="actionsDropdown" hideChildren={isMobile}>
                  Actions
                </Button>
              </Dropdown>
            )}
            <LoadableCount
              labelPlural={labelPlural}
              labelSingular={labelSingular}
              pageSize={pageSize}
              selectedCount={selectedExperimentIds.length}
              total={total}
              onActualSelectAll={onActualSelectAll}
            />
          </Row>
        </Column>
        <Column align="right">
          <Row>
            {heatmapBtnVisible && (
              <Tooltip content={'Toggle Metric Heatmap'}>
                <Button
                  data-test="heatmapToggle"
                  icon={<Icon name="heatmap" title="heatmap" />}
                  type={heatmapOn ? 'primary' : 'default'}
                  onClick={() => onHeatmapToggle?.(heatmapOn ?? false)}
                />
              </Tooltip>
            )}
            {!!onComparisonViewToggle && (
              <Button
                data-test="compare"
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
        experimentIds={selectedExperimentIds.filter(
          (id) =>
            canActionExperiment(ExperimentAction.Move, experimentMap[id]) &&
            permissions.canMoveExperiment({ experiment: experimentMap[id] }),
        )}
        sourceProjectId={project.id}
        sourceWorkspaceId={project.workspaceId}
        onSubmit={handleSubmitMove}
      />
      <ExperimentRetainLogsModal.Component
        experimentIds={selectedExperimentIds.filter(
          (id) =>
            canActionExperiment(ExperimentAction.RetainLogs, experimentMap[id]) &&
            permissions.canModifyExperiment({
              workspace: { id: experimentMap[id].workspaceId },
            }),
        )}
        projectId={project.id}
        onSubmit={handleSubmitRetainLogs}
      />
      <ExperimentTensorBoardModalComponent
        selectedExperiments={selectedExperiments}
        workspaceId={project?.workspaceId}
      />
    </div>
  );
};

export default TableActionBar;
