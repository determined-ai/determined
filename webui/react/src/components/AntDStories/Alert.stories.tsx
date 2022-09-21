import { ComponentStory, Meta } from '@storybook/react';
import { Alert } from 'antd';
import React from 'react';

export default {
  argTypes: {
    type: {
      control: {
        options: ['success', 'info', 'warning', 'error'],
        type: 'inline-radio',
      },
    },
  },
  component: Alert,
  title: 'Ant Design/Alert',
} as Meta<typeof Alert>;

export const Default: ComponentStory<typeof Alert> = (args) => <Alert {...args} />;

Default.args = {
  banner: false,
  closable: false,
  closeText: '',
  description: '',
  message: 'Message',
  showIcon: false,
  type: 'success',
};
