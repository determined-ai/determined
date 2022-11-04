import { ComponentStory, Meta } from '@storybook/react';
import { Slider } from 'antd';
import { SliderSingleProps } from 'antd/lib/slider';
import React from 'react';

export default {
  argTypes: { size: { control: { options: ['small', 'default'], type: 'inline-radio' } } },
  component: Slider,
  title: 'Ant Design/Slider',
} as Meta<typeof Slider>;

export const Default: ComponentStory<typeof Slider> = (args) => (
  <div style={{ width: 200 }}>
    <Slider {...args} />
  </div>
);

Default.args = {
  defaultValue: 30,
  dots: false,
  included: true,
  max: 100,
  min: 0,
  reverse: false,
  size: 'small',
  step: 1,
  vertical: false,
} as Partial<SliderSingleProps>;
