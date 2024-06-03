import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon, { IconName } from 'hew/Icon';
import { useModal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import { useCallback, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import usePermissions from 'hooks/usePermissions';
import FlatRunMoveModalComponent from 'pages/FlatRuns/FlatRunMoveModal';
import { archiveRuns, deleteRuns, killRuns, unarchiveRuns } from 'services/api';
import { BulkActionResult, ExperimentAction, FlatRun } from 'types';
import handleError from 'utils/error';
import { canActionFlatRun, getActionsForFlatRunsUnion } from 'utils/flatRun';
import { capitalizeWord } from 'utils/string';

const BATCH_ACTIONS = [
  ExperimentAction.Move,
  ExperimentAction.Archive,
  ExperimentAction.Unarchive,
  ExperimentAction.Delete,
  ExperimentAction.Kill,
] as const;

type BatchAction = (typeof BATCH_ACTIONS)[number];

const ACTION_ICONS: Record<BatchAction, IconName> = {
  [ExperimentAction.Archive]: 'archive',
  [ExperimentAction.Unarchive]: 'document',
  [ExperimentAction.Move]: 'workspaces',
  [ExperimentAction.Kill]: 'cancelled',
  [ExperimentAction.Delete]: 'error',
} as const;

const LABEL_PLURAL = 'runs';

interface Props {
  isMobile: boolean;
  selectedRuns: ReadonlyArray<Readonly<FlatRun>>;
  projectId: number;
  workspaceId: number;
  onActionSuccess?: (action: BatchAction, successfulIds: number[]) => void;
  onActionComplete?: () => Promise<void>;
}

const FlatRunActionButton = ({
  isMobile,
  selectedRuns,
  projectId,
  workspaceId,
  onActionSuccess,
  onActionComplete,
}: Props): JSX.Element => {
  const [batchAction, setBatchAction] = useState<BatchAction | undefined>(undefined);
  const permissions = usePermissions();
  const { openToast } = useToast();
  const { Component: FlatRunMoveComponentModal, open: flatRunMoveModalOpen } =
    useModal(FlatRunMoveModalComponent);
  const BatchActionConfirmModal = useModal(BatchActionConfirmModalComponent);

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionResult | void> => {
      const validRunIds = selectedRuns
        .filter((exp) => canActionFlatRun(action, exp))
        .map((run) => run.id);
      const params = {
        projectId,
        runIds: validRunIds,
      };
      switch (action) {
        case ExperimentAction.Move:
          return flatRunMoveModalOpen();
        case ExperimentAction.Archive:
          return await archiveRuns(params);
        case ExperimentAction.Kill:
          return await killRuns(params);
        case ExperimentAction.Unarchive:
          return await unarchiveRuns(params);
        case ExperimentAction.Delete:
          return await deleteRuns(params);
        default:
          break;
      }
    },
    [flatRunMoveModalOpen, projectId, selectedRuns],
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
        await onActionComplete?.();
      } catch (e) {
        const publicSubject = `Unable to ${action} Selected ${capitalizeWord(LABEL_PLURAL)}`;
        handleError(e, {
          isUserTriggered: true,
          publicMessage: 'Please try again later.',
          publicSubject,
          silent: false,
        });
      }
    },
    [sendBatchActions, onActionSuccess, openToast, onActionComplete],
  );

  const handleBatchAction = useCallback(
    (action: string) => {
      switch (action) {
        case ExperimentAction.Move:
          sendBatchActions(action);
          break;
        default:
          setBatchAction(action as BatchAction);
          BatchActionConfirmModal.open();
          break;
      }
    },
    [BatchActionConfirmModal, sendBatchActions],
  );

  const availableBatchActions = useMemo(() => {
    return getActionsForFlatRunsUnion(selectedRuns, [...BATCH_ACTIONS], permissions);
  }, [selectedRuns, permissions]);

  const editMenuItems = useMemo(() => {
    const groupedBatchActions = [BATCH_ACTIONS];
    const groupSize = groupedBatchActions.length;
    return groupedBatchActions.reduce((acc: MenuItem[], group, index) => {
      const isLastGroup = index === groupSize - 1;
      group.forEach((action) =>
        acc.push({
          danger: action === ExperimentAction.Delete,
          disabled: !availableBatchActions.includes(action),
          icon: <Icon name={ACTION_ICONS[action]} title={action} />,
          key: action,
          label: action,
        }),
      );
      if (!isLastGroup) acc.push({ type: 'divider' });
      return acc;
    }, []);
  }, [availableBatchActions]);

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
          isUnmanagedIncluded={selectedRuns.some((run) => run.experiment?.unmanaged ?? false)}
          onConfirm={() => submitBatchAction(batchAction)}
        />
      )}
      <FlatRunMoveComponentModal
        flatRuns={[...selectedRuns]}
        sourceProjectId={projectId}
        sourceWorkspaceId={workspaceId}
      />
    </>
  );
};

export default FlatRunActionButton;
