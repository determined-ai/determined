import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Menu, MenuProps } from 'antd';
import { MenuInfo } from 'rc-menu/lib/interface';
import React, { MutableRefObject, useCallback, useEffect, useRef } from 'react';

import useModalExperimentMove from 'hooks/useModal/Experiment/useModalExperimentMove';
import useModalHyperparameterSearch from 'hooks/useModal/HyperparameterSearch/useModalHyperparameterSearch';
import usePermissions from 'hooks/usePermissions';
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
import { ErrorLevel, ErrorType } from 'shared/utils/error';
import { capitalize } from 'shared/utils/string';
import { ExperimentAction as Action, ProjectExperiment } from 'types';
import { modal } from 'utils/dialogApi';
import handleError from 'utils/error';
import { getActionsForExperiment } from 'utils/experiment';
import { openCommandResponse } from 'utils/wait';

const dropdownActions = [
  // Action.SwitchPin, requires settings
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

// eslint-disable-next-line
function useOutsideClickHandler(ref: MutableRefObject<any>, handler: () => void) {
  useEffect(() => {
    /**
     * Alert if clicked on outside of element
     */
    function handleClickOutside(event: Event) {
      if (ref.current && !ref.current.contains(event.target)) {
        handler();
      }
    }
    // Bind the event listener
    document.addEventListener('mouseup', handleClickOutside);
    return () => {
      // Unbind the event listener on clean up
      document.removeEventListener('mouseup', handleClickOutside);
    };
  }, [ref, handler]);
}

export interface TableContextMenuProps extends MenuProps {
  fetchExperiments: () => void;
  open: boolean;
  experiment: ProjectExperiment;
  handleClose: () => void;
  x: number;
  y: number;
}

export const TableContextMenu: React.FC<TableContextMenuProps> = ({
  experiment,
  fetchExperiments,
  handleClose,
  open,
  x,
  y,
}) => {
  const containerRef = useRef(null);
  useOutsideClickHandler(containerRef, handleClose);

  const permissions = usePermissions();

  const onComplete = useCallback(() => {
    handleClose();
    fetchExperiments();
  }, [fetchExperiments, handleClose]);

  const menuItems = experiment
    ? getActionsForExperiment(experiment, dropdownActions, permissions).map((action) => ({
        danger: action === Action.Delete,
        key: action,
        label: action,
      }))
    : [];

  const { contextHolder: modalExperimentMoveContextHolder, modalOpen: openExperimentMove } =
    useModalExperimentMove({ onClose: onComplete });
  const {
    contextHolder: modalHyperparameterSearchContextHolder,
    modalOpen: openModalHyperparameterSearch,
  } = useModalHyperparameterSearch({ experiment, onClose: onComplete });

  const handleExperimentMove = useCallback(() => {
    openExperimentMove({
      experimentIds: [experiment.id],
      sourceProjectId: experiment.projectId,
      sourceWorkspaceId: experiment.workspaceId,
    });
  }, [openExperimentMove, experiment]);

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
            await activateExperiment({ experimentId: experiment.id });
            onComplete();
            break;
          case Action.Archive:
            await archiveExperiment({ experimentId: experiment.id });
            onComplete();
            break;
          case Action.Cancel:
            await cancelExperiment({ experimentId: experiment.id });
            onComplete();
            break;
          case Action.OpenTensorBoard: {
            const commandResponse = await openOrCreateTensorBoard({
              experimentIds: [experiment.id],
              workspaceId: experiment.workspaceId,
            });
            openCommandResponse(commandResponse);
            break;
          }
          case Action.Kill:
            modal.confirm({
              content: `
              Are you sure you want to kill
              experiment ${experiment.id}?
            `,
              icon: <ExclamationCircleOutlined />,
              okText: 'Kill',
              onOk: async () => {
                await killExperiment({ experimentId: experiment.id });
                onComplete();
              },
              title: 'Confirm Experiment Kill',
            });
            break;
          case Action.Pause:
            await pauseExperiment({ experimentId: experiment.id });
            onComplete();
            break;
          case Action.Unarchive:
            await unarchiveExperiment({ experimentId: experiment.id });
            onComplete();
            break;
          case Action.Delete:
            modal.confirm({
              content: `
            Are you sure you want to delete
            experiment ${experiment.id}?
          `,
              icon: <ExclamationCircleOutlined />,
              okText: 'Delete',
              onOk: async () => {
                await deleteExperiment({ experimentId: experiment.id });
                onComplete();
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
      }
    },
    [
      experiment.id,
      experiment.workspaceId,
      handleExperimentMove,
      handleHyperparameterSearch,
      onComplete,
    ],
  );

  return (
    <div
      ref={containerRef}
      style={{
        border: 'solid 1px gold',
        display: !open ? 'none' : undefined,
        left: x,
        position: 'fixed',
        top: y,
        width: 200,
      }}>
      <Menu items={menuItems} onClick={handleMenuClick} />
      {modalExperimentMoveContextHolder}
      {modalHyperparameterSearchContextHolder}
    </div>
  );
};
