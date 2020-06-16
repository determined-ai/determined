import React from 'react';

import { ExperimentsDecorator } from 'storybook/ConetextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';
import { Task, TaskType } from 'types';
import { generateTasks } from 'utils/task';

import TaskTable from './TaskTable';

export default {
  component: TaskTable,
  decorators: [ RouterDecorator, ExperimentsDecorator ],
  title: 'TaskTable',
};

const tasks: Task[] = generateTasks(20).filter(task => task.type !== TaskType.Experiment);

export const Default = (): React.ReactNode => {
  return <TaskTable tasks={tasks} />;
};

export const Loading = (): React.ReactNode => {
  return <TaskTable />;
};

export const LoadedNoRows = (): React.ReactNode => {
  return <TaskTable tasks={[]} />;
};
