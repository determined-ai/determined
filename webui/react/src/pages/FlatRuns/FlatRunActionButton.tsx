import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon, { IconName } from 'hew/Icon';
import { useModal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import { useCallback, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import { BatchAction } from 'components/TableActionBar';
// import usePermissions from 'hooks/usePermissions';
import {
  archiveRuns,
  deleteRuns,
  killRuns,
  openOrCreateTensorBoard,
  unarchiveRuns,
} from 'services/api';
import { BulkActionResult, ExperimentAction, FlatRun, Project } from 'types';
import handleError from 'utils/error';
import { capitalizeWord } from 'utils/string';
import { openCommandResponse } from 'utils/wait';

// export const getActionsForRunsUnion = (
//   experiments: FlatRun[],
//   targets: ExperimentAction[],
//   permissions: ExperimentPermissionSet,
// ): ExperimentAction[] => {
//   if (!experiments.length) return []; // redundant, for clarity
//   const actionsForExperiments = experiments.map((e) =>
//     getActionsForExperiment(e, targets, permissions),
//   );
//   return targets.filter((action) =>
//     actionsForExperiments.some((experimentActions) => experimentActions.includes(action)),
//   );
// };

const BATCH_ACTIONS = [
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

const ACTION_ICONS: Record<BatchAction, IconName> = {
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

const LABEL_PLURAL = 'runs';

interface Props {
  isMobile: boolean;
  selectedRuns: FlatRun[];
  project: Project;
  onActionSuccess?: (action: BatchAction, successfulIds: number[]) => void;
}

const FlatRunActionButton = ({
  isMobile,
  selectedRuns,
  project,
  onActionSuccess,
}: Props): JSX.Element => {
  const [batchAction, setBatchAction] = useState<BatchAction | undefined>(undefined);
  // const permissions = usePermissions();
  const { openToast } = useToast();
  const BatchActionConfirmModal = useModal(BatchActionConfirmModalComponent);

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionResult | void> => {
      const validRunIds = selectedRuns
        // .filter((exp) => canActionExperiment(action, exp)) TODO: Runs permission
        .map((run) => run.id);
      const params = {
        projectId: project.id,
        runIds: validRunIds,
      };
      switch (action) {
        case ExperimentAction.OpenTensorBoard: {
          if (validRunIds.length !== selectedRuns.length) {
            // if unmanaged experiments are selected, open experimentTensorBoardModal
            // openExperimentTensorBoardModal(); // TODO Tensorboard for Runs
          } else {
            openCommandResponse(
              await openOrCreateTensorBoard({
                experimentIds: params.runIds,
                workspaceId: project.workspaceId,
              }),
            );
          }
          return;
        }
        case ExperimentAction.Move:
        //   return ExperimentMoveModal.open();
        case ExperimentAction.RetainLogs:
        //   return ExperimentRetainLogsModal.open();
        case ExperimentAction.Activate:
        // return await activate(params);
        case ExperimentAction.Archive:
          return await archiveRuns(params);
        case ExperimentAction.Cancel:
        //   return await cancelExperiments(params);
        case ExperimentAction.Kill:
          return await killRuns(params);
        case ExperimentAction.Pause:
        //   return await pauseExperiments(params);
        case ExperimentAction.Unarchive:
          return await unarchiveRuns(params);
        case ExperimentAction.Delete:
          return await deleteRuns(params);
      }
    },
    [project.id, project.workspaceId, selectedRuns],
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
            description: `No selected ${LABEL_PLURAL.toLowerCase()} were eligible for ${action.toLowerCase()}`,
            title: `No eligible ${LABEL_PLURAL.toLowerCase()}`,
          });
        } else if (numFailures === 0) {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${
              results.successful.length
            } ${LABEL_PLURAL.toLowerCase()}`,
            title: `${action} Success`,
          });
        } else if (numSuccesses === 0) {
          openToast({
            description: `Unable to ${action.toLowerCase()} ${numFailures} ${LABEL_PLURAL.toLowerCase()}`,
            severity: 'Warning',
            title: `${action} Failure`,
          });
        } else {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${numSuccesses} out of ${
              numFailures + numSuccesses
            } eligible
            ${LABEL_PLURAL.toLowerCase()}`,
            severity: 'Warning',
            title: `Partial ${action} Failure`,
          });
        }
      } catch (e) {
        const publicSubject =
          action === ExperimentAction.OpenTensorBoard
            ? `Unable to View TensorBoard for Selected ${capitalizeWord(LABEL_PLURAL)}`
            : `Unable to ${action} Selected ${capitalizeWord(LABEL_PLURAL)}`;
        handleError(e, {
          isUserTriggered: true,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
        });
      } finally {
        // onActionComplete?.();
      }
    },
    [sendBatchActions, onActionSuccess, openToast],
  );

  const handleBatchAction = useCallback(
    (action: string) => {
      switch (action) {
        case ExperimentAction.OpenTensorBoard:
          submitBatchAction(action);
          break;
        case ExperimentAction.Move:
        case ExperimentAction.RetainLogs:
          sendBatchActions(action);
          break;
        default:
          setBatchAction(action as BatchAction);
          BatchActionConfirmModal.open();
          break;
      }
    },
    [BatchActionConfirmModal, sendBatchActions, submitBatchAction],
  );

  // const availableBatchActions = useMemo(() => {
  //   return getActionsForExperimentsUnion(selectedRuns, [...BATCH_ACTIONS], permissions);
  //   // Spreading batchActions is so TypeScript doesn't complain that it's readonly.
  // }, [selectedExperimentIds, experimentMap, permissions]);

  const editMenuItems = useMemo(() => {
    const groupedBatchActions = [
      BATCH_ACTIONS.slice(0, 1), // View in TensorBoard
      BATCH_ACTIONS.slice(1, 5), // Move, Archive, Unarchive, Delete
      BATCH_ACTIONS.slice(5), // Resume, Pause, Cancel, Kill
    ];
    const groupSize = groupedBatchActions.length;
    return groupedBatchActions.reduce((acc: MenuItem[], group, index) => {
      const isLastGroup = index === groupSize - 1;
      group.forEach((action) =>
        acc.push({
          danger: action === ExperimentAction.Delete,
          // disabled: !availableBatchActions.includes(action), // TODO uncomment later
          icon: <Icon name={ACTION_ICONS[action]} title={action} />,
          key: action,
          label: action,
        }),
      );
      if (!isLastGroup) acc.push({ type: 'divider' });
      return acc;
    }, []);
  }, []);

  return (
    <>
      {selectedRuns.length > 0 && (
        <Dropdown menu={editMenuItems} onClick={handleBatchAction}>
          <Button hideChildren={isMobile} icon={<Icon decorative name="heat" />}>
            Actions
          </Button>
        </Dropdown>
      )}
      {batchAction && (
        <BatchActionConfirmModal.Component
          batchAction={batchAction}
          // isUnmanagedIncluded={selectedExperiments.some((exp) => exp.unmanaged)} // TODO: is it needed for Runs?
          onConfirm={() => submitBatchAction(batchAction)}
        />
      )}
    </>
  );
};

export default FlatRunActionButton;
