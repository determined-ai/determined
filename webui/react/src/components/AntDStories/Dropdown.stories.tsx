import { ComponentStory, Meta } from '@storybook/react';
import { Dropdown } from 'antd';
import type { DropdownProps } from 'antd';
import React from 'react';

export default {
  argTypes: {
    placement: {
      control: {
        options: ['bottom', 'bottomLeft', 'bottomRight', 'top', 'topLeft', 'topRight'],
        type: 'select',
      },
    },
    size: { control: { options: ['small', 'middle', 'large'], type: 'inline-radio' } },
    trigger: { control: { options: ['click', 'hover', 'contextMenu'], type: 'inline-check' } },
    type: {
      control: {
        options: ['primary', 'dashed', 'link', 'text', 'default'],
        type: 'inline-radio',
      },
    },
  },
  component: Dropdown,
  title: 'Ant Design/Dropdown',
} as Meta<typeof Dropdown>;

const content: DropdownProps['menu'] = {
  items: [
    ...new Array(3).fill(null).map((_, index) => ({ key: index, label: `Menu Item ${index}` })),
    { type: 'divider' },
    { disabled: true, key: 5, label: 'Last Menu Item' },
  ],
};

export const Default: ComponentStory<typeof Dropdown> = (args) => (
  <Dropdown {...args} menu={content}>
    <a onClick={(e) => e.preventDefault()}>Default Dropdown</a>
  </Dropdown>
);

export const DropdownButton: ComponentStory<typeof Dropdown.Button> = (args) => (
  <Dropdown.Button {...args} menu={content}>
    Dropdown
  </Dropdown.Button>
);

Default.args = { arrow: true, disabled: false, placement: 'bottomLeft', trigger: ['hover'] };
DropdownButton.args = {
  disabled: false,
  loading: false,
  placement: 'bottomLeft',
  size: 'middle',
  trigger: ['hover'],
  type: 'default',
};
