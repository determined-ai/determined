import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Dropdown, Menu, Modal } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React, { useCallback } from 'react';

import useModalExperimentMove from 'hooks/useModal/Experiment/useModalExperimentMove';
import useModalHyperparameterSearch
  from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import {
  activateExperiment, archiveExperiment, cancelExperiment, deleteExperiment, killExperiment,
  openOrCreateTensorBoard, pauseExperiment, unarchiveExperiment,
} from 'services/api';
import css from 'shared/components/ActionDropdown/ActionDropdown.module.scss';
import Icon from 'shared/components/Icon/Icon';
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { capitalize } from 'shared/utils/string';
import {
  ExperimentAction as Action, DetailedUser, ProjectExperiment,
} from 'types';
import handleError from 'utils/error';
import {
  getActionsForExperiment,
} from 'utils/experiment';
import { openCommand } from 'utils/wait';

interface Props {
  children?: React.ReactNode;
  curUser?: DetailedUser;
  experiment: ProjectExperiment;
  onComplete?: (action?: Action) => void;
  onVisibleChange?: (visible: boolean) => void;
  workspaceId?: number;
}

const dropdownActions = [
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
  onComplete,
  curUser,
  onVisibleChange,
  children,
}: Props) => {
  const id = experiment.id;
  const {
    contextHolder: modalExperimentMoveContextHolder,
    modalOpen: openExperimentMove,
  } = useModalExperimentMove({ onClose: onComplete });
  const {
    contextHolder: modalHyperparameterSearchContextHolder,
    modalOpen: openModalHyperparameterSearch,
  } = useModalHyperparameterSearch({ experiment, onClose: onComplete });

  const handleExperimentMove = useCallback(() => {
    openExperimentMove({
      experimentIds: experiment.id ? [ experiment.id ] : undefined,
      sourceProjectId: experiment.projectId,
      sourceWorkspaceId: experiment.workspaceId,
    });
  }, [ openExperimentMove, experiment.id, experiment.projectId, experiment.workspaceId ]);

  const handleHyperparameterSearch = useCallback(() => {
    openModalHyperparameterSearch();
  }, [ openModalHyperparameterSearch ]);

  const handleMenuClick = async (params: MenuInfo): Promise<void> => {
    params.domEvent.stopPropagation();
    try {
      const action = params.key as Action;
      switch (action) { // Cases should match menu items.
        case Action.Activate:
          await activateExperiment({ experimentId: id });
          if (onComplete) onComplete(action);
          break;
        case Action.Archive:
          await archiveExperiment({ experimentId: id });
          if (onComplete) onComplete(action);
          break;
        case Action.Cancel:
          await cancelExperiment({ experimentId: id });
          if (onComplete) onComplete(action);
          break;
        case Action.OpenTensorBoard: {
          const tensorboard = await openOrCreateTensorBoard({ experimentIds: [ id ] });
          openCommand(tensorboard);
          break;
        }
        case Action.Kill:
          Modal.confirm({
            content: `
              Are you sure you want to kill
              experiment ${id}?
            `,
            icon: <ExclamationCircleOutlined />,
            okText: 'Kill',
            onOk: async () => {
              await killExperiment({ experimentId: id });
              onComplete?.(action);
            },
            title: 'Confirm Experiment Kill',
          });
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
          Modal.confirm({
            content: `
            Are you sure you want to delete
            experiment ${id}?
          `,
            icon: <ExclamationCircleOutlined />,
            okText: 'Delete',
            onOk: async () => {
              await deleteExperiment({ experimentId: id });
              if (onComplete) onComplete(action);
            },
            title: 'Confirm Experiment Deletion',
          });
          break;
        case Action.Move:
          handleExperimentMove();
          break;
        case Action.HyperparameterSearch:
          handleHyperparameterSearch();
          break;
      }
    } catch (e) {
      handleError(e, {
        level: ErrorLevel.Error,
        publicMessage: `Unable to ${params.key} experiment ${experiment.id}.`,
        publicSubject: `${capitalize(params.key.toString())} failed.`,
        silent: false,
        type: ErrorType.Server,
      });
    } finally {
      onVisibleChange?.(false);
    }
    // TODO show loading indicator when we have a button component that supports it.
  };

  const menuItems = getActionsForExperiment(experiment, dropdownActions, curUser).map((action) => (
    { danger: action === Action.Delete, key: action, label: action }
  ));

  if (menuItems.length === 0) {
    return (children as JSX.Element) ?? (
      <div className={css.base} title="No actions available" onClick={stopPropagation}>
        <button disabled>
          <Icon name="overflow-vertical" />
        </button>
      </div>
    );
  }

  const menu = <Menu items={menuItems} onClick={handleMenuClick} />;

  return children ? (
    <>
      <Dropdown
        overlay={menu}
        placement="bottomLeft"
        trigger={[ 'contextMenu' ]}
        onVisibleChange={onVisibleChange}>
        {children}
      </Dropdown>
      {modalExperimentMoveContextHolder}
      {modalHyperparameterSearchContextHolder}
    </>
  ) : (
    <div className={css.base} title="Open actions menu" onClick={stopPropagation}>
      <Dropdown overlay={menu} placement="bottomRight" trigger={[ 'click' ]}>
        <button onClick={stopPropagation}>
          <Icon name="overflow-vertical" />
        </button>
      </Dropdown>
      {modalExperimentMoveContextHolder}
      {modalHyperparameterSearchContextHolder}
    </div>
  );
};

export default ExperimentActionDropdown;
