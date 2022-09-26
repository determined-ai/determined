import { ComponentStory, Meta } from '@storybook/react';
import { Pagination } from 'antd';
import React from 'react';

export default {
  argTypes: {
    total: {
      control: {
        type: 'number',
      },
    },
  },
  component: Pagination,
  title: 'Ant Design/Pagination',
} as Meta<typeof Pagination>;

export const Default: ComponentStory<typeof Pagination> = (args) => <Pagination {...args} />;

Default.args = {
  showSizeChanger: true,
  total: 20,
};
