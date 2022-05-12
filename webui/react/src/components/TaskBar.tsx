import { Dropdown, Menu } from 'antd';
import React, {useCallback}  from 'react';
import { paths, routeToReactUrl } from 'routes/utils';
import { Modal } from 'antd';
import { CommandTask, CommandType } from 'types';
import { ExclamationCircleOutlined } from '@ant-design/icons';

import {
  killTask
} from 'services/api';

import Icon from './Icon';
import css from './TaskBar.module.scss';
interface Props{
  id: string;
  name: string;
  resourcePool: string;
  type: CommandType
}

export const TaskBar: React.FC<Props> = ({ id, name, resourcePool, type } : Props) => {

  const task = {id, name, resourcePool, type} as CommandTask;

  const deleteTask = useCallback((task: CommandTask) => {
    Modal.confirm({
      content: `
      Are you sure you want to kill
      this task?
    `,
      icon: <ExclamationCircleOutlined />,
      okText: 'Kill',
      onOk: async () => {
        console.log("killing task: ", task)
        await killTask(task);
        routeToReactUrl(paths.taskList())
      },
      title: 'Confirm Task Kill',
    });
  }, [task])

  const dropdownOptions = (
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
  );

  return (
    <div className={css.base}>
      <div className={css.barContent}>
        {name}  —  {resourcePool}
        <Dropdown
          overlay={dropdownOptions}
          placement="bottomRight"
          trigger={[ 'click' ]}>
          <span> <Icon name="arrow-down" size="tiny" /> </span>

        </Dropdown>
      </div>

    </div>
  );
};

export default TaskBar;
