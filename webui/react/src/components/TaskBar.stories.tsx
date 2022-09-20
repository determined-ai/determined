import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import { CommandType } from 'types';

import TaskBar from './TaskBar';

export default {
  component: TaskBar,
  title: 'Determined/Bars/Task Bar',
} as Meta<typeof TaskBar>;

export const Default: ComponentStory<typeof TaskBar> = (args) => (
  <TaskBar
    {...args}
    handleViewLogsClick={() => {
      return;
    }}
    id="task id"
    name="task name"
    resourcePool="task-resource-pool"
  />
);

Default.args = { type: CommandType.JupyterLab };
