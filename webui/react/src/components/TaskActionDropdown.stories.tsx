import React from 'react';

import { CommandState, CommandType, RunState } from 'types';
import { generateCommandTask, generateExperimentTask } from 'utils/task';

import TaskActionDropdown from './TaskActionDropdown';

export default {
  component: TaskActionDropdown,
  title: 'TaskActionDropdown',
};

export const ExperimentActive = (): React.ReactNode => (
  <TaskActionDropdown
    task={{
      ...generateExperimentTask(0),
      state: RunState.Active,
    }}
  />
);

export const NoActions = (): React.ReactNode => (
  <TaskActionDropdown
    task={{
      ...generateCommandTask(0),
      state: CommandState.Terminated,
      type: CommandType.Shell,
    }}
  />
);
