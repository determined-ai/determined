import { ComponentStory, Meta } from '@storybook/react';
import { Breadcrumb } from 'antd';
import React from 'react';

export default {
  component: Breadcrumb,
  title: 'Ant Design/Breadcrumb',
} as Meta<typeof Breadcrumb>;

export const Default: ComponentStory<typeof Breadcrumb> = (args) => (
  <Breadcrumb {...args}>
    <Breadcrumb.Item>Home</Breadcrumb.Item>
    <Breadcrumb.Item>
      <a href="">Application Center</a>
    </Breadcrumb.Item>
    <Breadcrumb.Item>
      <a href="">Application List</a>
    </Breadcrumb.Item>
    <Breadcrumb.Item>An Application</Breadcrumb.Item>
  </Breadcrumb>
);

Default.args = { separator: '/' };
