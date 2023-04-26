import { Menu, Popover, Space } from 'antd';
import { ItemType } from 'rc-menu/lib/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import Dropdown from 'components/Dropdown';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import Button from 'components/kit/Button';
import Checkbox from 'components/kit/Checkbox';
import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { useModal } from 'components/kit/Modal';
import Pivot from 'components/kit/Pivot';
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
import { V1BulkExperimentFilters, V1LocationType } from 'services/api-ts-sdk';
import Icon from 'shared/components/Icon';
import Spinner from 'shared/components/Spinner';
import { RecordKey } from 'shared/types';
import { ErrorLevel } from 'shared/utils/error';
import {
  BulkActionResult,
  ExperimentAction,
  ExperimentItem,
  Project,
  ProjectColumn,
  ProjectExperiment,
} from 'types';
import { notification } from 'utils/dialogApi';
import handleError from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperimentsUnion,
  getProjectExperimentForExperimentItem,
} from 'utils/experiment';
import { Loadable } from 'utils/loadable';
import { openCommandResponse } from 'utils/wait';

import { defaultExperimentColumns } from './columns';
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

const actionIcons: Record<BatchAction, string> = {
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
  experiments: Loadable<ExperimentItem>[];
  filters: V1BulkExperimentFilters;
  initialVisibleColumns: string[];
  onAction: () => Promise<void>;
  selectAll: boolean;
  selectedExperimentIds: number[];
  handleUpdateExperimentList: (action: BatchAction, successfulIds: number[]) => void;
  setVisibleColumns: Dispatch<SetStateAction<string[]>>;
  project: Project;
  projectColumns: Loadable<ProjectColumn[]>;
  total: Loadable<number>;
}

const TableActionBar: React.FC<Props> = ({
  experiments,
  filters,
  onAction,
  selectAll,
  selectedExperimentIds,
  handleUpdateExperimentList,
  project,
  projectColumns,
  total,
  initialVisibleColumns,
  setVisibleColumns,
}) => {
  const permissions = usePermissions();
  const [batchAction, setBatchAction] = useState<BatchAction>();
  const BatchActionConfirmModal = useModal(BatchActionConfirmModalComponent);
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);
  const [form] = Form.useForm();
  const [filteredColumns, setFilteredColumns] = useState(projectColumns);

  const experimentMap = useMemo(() => {
    return experiments.filter(Loadable.isLoaded).reduce((acc, experiment) => {
      acc[experiment.data.id] = getProjectExperimentForExperimentItem(experiment.data, project);
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
      switch (action) {
        case ExperimentAction.OpenTensorBoard:
          return openCommandResponse(
            await openOrCreateTensorBoard({
              experimentIds: selectedExperimentIds,
              filters: selectAll ? filters : undefined,
              workspaceId: project?.workspaceId,
            }),
          );
        case ExperimentAction.Move:
          return ExperimentMoveModal.open();
        case ExperimentAction.Activate:
          return await activateExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? filters : undefined,
          });
        case ExperimentAction.Archive:
          return await archiveExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? filters : undefined,
          });
        case ExperimentAction.Cancel:
          return await cancelExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? filters : undefined,
          });
        case ExperimentAction.Kill:
          return await killExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? filters : undefined,
          });
        case ExperimentAction.Pause:
          return await pauseExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? filters : undefined,
          });
        case ExperimentAction.Unarchive:
          return await unarchiveExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? filters : undefined,
          });
        case ExperimentAction.Delete:
          return await deleteExperiments({
            experimentIds: selectedExperimentIds,
            filters: selectAll ? filters : undefined,
          });
      }
    },
    [selectedExperimentIds, selectAll, filters, project?.workspaceId, ExperimentMoveModal],
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

  const editMenuItems: ItemType[] = useMemo(() => {
    return batchActions.map((action) => ({
      danger: action === ExperimentAction.Delete,
      disabled: !availableBatchActions.includes(action),
      // The icon doesn't show up without being wrapped in a div.
      icon: (
        <div>
          <Icon name={actionIcons[action]} />
        </div>
      ),
      key: action,
      label: action,
    }));
  }, [availableBatchActions]);

  const handleAction = useCallback(
    ({ key }: { key: string }) => {
      handleBatchAction(key);
    },
    [handleBatchAction],
  );

  const columnSearch: string = Form.useWatch('column-search', form) ?? '';

  useEffect(() => {
    const regex = new RegExp(columnSearch, 'i');
    setFilteredColumns(
      Loadable.map(projectColumns, (columns) =>
        columns.filter((col) => regex.test(col.displayName ?? col.column)),
      ),
    );
  }, [columnSearch, projectColumns]);

  const generalColumns: Record<string, boolean> = Form.useWatch('general', form);
  const hyperparametersColumns: Record<string, boolean> = Form.useWatch('hyperparameters', form);
  const metricsColumns: Record<string, boolean> = Form.useWatch('metrics', form);

  useEffect(() => {
    const allColumns = { ...generalColumns, ...hyperparametersColumns, ...metricsColumns };
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    setVisibleColumns((_prevColumns) => {
      const newCols = [];
      for (const [key, value] of Object.entries(allColumns)) {
        if (value === true) newCols.push(key);
      }
      return newCols;
    });
  }, [generalColumns, hyperparametersColumns, metricsColumns, setVisibleColumns]);

  const handleShowSuggested = useCallback(() => {
    setVisibleColumns(defaultExperimentColumns);
  }, [setVisibleColumns]);

  const handleShowHideAll = useCallback(() => {
    //form.setFieldsValue({}) TODO
  }, []);

  const tabContent = useCallback(
    (columns: ProjectColumn[], tab: V1LocationType) => {
      return (
        <div>
          <Form.Item name="column-search">
            <Input allowClear placeholder="Search" />
          </Form.Item>
          <div style={{ maxHeight: 360, overflow: 'hidden auto' }}>
            {columns
              .filter((column) => column.location === tab)
              .map((column) => (
                <Form.Item
                  initialValue={initialVisibleColumns.includes(column.column)}
                  key={column.column}
                  name={[tab, column.column]}
                  valuePropName="checked">
                  <Checkbox>{column.displayName ?? column.column}</Checkbox>
                </Form.Item>
              ))}
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Button type="text" onClick={handleShowHideAll}>
              Show all
            </Button>
            <Button type="text" onClick={handleShowSuggested}>
              Show suggested
            </Button>
          </div>
        </div>
      );
    },
    [handleShowHideAll, handleShowSuggested, initialVisibleColumns],
  );

  return (
    <>
      <Space className={css.base}>
        <Popover
          content={
            <div style={{ width: '300px' }}>
              <Form form={form}>
                {Loadable.match(filteredColumns, {
                  Loaded: (columns) => (
                    <Pivot
                      items={[
                        {
                          children: tabContent(columns, V1LocationType.EXPERIMENT),
                          forceRender: true,
                          key: 'general',
                          label: 'General',
                        },
                        {
                          children: tabContent(columns, V1LocationType.VALIDATIONS),
                          forceRender: true,
                          key: 'metrics',
                          label: 'Metrics',
                        },
                        {
                          children: tabContent(columns, V1LocationType.HYPERPARAMETERS),
                          forceRender: true,
                          key: 'hyperparameters',
                          label: 'Hyperparameters',
                        },
                      ]}
                    />
                  ),
                  NotLoaded: () => <Spinner />,
                })}
              </Form>
            </div>
          }
          placement="bottom"
          trigger="click">
          <Button>Columns</Button>
        </Popover>
        {(selectAll || selectedExperimentIds.length > 0) && (
          <Dropdown content={<Menu items={editMenuItems} onClick={handleAction} />}>
            <Button icon={<Icon name="pencil" />}>
              Edit (
              {selectAll
                ? Loadable.isLoaded(total)
                  ? total.data.toLocaleString()
                  : 'All'
                : selectedExperimentIds.length}
              )
            </Button>
          </Dropdown>
        )}
      </Space>
      {batchAction && (
        <BatchActionConfirmModal.Component
          batchAction={batchAction}
          selectAll={selectAll}
          onConfirm={() => submitBatchAction(batchAction)}
        />
      )}
      <ExperimentMoveModal.Component
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
    </>
  );
};

export default TableActionBar;
