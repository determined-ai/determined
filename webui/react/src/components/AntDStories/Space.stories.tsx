import { Meta, Story } from '@storybook/react';
import { Space } from 'antd';
import React from 'react';

export default {
  argTypes: {
    align: { table: { disable: true } },
    direction: { control: { options: ['horizontal', 'vertical'], type: 'inline-radio' } },
    numItems: { control: { max: 10, min: 1, step: 1, type: 'range' } },
    size: { control: { options: ['small', 'middle', 'large'], type: 'inline-radio' } },
  },
  component: Space,
  title: 'Ant Design/Space',
} as Meta<typeof Space>;

const SpaceComponent: React.FC = () => {
  return <div style={{ backgroundColor: 'gray', height: 100, width: 100 }} />;
};

type SpaceProps = React.ComponentProps<typeof Space>;

export const Default: Story<SpaceProps & { numItems: number }> = ({ numItems, ...args }) => (
  <div style={{ width: 500 }}>
    <Space {...args}>
      {new Array(numItems).fill(0).map((_item, idx) => (
        <SpaceComponent key={idx} />
      ))}
    </Space>
  </div>
);

Default.args = {
  direction: 'horizontal',
  numItems: 3,
  size: 'small',
  wrap: false,
};
