import { GridCell } from '@glideapps/glide-data-grid';
import Button from 'hew/Button';
import { ContextMenuCompleteHandlerProps } from 'hew/DataGrid/contextMenu';
import Dropdown, { DropdownEvent, MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import { useModal } from 'hew/Modal';
import { useToast } from 'hew/Toast';
import useConfirm from 'hew/useConfirm';
import { copyToClipboard } from 'hew/utils/functions';
import { Failed, Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { isString } from 'lodash';
import React, { useCallback, useMemo, useRef, useState } from 'react';

import css from 'components/ActionDropdown/ActionDropdown.module.scss';
import ExperimentEditModalComponent from 'components/ExperimentEditModal';
import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import ExperimentRetainLogsModalComponent from 'components/ExperimentRetainLogsModal';
import HyperparameterSearchModalComponent from 'components/HyperparameterSearchModal';
import InterstitialModalComponent, {
  type onInterstitialCloseActionType,
} from 'components/InterstitialModalComponent';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { handlePath } from 'routes/utils';
import {
  activateExperiment,
  archiveExperiment,
  cancelExperiment,
  deleteExperiment,
  getExperiment,
  killExperiment,
  openOrCreateTensorBoard,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import {
  BulkExperimentItem,
  ExperimentAction,
  FullExperimentItem,
  ProjectExperiment,
  ValueOf,
} from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { getActionsForExperiment } from 'utils/experiment';
import { capitalize } from 'utils/string';
import { openCommandResponse } from 'utils/wait';

interface Props {
  children?: React.ReactNode;
  cell?: GridCell;
  experiment: ProjectExperiment;
  isContextMenu?: boolean;
  link?: string;
  makeOpen?: boolean;
  onComplete?: ContextMenuCompleteHandlerProps<ExperimentAction, BulkExperimentItem>;
  onLink?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  workspaceId?: number;
}

export const Action = {
  Copy: 'Copy Value',
  NewTab: 'Open Link in New Tab',
  NewWindow: 'Open Link in New Window',
  ...ExperimentAction,
};

type Action = ValueOf<typeof Action>;

const dropdownActions = [
  Action.SwitchPin,
  Action.Activate,
  Action.Pause,
  Action.Archive,
  Action.Unarchive,
  Action.Cancel,
  Action.Kill,
  Action.Edit,
  Action.Move,
  Action.RetainLogs,
  Action.OpenTensorBoard,
  Action.HyperparameterSearch,
  Action.Delete,
];

const ExperimentActionDropdown: React.FC<Props> = ({
  experiment,
  cell,
  isContextMenu,
  link,
  makeOpen,
  onComplete,
  onLink,
  onVisibleChange,
  children,
}: Props) => {
  const ExperimentEditModal = useModal(ExperimentEditModalComponent);
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);
  const ExperimentRetainLogsModal = useModal(ExperimentRetainLogsModalComponent);
  const {
    Component: HyperparameterSearchModal,
    open: hyperparameterSearchModalOpen,
    close: hyperparameterSearchModalClose,
  } = useModal(HyperparameterSearchModalComponent);
  const {
    Component: InterstitialModal,
    open: interstitialModalOpen,
    close: interstitialModalClose,
  } = useModal(InterstitialModalComponent);
  const [experimentItem, setExperimentItem] = useState<Loadable<FullExperimentItem>>(NotLoaded);
  const canceler = useRef<AbortController>(new AbortController());
  const confirm = useConfirm();
  const { openToast } = useToast();
  const f_flat_runs = useFeature().isOn('flat_runs');

  const entityName = f_flat_runs ? 'search' : 'experiment';

  // this is required when experiment does not contain `config`.
  // since we removed config. See #8765 on GitHub
  const fetchedExperimentItem = useCallback(async () => {
    try {
      setExperimentItem(NotLoaded);
      const response: FullExperimentItem = await getExperiment(
        { id: experiment.id },
        { signal: canceler.current.signal },
      );
      setExperimentItem(Loaded(response));
    } catch (e) {
      handleError(e, { publicSubject: `Unable to fetch ${entityName} data.` });
      setExperimentItem(Failed(new Error('experiment data failure')));
    }
  }, [entityName, experiment.id]);

  const onInterstitalClose: onInterstitialCloseActionType = useCallback(
    (reason) => {
      switch (reason) {
        case 'ok':
          hyperparameterSearchModalOpen();
          break;
        case 'failed':
          break;
        case 'close':
          canceler.current.abort();
          canceler.current = new AbortController();
          break;
      }
      interstitialModalClose(reason);
    },
    [hyperparameterSearchModalOpen, interstitialModalClose],
  );

  const handleEditComplete = useCallback(
    (data: Partial<BulkExperimentItem>) => {
      onComplete?.(ExperimentAction.Edit, experiment.id, data);
    },
    [experiment.id, onComplete],
  );

  const handleMoveComplete = useCallback(() => {
    onComplete?.(ExperimentAction.Move, experiment.id);
  }, [experiment.id, onComplete]);

  const handleRetainLogsComplete = useCallback(() => {
    onComplete?.(ExperimentAction.RetainLogs, experiment.id);
  }, [experiment.id, onComplete]);

  const menuItems = getActionsForExperiment(experiment, dropdownActions, usePermissions())
    .filter((action) => action !== Action.SwitchPin)
    .map((action) => {
      return { danger: action === Action.Delete, key: action, label: action };
    });

  const cellCopyData = useMemo(() => {
    if (cell && 'displayData' in cell && isString(cell.displayData)) return cell.displayData;
    if (cell?.copyData) return cell.copyData;
    return undefined;
  }, [cell]);

  const dropdownMenu = useMemo(() => {
    const items: MenuItem[] = [];
    if (link) {
      items.push(
        { key: Action.NewTab, label: Action.NewTab },
        { key: Action.NewWindow, label: Action.NewWindow },
        { type: 'divider' },
      );
    }
    if (cellCopyData) {
      items.push({ key: Action.Copy, label: Action.Copy });
    }
    items.push(...menuItems);
    return items;
  }, [link, menuItems, cellCopyData]);

  const handleDropdown = useCallback(
    async (action: string, e: DropdownEvent) => {
      try {
        switch (action) {
          case Action.NewTab:
            handlePath(e, { path: link, popout: 'tab' });
            await onLink?.();
            break;
          case Action.NewWindow:
            handlePath(e, { path: link, popout: 'window' });
            await onLink?.();
            break;
          case Action.Activate:
            await activateExperiment({ experimentId: experiment.id });
            await onComplete?.(action, experiment.id);
            break;
          case Action.Archive:
            await archiveExperiment({ experimentId: experiment.id });
            await onComplete?.(action, experiment.id);
            break;
          case Action.Cancel:
            await cancelExperiment({ experimentId: experiment.id });
            await onComplete?.(action, experiment.id);
            break;
          case Action.OpenTensorBoard: {
            const commandResponse = await openOrCreateTensorBoard({
              experimentIds: [experiment.id],
              workspaceId: experiment.workspaceId,
            });
            openCommandResponse(commandResponse);
            break;
          }
          case Action.SwitchPin: {
            // TODO: leaving old code behind for when we want to enable this for our current experiment list.
            // const newPinned = { ...(settings?.pinned ?? {}) };
            // const pinSet = new Set(newPinned[experiment.projectId]);
            // if (pinSet.has(id)) {
            //   pinSet.delete(id);
            // } else {
            //   if (pinSet.size >= 5) {
            //     notification.warning({
            //       description: 'Up to 5 pinned items',
            //       message: 'Unable to pin this item',
            //     });
            //     break;
            //   }
            //   pinSet.add(id);
            // }
            // newPinned[experiment.projectId] = Array.from(pinSet);
            // updateSettings?.({ pinned: newPinned });
            // await onComplete?.(action, id);
            break;
          }
          case Action.Kill:
            confirm({
              content: `Are you sure you want to kill ${entityName} ${experiment.id}?`,
              danger: true,
              okText: 'Kill',
              onConfirm: async () => {
                await killExperiment({ experimentId: experiment.id });
                await onComplete?.(action, experiment.id);
              },
              onError: handleError,
              title: `Confirm ${capitalize(entityName)} Kill`,
            });
            break;
          case Action.Pause:
            await pauseExperiment({ experimentId: experiment.id });
            await onComplete?.(action, experiment.id);
            break;
          case Action.Unarchive:
            await unarchiveExperiment({ experimentId: experiment.id });
            await onComplete?.(action, experiment.id);
            break;
          case Action.Delete:
            confirm({
              content: `Are you sure you want to delete ${entityName} ${experiment.id}?`,
              danger: true,
              okText: 'Delete',
              onConfirm: async () => {
                await deleteExperiment({ experimentId: experiment.id });
                await onComplete?.(action, experiment.id);
              },
              onError: handleError,
              title: `Confirm ${capitalize(entityName)} Deletion`,
            });
            break;
          case Action.Edit:
            ExperimentEditModal.open();
            break;
          case Action.Move:
            ExperimentMoveModal.open();
            break;
          case Action.RetainLogs:
            ExperimentRetainLogsModal.open();
            break;
          case Action.HyperparameterSearch:
            interstitialModalOpen();
            fetchedExperimentItem();
            break;
          case Action.Copy:
            await copyToClipboard(cellCopyData ?? '');
            openToast({
              severity: 'Confirm',
              title: 'Value has been copied to clipboard.',
            });
            break;
        }
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: `Unable to ${action} ${entityName} ${experiment.id}.`,
          publicSubject: `${capitalize(action)} failed.`,
          silent: false,
          type: ErrorType.Server,
        });
      } finally {
        onVisibleChange?.(false);
      }
    },
    [
      entityName,
      link,
      onLink,
      experiment.id,
      onComplete,
      confirm,
      ExperimentEditModal,
      ExperimentMoveModal,
      ExperimentRetainLogsModal,
      interstitialModalOpen,
      fetchedExperimentItem,
      cellCopyData,
      openToast,
      experiment.workspaceId,
      onVisibleChange,
    ],
  );

  if (dropdownMenu.length === 0) {
    return (
      (children as JSX.Element) ?? (
        <div className={css.base} title="No actions available">
          <Button disabled type="text">
            <Icon name="overflow-vertical" title="Disabled action menu" />
          </Button>
        </div>
      )
    );
  }

  const shared = (
    <>
      <ExperimentEditModal.Component
        description={experiment.description ?? ''}
        experimentId={experiment.id}
        experimentName={experiment.name}
        onEditComplete={handleEditComplete}
      />
      <ExperimentMoveModal.Component
        experimentIds={[experiment.id]}
        sourceProjectId={experiment.projectId}
        sourceWorkspaceId={experiment.workspaceId}
        onSubmit={handleMoveComplete}
      />
      <ExperimentRetainLogsModal.Component
        experimentIds={[experiment.id]}
        projectId={experiment.projectId}
        onSubmit={handleRetainLogsComplete}
      />
      {experimentItem.isLoaded && (
        <HyperparameterSearchModal
          closeModal={hyperparameterSearchModalClose}
          experiment={experimentItem.data}
        />
      )}
      <InterstitialModal loadableData={experimentItem} onCloseAction={onInterstitalClose} />
    </>
  );

  return children ? (
    <>
      <Dropdown
        isContextMenu={isContextMenu}
        menu={dropdownMenu}
        open={makeOpen}
        onClick={handleDropdown}>
        {children}
      </Dropdown>
      {shared}
    </>
  ) : (
    <div className={css.base} title="Open actions menu">
      <Dropdown menu={dropdownMenu} placement="bottomRight" onClick={handleDropdown}>
        <Button icon={<Icon name="overflow-vertical" size="small" title="Action menu" />} />
      </Dropdown>
      {shared}
    </div>
  );
};

export default ExperimentActionDropdown;
