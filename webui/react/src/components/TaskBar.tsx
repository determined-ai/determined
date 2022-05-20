import { ExclamationCircleOutlined } from '@ant-design/icons';
import { Dropdown, Menu } from 'antd';
import { Modal } from 'antd';
import React, { useCallback, useMemo } from 'react';

import { paths } from 'routes/utils';
import {
  killTask,
} from 'services/api';
import { routeToReactUrl } from 'shared/utils/routes';
import { CommandTask, CommandType } from 'types';

import Icon from '../shared/components/Icon/Icon';

import css from './TaskBar.module.scss';
interface Props{
  id: string;
  name: string;
  resourcePool: string;
  type: CommandType
}

export const TaskBar: React.FC<Props> = ({ id, name, resourcePool, type } : Props) => {

  const task = useMemo(() => {
    const commandTask = { id, name, resourcePool, type } as CommandTask;
    return commandTask;
  }, [ id, name, resourcePool, type ]);

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

  const dropdownOverlay = useMemo(() => (
    <Menu>
      <Menu.Item
        key="kill"
        onClick={() => deleteTask(task)}>
        Kill
      </Menu.Item>
      <Menu.Item
        key="viewLogs"
        onClick={() => routeToReactUrl(paths.taskLogs(task))}>
        View Logs
      </Menu.Item>
    </Menu>
  ), [ task, deleteTask ]);

  return (
    <div className={css.base}>
      <div className={css.barContent}>
        <span>{name}</span>
        <span>&#8212;</span>
        <Dropdown
          overlay={dropdownOverlay}
          placement="bottomRight"
          trigger={[ 'click' ]}>
          <div
            className={css.dropdownTrigger}
            data-testid="task-action-dropdown-trigger">
            <span className={css.dropdownTrigger}>{resourcePool}</span>
            <Icon name="arrow-down" size="tiny" />
          </div>
        </Dropdown>
      </div>
    </div>
  );
};

export default TaskBar;
