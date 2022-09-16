import { ComponentStory, Meta } from '@storybook/react';
import { Button, Menu } from 'antd';
import React from 'react';

import Dropdown, { Placement } from './Dropdown';

export default {
  component: Dropdown,
  title: 'Determined/Dropdowns/Dropdown',
} as Meta<typeof Dropdown>;

const content = (
  <Menu
    items={new Array(7).fill(null).map((_, index) => ({ key: index, label: `Menu Item ${index}` }))}
  />
);

export const Default: ComponentStory<typeof Dropdown> = (args) => (
  <Dropdown {...args} content={content}>
    <Button>Default Dropdown</Button>
  </Dropdown>
);

Default.args = { placement: Placement.BottomLeft, showArrow: true };
