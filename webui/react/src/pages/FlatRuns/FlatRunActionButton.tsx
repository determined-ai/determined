import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon, { IconName } from 'hew/Icon';
import { useModal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import { Loadable } from 'hew/utils/loadable';
import { List } from 'immutable';
import { useObservable } from 'micro-observables';
import { useCallback, useMemo, useState } from 'react';

import BatchActionConfirmModalComponent from 'components/BatchActionConfirmModal';
import Link from 'components/Link';
import usePermissions from 'hooks/usePermissions';
import FlatRunMoveModalComponent from 'pages/FlatRuns/FlatRunMoveModal';
import { paths } from 'routes/utils';
import {
  archiveRuns,
  deleteRuns,
  killRuns,
  pauseRuns,
  resumeRuns,
  unarchiveRuns,
} from 'services/api';
import { RunBulkActionParams } from 'services/types';
import projectStore from 'stores/projects';
import { BulkActionResult, ExperimentAction, FlatRun, Project, SelectionType } from 'types';
import handleError from 'utils/error';
import { canActionFlatRun, getActionsForFlatRunsUnion, getIdsFilter } from 'utils/flatRun';
import { capitalizeWord, pluralizer } from 'utils/string';

const BATCH_ACTIONS = [
  ExperimentAction.Move,
  ExperimentAction.Archive,
  ExperimentAction.Unarchive,
  ExperimentAction.Pause,
  ExperimentAction.Activate,
  ExperimentAction.Kill,
  ExperimentAction.Delete,
] as const;

type BatchAction = (typeof BATCH_ACTIONS)[number];

const ACTION_ICONS: Record<BatchAction, IconName> = {
  [ExperimentAction.Archive]: 'archive',
  [ExperimentAction.Unarchive]: 'document',
  [ExperimentAction.Move]: 'workspaces',
  [ExperimentAction.Kill]: 'cancelled',
  [ExperimentAction.Delete]: 'error',
  [ExperimentAction.Activate]: 'play',
  [ExperimentAction.Pause]: 'pause',
} as const;

const LABEL_PLURAL = 'runs';

interface Props {
  tableFilterString: string;
  isMobile: boolean;
  selectedRuns: ReadonlyArray<Readonly<FlatRun>>;
  projectId: number;
  workspaceId: number;
  onActionSuccess?: (action: BatchAction, successfulIds: number[]) => void;
  onActionComplete?: () => void | Promise<void>;
  selection: SelectionType;
}

const FlatRunActionButton = ({
  tableFilterString,
  isMobile,
  selectedRuns,
  projectId,
  selection,
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
  const loadableProjects: Loadable<List<Project>> = useObservable(
    projectStore.getProjectsByWorkspace(workspaceId),
  );

  const sendBatchActions = useCallback(
    async (action: BatchAction): Promise<BulkActionResult | void> => {
      const params: RunBulkActionParams = { projectId };
      if (selection.type === 'ONLY_IN') {
        const validRunIds = selectedRuns
          .filter((run) => canActionFlatRun(action, run))
          .map((run) => run.id);
        params.runIds = validRunIds;
      } else if (selection.type === 'ALL_EXCEPT') {
        const filters = JSON.parse(tableFilterString);
        params.filter = JSON.stringify(getIdsFilter(filters, selection));
      }
      switch (action) {
        case ExperimentAction.Move:
          flatRunMoveModalOpen();
          break;
        case ExperimentAction.Archive:
          return await archiveRuns(params);
        case ExperimentAction.Kill:
          return await killRuns(params);
        case ExperimentAction.Unarchive:
          return await unarchiveRuns(params);
        case ExperimentAction.Delete:
          return await deleteRuns(params);
        case ExperimentAction.Pause:
          return await pauseRuns(params);
        case ExperimentAction.Activate:
          return await resumeRuns(params);
      }
    },
    [flatRunMoveModalOpen, projectId, selectedRuns, selection, tableFilterString],
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
            description: `${action} succeeded for ${results.successful.length} ${pluralizer(results.successful.length, 'run')}`,
            title: `${action} Success`,
          });
        } else if (numSuccesses === 0) {
          openToast({
            description: `Unable to ${action.toLowerCase()} ${numFailures} ${pluralizer(numFailures, 'run')}`,
            severity: 'Warning',
            title: `${action} Failure`,
          });
        } else {
          openToast({
            closeable: true,
            description: `${action} succeeded for ${numSuccesses} out of ${numFailures + numSuccesses}{' '}
            ${pluralizer(numFailures + numSuccesses, 'run')}`,
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
    if (selection.type === 'ONLY_IN') {
      return getActionsForFlatRunsUnion(selectedRuns, [...BATCH_ACTIONS], permissions);
    } else if (selection.type === 'ALL_EXCEPT') {
      return BATCH_ACTIONS;
    }
    return [];
  }, [selection.type, selectedRuns, permissions]);

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

  const onSubmit = useCallback(
    async (results: BulkActionResult, destinationProjectId: number) => {
      const numSuccesses = results?.successful.length ?? 0;
      const numFailures = results?.failed.length ?? 0;

      const destinationProjectName =
        Loadable.getOrElse(List<Project>(), loadableProjects).find(
          (p) => p.id === destinationProjectId,
        )?.name ?? '';

      if (numSuccesses === 0 && numFailures === 0) {
        openToast({
          description: 'No selected runs were eligible for moving',
          title: 'No eligible runs',
        });
      } else if (numFailures === 0) {
        openToast({
          closeable: true,
          description: `${results.successful.length} ${pluralizer(results.successful.length, 'run')} moved to project ${destinationProjectName}`,
          link: <Link path={paths.projectDetails(destinationProjectId)}>View Project</Link>,
          title: 'Move Success',
        });
      } else if (numSuccesses === 0) {
        openToast({
          description: `Unable to move ${numFailures} ${pluralizer(numFailures, 'run')}`,
          severity: 'Warning',
          title: 'Move Failure',
        });
      } else {
        openToast({
          closeable: true,
          description: `${numFailures} out of ${numFailures + numSuccesses} eligible ${pluralizer(numFailures + numSuccesses, 'run')} failed to move to project ${destinationProjectName}`,
          link: <Link path={paths.projectDetails(destinationProjectId)}>View Project</Link>,
          severity: 'Warning',
          title: 'Partial Move Failure',
        });
      }
      await onActionComplete?.();
    },
    [loadableProjects, onActionComplete, openToast],
  );

  return (
    <>
      {selectedRuns.length > 0 && (
        <Dropdown menu={editMenuItems} onClick={handleBatchAction}>
          <Button hideChildren={isMobile} icon={<Icon decorative name="pencil" />}>
            Actions
          </Button>
        </Dropdown>
      )}
      {batchAction && (
        <BatchActionConfirmModal.Component
          batchAction={batchAction}
          isUnmanagedIncluded={selectedRuns.some((run) => run.experiment?.unmanaged ?? false)}
          itemName="run"
          onConfirm={() => submitBatchAction(batchAction)}
        />
      )}
      <FlatRunMoveComponentModal
        flatRuns={[...selectedRuns]}
        sourceProjectId={projectId}
        sourceWorkspaceId={workspaceId}
        onSubmit={onSubmit}
      />
    </>
  );
};

export default FlatRunActionButton;
