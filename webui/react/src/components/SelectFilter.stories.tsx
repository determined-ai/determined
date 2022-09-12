import { Meta, Story } from '@storybook/react';
import { Select } from 'antd';
import React from 'react';

import SelectFilter from './SelectFilter';

const { Option } = Select;

export default {
  argTypes: { count: { control: { max: 26, min: 0, step: 1, type: 'range' } } },
  component: SelectFilter,
  title: 'SelectFilter',
} as Meta<typeof SelectFilter>;

type SelectFilterProps = React.ComponentProps<typeof SelectFilter>;

export const Default: Story<SelectFilterProps & { count: number }> = ({ count, ...args }) => {
  return (
    <SelectFilter {...args}>
      {new Array(count).fill(null).map((v, index) => (
        <Option key={index} value={String.fromCharCode(65 + index)}>
          {'Option ' + String.fromCharCode(65 + index)}
        </Option>
      ))}
    </SelectFilter>
  );
};

Default.args = {
  count: 5,
  label: 'Default Label',
  placeholder: 'Pick an option',
};
