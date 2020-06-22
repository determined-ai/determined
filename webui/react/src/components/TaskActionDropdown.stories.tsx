import React from 'react';

import { ExperimentsDecorator } from 'storybook/ConetextDecorators';
import { CommandState, CommandType, RunState } from 'types';
import { generateCommandTask, generateExperimentTasks } from 'utils/task';

import TaskActionDropdown from './TaskActionDropdown';

export default {
  component: TaskActionDropdown,
  decorators: [ ExperimentsDecorator ],
  title: 'TaskActionDropdown',
};

export const ExperimentActive = (): React.ReactNode => {
  return <TaskActionDropdown
    task={{
      ...generateExperimentTasks(0),
      state: RunState.Active,
    }} />;
};

export const NoActions = (): React.ReactNode => {
  return <TaskActionDropdown
    task={{
      ...generateCommandTask(0),
      state: CommandState.Terminated,
      type: CommandType.Shell,
    }} />;
};
