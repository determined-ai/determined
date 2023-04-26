import { Dropdown } from 'antd';
import type { DropdownProps } from 'antd';
import { ItemType } from 'antd/es/menu/hooks/useItems';
import { MenuInfo } from 'rc-menu/lib/interface';
import React, { MouseEvent, useCallback, useMemo } from 'react';

import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePermissions from 'hooks/usePermissions';
import { UpdateSettings } from 'hooks/useSettings';
import { ExperimentListSettings } from 'pages/ExperimentList.settings';
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
import Icon from 'shared/components/Icon/Icon';
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
  link?: string;
  makeOpen?: boolean;
  onComplete?: (action?: Action) => Promise<void>;
  onLink?: () => void;
  onVisibleChange?: (visible: boolean) => void;
  settings?: ExperimentListSettings;
  updateSettings?: UpdateSettings;
  workspaceId?: number;
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

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ExperimentActionDropdown: React.FC<Props> = ({
  experiment,
  link,
  makeOpen,
  onComplete,
  onLink,
  onVisibleChange,
  settings,
  updateSettings,
  children,
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

  const handleMenuClick = useCallback(
    async (params: MenuInfo): Promise<void> => {
      params.domEvent.stopPropagation();
      try {
        const action = params.key as Action;
        switch (
          action // Cases should match menu items.
        ) {
          case Action.NewTab:
            handlePath(params.domEvent as MouseEvent, { path: link, popout: 'tab' });
            await onLink?.();
            break;
          case Action.NewWindow:
            handlePath(params.domEvent as MouseEvent, { path: link, popout: 'window' });
            await onLink?.();
            break;
          case Action.Activate:
            await activateExperiment({ experimentId: id });
            await onComplete?.(action);
            break;
          case Action.Archive:
            await archiveExperiment({ experimentId: id });
            await onComplete?.(action);
            break;
          case Action.Cancel:
            await cancelExperiment({ experimentId: id });
            await onComplete?.(action);
            break;
          case Action.OpenTensorBoard: {
            const commandResponse = await openOrCreateTensorBoard({
              experimentIds: [id],
              workspaceId: experiment.workspaceId,
            });
            openCommandResponse(commandResponse);
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
            break;
          case Action.Pause:
            await pauseExperiment({ experimentId: id });
            await onComplete?.(action);
            break;
          case Action.Unarchive:
            await unarchiveExperiment({ experimentId: id });
            await onComplete?.(action);
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
          publicMessage: `Unable to ${params.key} experiment ${id}.`,
          publicSubject: `${capitalize(params.key.toString())} failed.`,
          silent: false,
          type: ErrorType.Server,
        });
      } finally {
        onVisibleChange?.(false);
      }
      // TODO show loading indicator when we have a button component that supports it.
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
    ],
  );

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

  const menu: DropdownProps['menu'] = useMemo(() => {
    const items: ItemType[] = [...menuItems];
    if (link) {
      items.unshift(
        { key: Action.NewTab, label: Action.NewTab },
        { key: Action.NewWindow, label: Action.NewWindow },
        { type: 'divider' },
      );
    }
    return { items, onClick: handleMenuClick };
  }, [link, menuItems, handleMenuClick]);

  if (menuItems.length === 0) {
    return (
      (children as JSX.Element) ?? (
        <div className={css.base} title="No actions available" onClick={stopPropagation}>
          <Button disabled ghost type="text">
            <Icon name="overflow-vertical" />
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
        menu={menu}
        open={makeOpen}
        placement="bottomLeft"
        trigger={['contextMenu']}
        onOpenChange={onVisibleChange}>
        {children}
      </Dropdown>
      {shared}
    </>
  ) : (
    <div className={css.base} title="Open actions menu" onClick={stopPropagation}>
      <Dropdown menu={menu} placement="bottomRight" trigger={['click']}>
        <Button ghost icon={<Icon name="overflow-vertical" />} onClick={stopPropagation} />
      </Dropdown>
      {shared}
    </div>
  );
};

export default ExperimentActionDropdown;
