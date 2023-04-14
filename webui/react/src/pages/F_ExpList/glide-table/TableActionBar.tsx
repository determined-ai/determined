import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Menu, Space } from 'antd';
import { ItemType } from 'rc-menu/lib/interface';
import React, { Dispatch, SetStateAction, useCallback, useMemo } from 'react';

import Dropdown from 'components/Dropdown';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
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
import Icon from 'shared/components/Icon';
import { RecordKey } from 'shared/types';
import { ErrorLevel } from 'shared/utils/error';
import {
  BulkActionResult,
  ExperimentAction,
  ExperimentItem,
  Project,
  ProjectExperiment,
  RunState,
} from 'types';
import { modal, notification } from 'utils/dialogApi';
import handleError from 'utils/error';
import {
  canActionExperiment,
  getActionsForExperimentsUnion,
  getProjectExperimentForExperimentItem,
} from 'utils/experiment';
import { Loadable } from 'utils/loadable';
import { openCommandResponse } from 'utils/wait';

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

type BatchAction = (typeof batchActions)[number];

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
  onAction: () => Promise<void>;
  selectAll: boolean;
  selectedExperimentIds: number[];
  setExperiments: Dispatch<SetStateAction<Loadable<ExperimentItem>[]>>;
  project: Project;
  total: Loadable<number>;
}

const TableActionBar: React.FC<Props> = ({
  experiments,
  filters,
  onAction,
  selectAll,
  selectedExperimentIds,
  setExperiments,
  project,
  total,
}) => {
  const permissions = usePermissions();
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);

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

  const handleUpdateExperimentList = useCallback(
    (action: BatchAction, successfulIds: number[]) => {
      const idSet = new Set(successfulIds);
      switch (action) {
        case ExperimentAction.OpenTensorBoard:
          break;
        case ExperimentAction.Activate:
          setExperiments((prev) =>
            prev.map((expLoadable) =>
              Loadable.map(expLoadable, (experiment) =>
                idSet.has(experiment.id) ? { ...experiment, state: RunState.Active } : experiment,
              ),
            ),
          );
          break;
        case ExperimentAction.Archive:
          setExperiments((prev) =>
            prev.map((expLoadable) =>
              Loadable.map(expLoadable, (experiment) =>
                idSet.has(experiment.id) ? { ...experiment, archived: true } : experiment,
              ),
            ),
          );
          break;
        case ExperimentAction.Cancel:
          setExperiments((prev) =>
            prev.map((expLoadable) =>
              Loadable.map(expLoadable, (experiment) =>
                idSet.has(experiment.id)
                  ? { ...experiment, state: RunState.StoppingCanceled }
                  : experiment,
              ),
            ),
          );
          break;
        case ExperimentAction.Kill:
          setExperiments((prev) =>
            prev.map((expLoadable) =>
              Loadable.map(expLoadable, (experiment) =>
                idSet.has(experiment.id)
                  ? { ...experiment, state: RunState.StoppingKilled }
                  : experiment,
              ),
            ),
          );
          break;
        case ExperimentAction.Pause:
          setExperiments((prev) =>
            prev.map((expLoadable) =>
              Loadable.map(expLoadable, (experiment) =>
                idSet.has(experiment.id) ? { ...experiment, state: RunState.Paused } : experiment,
              ),
            ),
          );
          break;
        case ExperimentAction.Unarchive:
          setExperiments((prev) =>
            prev.map((expLoadable) =>
              Loadable.map(expLoadable, (experiment) =>
                idSet.has(experiment.id) ? { ...experiment, archived: false } : experiment,
              ),
            ),
          );
          break;
        case ExperimentAction.Move:
        case ExperimentAction.Delete:
          setExperiments((prev) =>
            prev.filter((expLoadable) =>
              Loadable.match(expLoadable, {
                Loaded: (experiment) => !idSet.has(experiment.id),
                NotLoaded: () => true,
              }),
            ),
          );
          break;
      }
    },
    [setExperiments],
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

  const showConfirmation = useCallback(
    (action: BatchAction) => {
      modal.confirm({
        content: `
        Are you sure you want to ${action.toLocaleLowerCase()}
        all eligible ${
          selectAll ? 'experiments matching the current filters' : 'selected experiments'
        }?
      `,
        icon: <ExclamationCircleOutlined />,
        okText: /cancel/i.test(action) ? 'Confirm' : action,
        onOk: () => submitBatchAction(action),
        title: 'Confirm Batch Action',
      });
    },
    [selectAll, submitBatchAction],
  );

  const handleBatchAction = useCallback(
    (action: string) => {
      if (action === ExperimentAction.OpenTensorBoard) {
        submitBatchAction(action);
      } else if (action === ExperimentAction.Move) {
        sendBatchActions(action);
      } else {
        showConfirmation(action as BatchAction);
      }
    },
    [submitBatchAction, sendBatchActions, showConfirmation],
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

  return (
    <>
      <Space className={css.base}>
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
