import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import { CommandState, RunState } from 'types';

import ProgressBar from './ProgressBar';

export default {
  argTypes: {
    percent: { control: { max: 100, min: 0, step: 1, type: 'range' } },
    state: { control: { options: { ...RunState, ...CommandState }, type: 'select' } },
  },
  component: ProgressBar,
  title: 'Determined/Bars/ProgressBar',
} as Meta<typeof ProgressBar>;

export const Default: ComponentStory<typeof ProgressBar> = (args) => (
  <div style={{ width: 240 }}>
    <ProgressBar {...args} />
  </div>
);

Default.args = {
  percent: 50,
  state: RunState.Active,
};
