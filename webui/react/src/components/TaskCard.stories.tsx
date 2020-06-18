import React from 'react';

import Grid from 'components/Grid';
import { ExperimentsDecorator } from 'storybook/ConetextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';
import { ShirtSize } from 'themes';
import { generateCommandTask, generateExperimentTasks, generateTasks } from 'utils/task';

import TaskCard from './TaskCard';

export default {
  component: TaskCard,
  decorators: [ RouterDecorator, ExperimentsDecorator ],
  title: 'TaskCard',
};

export const DefaultExperiment = (): React.ReactNode => {
  return <TaskCard {...generateExperimentTasks(0)} />;
};

export const DefaultCommand = (): React.ReactNode => {
  return <TaskCard {...generateCommandTask(0)} />;
};

export const InAGrid = (): React.ReactNode => {
  const tasks: React.ReactNodeArray =
    generateTasks().map((task, idx) => {
      return <TaskCard key={idx} {...task} />;
    });
  return (
    <Grid gap={ShirtSize.large} minItemWidth={20}>{tasks}</Grid>
  );
};
