import { ComponentStory, Meta } from '@storybook/react';
import { Switch } from 'antd';
import React from 'react';

export default {
  argTypes: { size: { control: { options: ['small', 'default'], type: 'inline-radio' } } },
  component: Switch,
  title: 'Ant Design/Switch',
} as Meta<typeof Switch>;

export const Default: ComponentStory<typeof Switch> = (args) => <Switch {...args} />;

Default.args = { size: 'small' };
