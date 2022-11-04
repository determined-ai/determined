import { ComponentStory, Meta } from '@storybook/react';
import { Tooltip } from 'antd';
import React from 'react';

export default {
  argTypes: {
    color: { control: 'color' },
    placement: {
      control: {
        options: [
          'top',
          'left',
          'right',
          'bottom',
          'topLeft',
          'topRight',
          'bottomLeft',
          'bottomRight',
          'leftTop',
          'leftBottom',
          'rightTop',
          'rightBottom',
        ],
        type: 'inline-radio',
      },
    },
    trigger: {
      control: {
        options: ['hover', 'focus', 'click', 'contextMenu'],
        type: 'inline-check',
      },
    },
  },
  component: Tooltip,
  title: 'Ant Design/Tooltip',
} as Meta<typeof Tooltip>;

export const Default: ComponentStory<typeof Tooltip> = (args) => (
  <Tooltip {...args}>Trigger</Tooltip>
);

Default.args = {
  mouseEnterDelay: 0.1,
  mouseLeaveDelay: 0.1,
  placement: 'top',
  title: 'Tooltip text',
  trigger: 'hover',
};
