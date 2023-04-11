import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Menu, Space } from 'antd';
import { ItemType } from 'rc-menu/lib/interface';
import React, { useCallback, useMemo } from 'react';

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
  BulkActionError,
  ExperimentAction,
  ExperimentItem,
  Project,
  ProjectExperiment,
} from 'types';
import { modal } from 'utils/dialogApi';
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
  ExperimentAction.Activate,
  ExperimentAction.Move,
  ExperimentAction.Pause,
  ExperimentAction.Archive,
  ExperimentAction.Unarchive,
  ExperimentAction.Cancel,
  ExperimentAction.Kill,
  ExperimentAction.Delete,
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
  onAction: () => void;
  selectAll: boolean;
  selectedExperimentIds: number[];
  project: Project;
}

const TableActionBar: React.FC<Props> = ({
  experiments,
  filters,
  onAction,
  selectAll,
  selectedExperimentIds,
  project,
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
    if (selectAll) return [...batchActions];
    const experiments = selectedExperimentIds.map((id) => experimentMap[id]) ?? [];
    return getActionsForExperimentsUnion(experiments, [...batchActions], permissions);
    // Spreading batchActions is so TypeScript doesn't complain that it's readonly.
  }, [experimentMap, permissions, selectAll, selectedExperimentIds]);

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionError[] | void> => {
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

  const submitBatchAction = useCallback(
    async (action: BatchAction) => {
      try {
        await sendBatchActions(action);

        onAction();
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
      }
    },
    [sendBatchActions, onAction],
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
              Edit ({selectAll ? 'All' : selectedExperimentIds.length})
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
        onClose={onAction}
      />
    </>
  );
};

export default TableActionBar;
