import React, { useCallback, useMemo } from 'react';

import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import useConfirm from 'components/kit/useConfirm';
import css from 'components/TaskBar.module.scss';
import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { killTask } from 'services/api';
import { CommandTask, CommandType } from 'types';
import handleError from 'utils/error';
import { routeToReactUrl } from 'utils/routes';

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
  const task = useMemo(() => {
    return { id, name, resourcePool, type } as CommandTask;
  }, [id, name, resourcePool, type]);

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
        disabled: !canModifyWorkspaceNSC({ workspace: { id: task.workspaceId } }),
        key: MenuKey.Kill,
        label: 'Kill',
      },
      { key: MenuKey.ViewLogs, label: 'View Logs' },
    ],
    [canModifyWorkspaceNSC, task.workspaceId],
  );

  const handleDropdown = (key: string) => {
    switch (key) {
      case MenuKey.Kill:
        deleteTask(task);
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
            <Icon name="arrow-down" size="tiny" title="Action menu" />
          </div>
        </Dropdown>
      </div>
    </div>
  );
};

export default TaskBar;
