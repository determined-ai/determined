import React from 'react';

import { CommandType } from 'types';

import TaskBar from './TaskBar';

export default {
  component: TaskBar,
  title: 'Task Bar',
};

export const Default = (): React.ReactNode => (
  <TaskBar
    // eslint-disable-next-line   @typescript-eslint/no-empty-function
    handleViewLogsClick={() => {}}
    id="task id"
    name="task name"
    resourcePool="task-resource-pool"
    type={CommandType.JupyterLab}
  />
);
