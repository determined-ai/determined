import React from 'react';

import { ExperimentsDecorator } from 'storybook/ConetextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';
import { Task } from 'types';
import { generateTasks } from 'utils/task';

import TasksTable from './TasksTable';

export default {
  component: TasksTable,
  decorators: [ RouterDecorator, ExperimentsDecorator ],
  title: 'TasksTable',
};

const tasks: Task[] = generateTasks();

export const Default = (): React.ReactNode => {
  return <TasksTable tasks={tasks} />;
};
