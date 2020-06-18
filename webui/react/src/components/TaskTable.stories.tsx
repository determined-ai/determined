import React from 'react';

import { ExperimentsDecorator } from 'storybook/ConetextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';
import { CommandTask } from 'types';
import { generateCommandTask } from 'utils/task';

import TaskTable from './TaskTable';

export default {
  component: TaskTable,
  decorators: [ RouterDecorator, ExperimentsDecorator ],
  title: 'TaskTable',
};

const tasks: CommandTask[] = new Array(20).fill(0).map((_, idx) => generateCommandTask(idx));

export const Default = (): React.ReactNode => {
  return <TaskTable tasks={tasks} />;
};

export const Loading = (): React.ReactNode => {
  return <TaskTable />;
};

export const LoadedNoRows = (): React.ReactNode => {
  return <TaskTable tasks={[]} />;
};
