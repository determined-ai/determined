import React from 'react';

import { ExperimentsDecorator } from 'storybook/ConetextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';
import { Experiment, ExperimentTask } from 'types';
import { generateExperimentTask } from 'utils/task';

import ExperimentsTable from './ExperimentTable';

export default {
  component: ExperimentsTable,
  decorators: [ RouterDecorator, ExperimentsDecorator ],
  title: 'ExperimentsTable',
};

const experimentTasks: ExperimentTask[] = new Array(30).fill(0)
  .map((_, idx) => generateExperimentTask(idx));

export const Default = (): React.ReactNode => {
  // FIXME the conversion
  return <ExperimentsTable experiments={experimentTasks as unknown as Experiment[]} />;
};
