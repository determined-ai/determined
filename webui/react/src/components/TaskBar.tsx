import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import useConfirm from 'hew/useConfirm';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useMemo } from 'react';

import { useAsync } from 'hooks/useAsync';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { getCommand, getJupyterLab, getShell, getTensorBoard, killTask } from 'services/api';
import { CommandTask, CommandType } from 'types';
import handleError from 'utils/error';
import { routeToReactUrl } from 'utils/routes';

import css from './TaskBar.module.scss';

interface Props {
  handleViewLogsClick: () => void;
  id: string;
  name: string;
  resourcePool: string;
  type: CommandType;
}

const MenuKey = {
  Kill: 'kill',
  ViewLogs: 'viewLogs',
} as const;

export const TaskBar: React.FC<Props> = ({
  handleViewLogsClick,
  id,
  name,
  resourcePool,
  type,
}: Props) => {
  const { canModifyWorkspaceNSC } = usePermissions();
  const confirm = useConfirm();

  const task = useAsync(() => {
    const call = {
      'command': getCommand,
      'jupyter-lab': getJupyterLab,
      'shell': getShell,
      'tensor-board': getTensorBoard,
    }[type];
    return call({ commandId: id });
  }, [type, id]);

  const deleteTask = useCallback(
    (task: CommandTask) => {
      confirm({
        content: 'Are you sure you want to kill this task?',
        danger: true,
        okText: 'Kill',
        onConfirm: async () => {
          try {
            await killTask(task);
            routeToReactUrl(paths.taskList());
          } catch (e) {
            handleError(e, {
              publicMessage: `Unable to kill task ${task.id}.`,
              publicSubject: 'Kill failed.',
              silent: false,
            });
          }
        },
        onError: handleError,
        title: 'Confirm Task Kill',
      });
    },
    [confirm],
  );

  const menuItems: MenuItem[] = useMemo(
    () => [
      {
        disabled: Loadable.match(task, {
          _: () => true,
          Loaded: (t) => !canModifyWorkspaceNSC({ workspace: { id: t.workspaceId } }),
        }),
        key: MenuKey.Kill,
        label: 'Kill',
      },
      { key: MenuKey.ViewLogs, label: 'View Logs' },
    ],
    [canModifyWorkspaceNSC, task],
  );

  const handleDropdown = (key: string) => {
    switch (key) {
      case MenuKey.Kill:
        Loadable.forEach(task, deleteTask);
        break;
      case MenuKey.ViewLogs:
        handleViewLogsClick();
        break;
    }
  };

  return (
    <div className={css.base}>
      <div className={css.barContent}>
        <span>{name}</span>
        <span>&#8212;</span>
        <Dropdown menu={menuItems} placement="bottomRight" onClick={handleDropdown}>
          <div className={css.dropdownTrigger} data-testid="task-action-dropdown-trigger">
            <span className={css.dropdownTrigger}>{resourcePool}</span>
            <Icon name="arrow-down" title="Action menu" />
          </div>
        </Dropdown>
      </div>
    </div>
  );
};

export default TaskBar;
