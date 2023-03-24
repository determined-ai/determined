import { ExclamationCircleOutlined } from '@ant-design/icons';
import type { DropDownProps, MenuProps } from 'antd';
import { Dropdown } from 'antd';
import React, { useCallback, useMemo } from 'react';

import usePermissions from 'hooks/usePermissions';
import { paths } from 'routes/utils';
import { killTask } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { routeToReactUrl } from 'shared/utils/routes';
import { CommandTask, CommandType } from 'types';
import { modal } from 'utils/dialogApi';
import handleError from 'utils/error';

import css from './TaskBar.module.scss';
interface Props {
  handleViewLogsClick: () => void;
  id: string;
  name: string;
  resourcePool: string;
  type: CommandType;
}

export const TaskBar: React.FC<Props> = ({
  handleViewLogsClick,
  id,
  name,
  resourcePool,
  type,
}: Props) => {
  const { canModifyWorkspaceNSC } = usePermissions();
  const task = useMemo(() => {
    const commandTask = { id, name, resourcePool, type } as CommandTask;
    return commandTask;
  }, [id, name, resourcePool, type]);

  const deleteTask = useCallback((task: CommandTask) => {
    modal.confirm({
      content: `
      Are you sure you want to kill
      this task?
    `,
      icon: <ExclamationCircleOutlined />,
      okText: 'Kill',
      onOk: async () => {
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
      title: 'Confirm Task Kill',
    });
  }, []);

  const dropdownOverlay: DropDownProps['menu'] = useMemo(() => {
    const MenuKey = {
      Kill: 'kill',
      ViewLogs: 'viewLogs',
    } as const;

    const funcs = {
      [MenuKey.Kill]: () => {
        deleteTask(task);
      },
      [MenuKey.ViewLogs]: () => {
        handleViewLogsClick();
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as ValueOf<typeof MenuKey>]();
    };

    const menuItems: MenuProps['items'] = [
      {
        disabled: !canModifyWorkspaceNSC({ workspace: { id: task.workspaceId } }),
        key: MenuKey.Kill,
        label: 'Kill',
      },
      { key: MenuKey.ViewLogs, label: 'View Logs' },
    ];

    return { items: menuItems, onClick: onItemClick };
  }, [task, deleteTask, handleViewLogsClick, canModifyWorkspaceNSC]);

  return (
    <div className={css.base}>
      <div className={css.barContent}>
        <span>{name}</span>
        <span>&#8212;</span>
        <Dropdown menu={dropdownOverlay} placement="bottomRight" trigger={['click']}>
          <div className={css.dropdownTrigger} data-testid="task-action-dropdown-trigger">
            <span className={css.dropdownTrigger}>{resourcePool}</span>
            <Icon name="arrow-down" size="tiny" />
          </div>
        </Dropdown>
      </div>
    </div>
  );
};

export default TaskBar;
