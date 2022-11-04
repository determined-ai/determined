import { ComponentStory, Meta } from '@storybook/react';
import { InputNumber } from 'antd';
import React from 'react';

export default {
  argTypes: {
    max: { control: { type: 'number' } },
    min: { control: { type: 'number' } },
    precision: { control: { max: 10, min: 1, step: 1, type: 'range' } },
    size: { control: { options: ['small', 'middle', 'large'], type: 'inline-radio' } },
    step: { control: { type: 'number' } },
  },
  component: InputNumber,
  title: 'Ant Design/InputNumber',
} as Meta<typeof InputNumber>;

export const Default: ComponentStory<typeof InputNumber> = (args) => <InputNumber {...args} />;

Default.args = {
  addonAfter: '',
  addonBefore: '',
  bordered: true,
  controls: true,
  disabled: false,
  max: undefined,
  min: undefined,
  placeholder: 'Placeholder text',
  precision: undefined,
  readOnly: false,
  size: 'middle',
  step: 1,
};
