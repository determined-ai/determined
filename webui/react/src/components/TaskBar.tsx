import { Button, Dropdown, Menu } from 'antd';
import React from 'react';

import Icon from './Icon';
import css from './TaskBar.module.scss';
interface Props{
  taskId: string;
  taskName: string;
  resourcePool: string;
}

export const TaskBar: React.FC<Props> = ({ taskId, taskName, resourcePool } : Props) => {

  const dropdownOptions = (
    <Menu>
      <Menu.Item
        key={1}
        onClick={() => {}}>
        Kill
      </Menu.Item>
      <Menu.Item
        key={2}
        onClick={() => {}}>
        View Logs
      </Menu.Item>
    </Menu>
  );

  return (
    <div className={css.base}>
      <div className={css.barContent}>
        {taskName}  —  {resourcePool}
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
