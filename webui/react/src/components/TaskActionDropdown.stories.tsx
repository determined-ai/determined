import React from 'react';

import { ExperimentsDecorator } from 'storybook/ContextDecorators';
import { CommandState, CommandType, RunState } from 'types';
import { generateCommandTask, generateExperimentTask } from 'utils/task';

import TaskActionDropdown from './TaskActionDropdown';

export default {
  component: TaskActionDropdown,
  decorators: [ ExperimentsDecorator ],
  title: 'TaskActionDropdown',
};

export const ExperimentActive = (): React.ReactNode => {
  return <TaskActionDropdown
    task={{
      ...generateExperimentTask(0),
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
