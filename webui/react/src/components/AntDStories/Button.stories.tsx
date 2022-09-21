import { ComponentStory, Meta } from '@storybook/react';
import { Button } from 'antd';
import React from 'react';

export default {
  argTypes: {
    shape: { control: { options: ['default', 'circle', 'round'], type: 'inline-radio' } },
    size: { control: { options: ['small', 'middle', 'large'], type: 'inline-radio' } },
    type: {
      control: {
        options: ['primary', 'dashed', 'link', 'text', 'default'],
        type: 'inline-radio',
      },
    },
  },
  component: Button,
  title: 'Ant Design/Button',
} as Meta<typeof Button>;

export const Default: ComponentStory<typeof Button> = (args) => <Button {...args} />;

Default.args = {
  children: 'Button Text',
  danger: false,
  disabled: false,
  ghost: false,
  shape: 'default',
  size: 'middle',
  type: 'primary',
};
