import { Dropdown } from 'antd';
import type { DropdownProps } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React, { useCallback, useMemo } from 'react';

import ExperimentMoveModalComponent from 'components/ExperimentMoveModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePermissions from 'hooks/usePermissions';
import { UpdateSettings } from 'hooks/useSettings';
import { ExperimentListSettings } from 'pages/ExperimentList.settings';
import {
  activateExperiment,
  archiveExperiment,
  openOrCreateTensorBoard,
  pauseExperiment,
  unarchiveExperiment,
} from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { capitalize } from 'shared/utils/string';
import { ExperimentAction as Action, ProjectExperiment } from 'types';
import { notification } from 'utils/dialogApi';
import handleError from 'utils/error';
import { getActionsForExperiment } from 'utils/experiment';
import { openCommandResponse } from 'utils/wait';

import ExperimentDeleteModalComponent from './ExperimentDeleteModal';
import ExperimentStopModalComponent from './ExperimentStopModal';

interface Props {
  children?: React.ReactNode;
  experiment: ProjectExperiment;
  makeOpen?: boolean;
  onComplete?: (action?: Action) => void;
  onVisibleChange?: (visible: boolean) => void;
  settings?: ExperimentListSettings;
  updateSettings?: UpdateSettings;
  workspaceId?: number;
}

const dropdownActions = [
  Action.SwitchPin,
  Action.Activate,
  Action.Pause,
  Action.Archive,
  Action.Unarchive,
  Action.Stop,
  Action.Move,
  Action.OpenTensorBoard,
  Action.HyperparameterSearch,
  Action.Delete,
];

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const ExperimentActionDropdown: React.FC<Props> = ({
  experiment,
  makeOpen,
  onComplete,
  onVisibleChange,
  settings,
  updateSettings,
  children,
}: Props) => {
  const id = experiment.id;
  const ExperimentDeleteModal = useModal(ExperimentDeleteModalComponent);
  const ExperimentMoveModal = useModal(ExperimentMoveModalComponent);
  const ExperimentStopModal = useModal(ExperimentStopModalComponent);
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
          case Action.Activate:
            await activateExperiment({ experimentId: id });
            if (onComplete) onComplete(action);
            break;
          case Action.Archive:
            await archiveExperiment({ experimentId: id });
            if (onComplete) onComplete(action);
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
          case Action.Stop:
            ExperimentStopModal.open();
            break;
          case Action.Pause:
            await pauseExperiment({ experimentId: id });
            if (onComplete) onComplete(action);
            break;
          case Action.Unarchive:
            await unarchiveExperiment({ experimentId: id });
            if (onComplete) onComplete(action);
            break;
          case Action.Delete:
            ExperimentDeleteModal.open();
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
      experiment.projectId,
      ExperimentDeleteModal,
      ExperimentMoveModal,
      ExperimentStopModal,
      experiment.workspaceId,
      handleHyperparameterSearch,
      id,
      onComplete,
      onVisibleChange,
      settings?.pinned,
      updateSettings,
    ],
  );

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
    return { items: [...menuItems], onClick: handleMenuClick };
  }, [menuItems, handleMenuClick]);

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
      <ExperimentDeleteModal.Component experiment={experiment} />
      <ExperimentMoveModal.Component
        experimentIds={[id]}
        sourceProjectId={experiment.projectId}
        sourceWorkspaceId={experiment.workspaceId}
        onClose={onComplete}
      />
      <ExperimentStopModal.Component experimentId={id} />
      {modalHyperparameterSearchContextHolder}
    </>
  ) : (
    <div className={css.base} title="Open actions menu" onClick={stopPropagation}>
      <Dropdown menu={menu} placement="bottomRight" trigger={['click']}>
        <Button ghost icon={<Icon name="overflow-vertical" />} onClick={stopPropagation} />
      </Dropdown>
      <ExperimentDeleteModal.Component experiment={experiment} />
      <ExperimentMoveModal.Component
        experimentIds={[id]}
        sourceProjectId={experiment.projectId}
        sourceWorkspaceId={experiment.workspaceId}
        onClose={onComplete}
      />
      <ExperimentStopModal.Component experimentId={id} />
      {modalHyperparameterSearchContextHolder}
    </div>
  );
};

export default ExperimentActionDropdown;
