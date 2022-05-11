import { string } from 'fp-ts';
import React from 'react';

import css from './TaskBar.module.scss';

interface Props{
  taskName: string;
  resourcePool: string
}

export const TaskBar: React.FC<Props> = ({taskName, resourcePool} : Props) => {
  return (
    <div className={css.base}>
      {taskName} —— {resourcePool} 
    </div>
  );
};

export default TaskBar;
