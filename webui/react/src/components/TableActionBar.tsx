import Button from 'hew/Button';
import Column from 'hew/Column';
import { Sort } from 'hew/DataGrid/DataGrid';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon, { IconName } from 'hew/Icon';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import { useToast } from 'hew/Toast';
import Tooltip from 'hew/Tooltip';
import { Loadable, NotLoaded } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { useCallback, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import ColumnPickerMenu from 'components/ColumnPickerMenu';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import ExperimentRetainLogsModalComponent from 'components/ExperimentRetainLogsModal';
import ExperimentTensorBoardModal from 'components/ExperimentTensorBoardModal';
import { FilterFormStore } from 'components/FilterForm/components/FilterFormStore';
import {
  Conjunction,
  FilterFormSetWithoutId,
  FormKind,
  Operator,
} from 'components/FilterForm/components/type';
import TableFilter from 'components/FilterForm/TableFilter';
import MultiSortMenu from 'components/MultiSortMenu';
import { OptionsMenu, RowHeight, TableViewMode } from 'components/OptionsMenu';
import useMobile from 'hooks/useMobile';
import usePermissions from 'hooks/usePermissions';
import { SelectionType } from 'pages/F_ExpList/F_ExperimentList.settings';
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
import { V1ColumnType, V1LocationType } from 'services/api-ts-sdk';
import { BulkActionParams } from 'services/types';
import {
  BulkActionResult,
  ExperimentAction,
  BulkExperimentItem,
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
import { capitalizeWord, pluralizer } from 'utils/string';
import { openCommandResponse } from 'utils/wait';

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
  selectedExperimentsMap: Map<number, { experiment: BulkExperimentItem }>;
  selection: SelectionType;
  sorts: Sort[];
  tableViewMode: TableViewMode;
  total: Loadable<number>;
  labelSingular: string;
  labelPlural: string;
  columnGroups: (V1LocationType | V1LocationType[])[];
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
  selection,
  selectedExperimentsMap,
  sorts,
  tableViewMode,
  total,
  labelSingular,
  labelPlural,
  columnGroups,
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
  const validFilterSet = useObservable(formStore.validFilterSet);

  const selectedProjectExperimentMap = useMemo(() => {
    return [...selectedExperimentsMap].reduce(
      (acc, [id, { experiment }]) => {
        acc[id] = getProjectExperimentForExperimentItem(experiment, project);
        return acc;
      },
      {} as Record<number, ProjectExperiment>,
    );
  }, [selectedExperimentsMap, project]);

  const selectedExperiments = useMemo(
    () => Object.values(selectedProjectExperimentMap),
    [selectedProjectExperimentMap],
  );

  const availableBatchActions = useMemo(() => {
    if (selection.type === 'ALL_EXCEPT')
      return batchActions.filter((action) => action !== ExperimentAction.OpenTensorBoard);
    return getActionsForExperimentsUnion(selectedExperiments, [...batchActions], permissions);
    // Spreading batchActions is so TypeScript doesn't complain that it's readonly.
  }, [permissions, selectedExperiments, selection.type]);

  const completeFilterSet: Loadable<FilterFormSetWithoutId | undefined> = useMemo(() => {
    if (selection.type === 'ONLY_IN') {
      return NotLoaded;
    }
    return validFilterSet.map((fs) => {
      // filter group consisting of all excluded ids and project id
      const filterGroup = {
        children: [
          ...selection.exclusions.map((e) => ({
            columnName: 'id',
            kind: FormKind.Field,
            location: V1LocationType.EXPERIMENT,
            operator: Operator.NotEq,
            type: V1ColumnType.NUMBER,
            value: e,
          })),
          {
            columnName: 'projectId',
            kind: FormKind.Field,
            location: V1LocationType.EXPERIMENT,
            operator: Operator.Eq,
            type: V1ColumnType.NUMBER,
            value: project.id,
          },
        ],
        conjunction: Conjunction.And,
        kind: FormKind.Group,
      };
      return {
        ...fs,
        filterGroup: {
          children: [filterGroup, fs.filterGroup],
          conjunction: Conjunction.And,
          kind: FormKind.Group,
        },
      };
    });
  }, [selection, validFilterSet, project.id]);

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionResult | void> => {
      let params: BulkActionParams;
      const managedExperimentIds = selectedExperiments
        .filter((exp) => !exp.unmanaged)
        .map((exp) => exp.id);
      if (selection.type === 'ONLY_IN') {
        // TODO: when the selection spans pages, this check will not work --
        // should probably warn the user and/handle on the backend
        params = {
          experimentIds: selectedExperiments.filter((exp) => !exp.unmanaged).map((exp) => exp.id),
        };
      } else {
        params = completeFilterSet.match({
          _: () => ({
            experimentIds: [],
          }),
          Loaded: (filterSet) => {
            return {
              experimentIds: [],
              searchFilter: JSON.stringify(filterSet),
            };
          },
        });
      }
      switch (action) {
        case ExperimentAction.OpenTensorBoard: {
          if (managedExperimentIds.length !== selectedExperiments.length) {
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
      ExperimentMoveModal,
      ExperimentRetainLogsModal,
      completeFilterSet,
      openExperimentTensorBoardModal,
      project?.workspaceId,
      selectedExperiments,
      selection.type,
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

  const selectionLabel = useMemo(() => {
    return Loadable.match(total, {
      Failed: () => null,
      Loaded: (totalExperiments) => {
        let label = `${totalExperiments.toLocaleString()} ${pluralizer(
          totalExperiments,
          labelSingular.toLowerCase(),
        )}`;

        if (selection.type === 'ALL_EXCEPT') {
          const all = selection.exclusions.length === 0 ? 'All ' : '';
          const totalSelected =
            (totalExperiments - (selection.exclusions.length ?? 0)).toLocaleString() + ' ';
          label = `${all}${totalSelected}${labelPlural.toLowerCase()} selected`;
        } else if (selection.selections.length > 0) {
          label = `${selection.selections.length} of ${label} selected`;
        }

        return label;
      },
      NotLoaded: () => `Loading ${labelPlural.toLowerCase()}...`,
    });
  }, [selection, total, labelPlural, labelSingular]);

  const handleAction = useCallback((key: string) => handleBatchAction(key), [handleBatchAction]);

  return (
    <div className={css.base} data-test-component="tableActionBar">
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
              tabs={columnGroups}
              onVisibleColumnChange={onVisibleColumnChange}
            />
            <OptionsMenu
              rowHeight={rowHeight}
              tableViewMode={tableViewMode}
              onRowHeightChange={onRowHeightChange}
              onTableViewModeChange={onTableViewModeChange}
            />
            {(selection.type === 'ALL_EXCEPT' || selection.selections.length > 0) && (
              <Dropdown menu={editMenuItems} onClick={handleAction}>
                <Button hideChildren={isMobile}>Actions</Button>
              </Dropdown>
            )}
            {!isMobile && <span className={css.expNum}>{selectionLabel}</span>}
          </Row>
        </Column>
        <Column align="right">
          <Row>
            {heatmapBtnVisible && (
              <Tooltip content={'Toggle Metric Heatmap'}>
                <Button
                  icon={<Icon name="heatmap" title="heatmap" />}
                  type={heatmapOn ? 'primary' : 'default'}
                  onClick={() => onHeatmapToggle?.(heatmapOn ?? false)}
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
        experimentIds={selectedExperiments.reduce((acc, experiment) => {
          if (
            canActionExperiment(ExperimentAction.Move, experiment) &&
            permissions.canMoveExperiment({ experiment })
          ) {
            acc.push(experiment.id);
          }
          return acc;
        }, [] as number[])}
        filters={completeFilterSet.getOrElse(undefined)}
        sourceProjectId={project.id}
        sourceWorkspaceId={project.workspaceId}
        onSubmit={handleSubmitMove}
      />
      <ExperimentRetainLogsModal.Component
        experimentIds={selectedExperiments.reduce((acc, experiment) => {
          if (
            canActionExperiment(ExperimentAction.RetainLogs, experiment) &&
            permissions.canModifyExperiment({
              workspace: { id: experiment.workspaceId },
            })
          ) {
            acc.push(experiment.id);
          }
          return acc;
          }, [] as number[])}
        filters={completeFilterSet.getOrElse(undefined)}
        onSubmit={handleSubmitRetainLogs}
      />
      <ExperimentTensorBoardModalComponent
        filters={completeFilterSet.getOrElse(undefined)}
        selectedExperiments={selectedExperiments}
        workspaceId={project?.workspaceId}
      />
    </div>
  );
};

export default TableActionBar;
