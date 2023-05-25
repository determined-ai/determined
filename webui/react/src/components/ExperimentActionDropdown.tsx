import React, { MouseEvent, useCallback, useMemo } from 'react';

import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import Button from 'components/kit/Button';
import Dropdown, { DropdownEvent, MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePermissions from 'hooks/usePermissions';
import { UpdateSettings } from 'hooks/useSettings';
import { ExperimentListSettings } from 'pages/ExperimentList.settings';
import { BatchAction } from 'pages/F_ExpList/glide-table/TableActionBar';
import { handlePath } from 'routes/utils';
import {
  activateExperiment,
  archiveExperiment,
  cancelExperiment,
  deleteExperiment,
  killExperiment,
  openOrCreateTensorBoard,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import { ValueOf } from 'shared/types';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { capitalize } from 'shared/utils/string';
import { ExperimentAction, ProjectExperiment } from 'types';
import { notification } from 'utils/dialogApi';
import handleError from 'utils/error';
import { getActionsForExperiment } from 'utils/experiment';
import { openCommandResponse } from 'utils/wait';

import useConfirm from './kit/useConfirm';

interface Props {
  children?: React.ReactNode;
  experiment: ProjectExperiment;
  isContextMenu?: boolean;
  link?: string;
  makeOpen?: boolean;
  onComplete?: (action?: Action) => void | Promise<void>;
  onLink?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  settings?: ExperimentListSettings;
  updateSettings?: UpdateSettings<ExperimentListSettings>;
  workspaceId?: number;
  handleUpdateExperimentList?: (action: BatchAction, successfulIds: number[]) => void;
}

const Action = {
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
  Action.Move,
  Action.OpenTensorBoard,
  Action.HyperparameterSearch,
  Action.Delete,
];

const ExperimentActionDropdown: React.FC<Props> = ({
  experiment,
  isContextMenu,
  link,
  makeOpen,
  onComplete,
  onLink,
  onVisibleChange,
  settings,
  updateSettings,
  children,
  handleUpdateExperimentList,
}: Props) => {
  const id = experiment.id;
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);
  const confirm = useConfirm();
  const {
    contextHolder: modalHyperparameterSearchContextHolder,
    modalOpen: openModalHyperparameterSearch,
  } = useModalHyperparameterSearch({ experiment, onClose: onComplete });

  const handleHyperparameterSearch = useCallback(() => {
    openModalHyperparameterSearch();
  }, [openModalHyperparameterSearch]);

  const handleMoveComplete = useCallback(() => {
    onComplete?.(Action.Move);
  }, [onComplete]);

  const menuItems = getActionsForExperiment(experiment, dropdownActions, usePermissions())
    .filter((action) => action !== Action.SwitchPin || settings)
    .map((action) => {
      if (action === Action.SwitchPin) {
        const label = (settings?.pinned?.[experiment.projectId] ?? []).includes(id)
          ? 'Unpin'
          : 'Pin';
        return { key: action, label };
      } else {
        return { danger: action === Action.Delete, key: action, label: action };
      }
    });

  const dropdownMenu = useMemo(() => {
    const items: MenuItem[] = [...menuItems];
    if (link) {
      items.unshift(
        { key: Action.NewTab, label: Action.NewTab },
        { key: Action.NewWindow, label: Action.NewWindow },
        { type: 'divider' },
      );
    }
    return items;
  }, [link, menuItems]);

  const handleDropdown = useCallback(
    async (action: string, e: DropdownEvent) => {
      try {
        switch (action) {
          case Action.NewTab:
            handlePath(e as MouseEvent, { path: link, popout: 'tab' });
            await onLink?.();
            break;
          case Action.NewWindow:
            handlePath(e as MouseEvent, { path: link, popout: 'window' });
            await onLink?.();
            break;
          case Action.Activate:
            await activateExperiment({ experimentId: id });
            await onComplete?.(action);
            handleUpdateExperimentList?.(Action.Activate, [id]);
            break;
          case Action.Archive:
            await archiveExperiment({ experimentId: id });
            await onComplete?.(action);
            handleUpdateExperimentList?.(Action.Archive, [id]);
            break;
          case Action.Cancel:
            await cancelExperiment({ experimentId: id });
            await onComplete?.(action);
            handleUpdateExperimentList?.(Action.Cancel, [id]);
            break;
          case Action.OpenTensorBoard: {
            const commandResponse = await openOrCreateTensorBoard({
              experimentIds: [id],
              workspaceId: experiment.workspaceId,
            });
            openCommandResponse(commandResponse);
            handleUpdateExperimentList?.(Action.OpenTensorBoard, [id]);
            break;
          }
          case Action.SwitchPin: {
            const newPinned = { ...(settings?.pinned ?? {}) };
            const pinSet = new Set(newPinned[experiment.projectId]);
            if (pinSet.has(id)) {
              pinSet.delete(id);
            } else {
              if (pinSet.size >= 5) {
                notification.warning({
                  description: 'Up to 5 pinned items',
                  message: 'Unable to pin this item',
                });
                break;
              }
              pinSet.add(id);
            }
            newPinned[experiment.projectId] = Array.from(pinSet);
            updateSettings?.({ pinned: newPinned });
            break;
          }
          case Action.Kill:
            confirm({
              content: `Are you sure you want to kill experiment ${id}?`,
              danger: true,
              okText: 'Kill',
              onConfirm: async () => {
                await killExperiment({ experimentId: id });
                await onComplete?.(action);
              },
              title: 'Confirm Experiment Kill',
            });
            handleUpdateExperimentList?.(Action.Kill, [id]);
            break;
          case Action.Pause:
            await pauseExperiment({ experimentId: id });
            await onComplete?.(action);
            handleUpdateExperimentList?.(Action.Pause, [id]);
            break;
          case Action.Unarchive:
            await unarchiveExperiment({ experimentId: id });
            await onComplete?.(action);
            handleUpdateExperimentList?.(Action.Unarchive, [id]);
            break;
          case Action.Delete:
            confirm({
              content: `Are you sure you want to delete experiment ${id}?`,
              danger: true,
              okText: 'Delete',
              onConfirm: async () => {
                await deleteExperiment({ experimentId: id });
                await onComplete?.(action);
              },
              title: 'Confirm Experiment Deletion',
            });
            handleUpdateExperimentList?.(Action.Delete, [id]);
            break;
          case Action.Move:
            ExperimentMoveModal.open();
            break;
          case Action.HyperparameterSearch:
            handleHyperparameterSearch();
            break;
        }
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: `Unable to ${action} experiment ${id}.`,
          publicSubject: `${capitalize(action)} failed.`,
          silent: false,
          type: ErrorType.Server,
        });
      } finally {
        onVisibleChange?.(false);
      }
    },
    [
      confirm,
      experiment.projectId,
      ExperimentMoveModal,
      experiment.workspaceId,
      handleHyperparameterSearch,
      id,
      link,
      onComplete,
      onLink,
      onVisibleChange,
      settings?.pinned,
      updateSettings,
      handleUpdateExperimentList,
    ],
  );

  if (menuItems.length === 0) {
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
      <ExperimentMoveModal.Component
        experimentIds={[id]}
        sourceProjectId={experiment.projectId}
        sourceWorkspaceId={experiment.workspaceId}
        onSubmit={handleMoveComplete}
      />
      {modalHyperparameterSearchContextHolder}
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
