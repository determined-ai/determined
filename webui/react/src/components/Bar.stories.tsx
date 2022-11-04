import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import { ShirtSize } from 'themes';

import Bar from './Bar';

export default {
  argTypes: { size: { control: 'inline-radio' } },
  component: Bar,
  title: 'Determined/Bars/Bar',
} as Meta<typeof Bar>;

export const Default: ComponentStory<typeof Bar> = (args) => (
  <div style={{ width: 240 }}>
    <Bar
      {...args}
      parts={[
        { color: 'red', label: 'labelA', percent: 0.3 },
        { color: 'blue', label: 'labelB', percent: 0.2 },
        { color: 'yellow', label: 'labelC', percent: 0.5 },
      ]}
    />
  </div>
);

Default.args = {
  barOnly: false,
  inline: false,
  size: ShirtSize.Small,
};
