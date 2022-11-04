import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Dropdown, Menu } from 'antd';
import type { MenuProps } from 'antd';
import { Modal } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { paths } from 'routes/utils';
import { killTask } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import { ValueOf } from 'shared/types';
import { routeToReactUrl } from 'shared/utils/routes';
import { CommandTask, CommandType } from 'types';

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
  const task = useMemo(() => {
    const commandTask = { id, name, resourcePool, type } as CommandTask;
    return commandTask;
  }, [id, name, resourcePool, type]);

  const deleteTask = useCallback((task: CommandTask) => {
    Modal.confirm({
      content: `
      Are you sure you want to kill
      this task?
    `,
      icon: <ExclamationCircleOutlined />,
      okText: 'Kill',
      onOk: async () => {
        await killTask(task);
        routeToReactUrl(paths.taskList());
      },
      title: 'Confirm Task Kill',
    });
  }, []);

  const dropdownOverlay = useMemo(() => {
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
      { key: MenuKey.Kill, label: 'Kill' },
      { key: MenuKey.ViewLogs, label: 'View Logs' },
    ];

    return <Menu items={menuItems} onClick={onItemClick} />;
  }, [task, deleteTask, handleViewLogsClick]);

  return (
    <div className={css.base}>
      <div className={css.barContent}>
        <span>{name}</span>
        <span>&#8212;</span>
        <Dropdown overlay={dropdownOverlay} placement="bottomRight" trigger={['click']}>
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
