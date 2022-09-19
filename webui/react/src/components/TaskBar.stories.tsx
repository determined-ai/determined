import React from 'react';

import { CommandType } from 'types';

import TaskBar from './TaskBar';

export default {
  component: TaskBar,
  title: 'Task Bar',
};

export const Default = (): React.ReactNode => (
  <TaskBar
    handleViewLogsClick={() => { return; }}
    id="task id"
    name="task name"
    resourcePool="task-resource-pool"
    type={CommandType.JupyterLab}
  />
);
