import { Meta, Story } from '@storybook/react';
import { Select } from 'antd';
import React from 'react';

import SelectFilter from './SelectFilter';

const { OptGroup, Option } = Select;

export default {
  argTypes: { count: { control: { max: 26, min: 0, step: 1, type: 'range' } } },
  component: SelectFilter,
  title: 'Determined/Dropdowns/SelectFilter',
} as Meta<typeof SelectFilter>;

type SelectFilterProps = React.ComponentProps<typeof SelectFilter>;

export const Default: Story<SelectFilterProps & { count: number }> = ({ count, ...args }) => {
  return (
    <SelectFilter {...args}>
      <OptGroup key="roup" label="Optional Grouping">
        <Option value="A">Option A</Option>
      </OptGroup>
      {new Array(count - 1).fill(null).map((v, index) => (
        <Option key={index + 1} value={String.fromCharCode(65 + index + 1)}>
          {'Option ' + String.fromCharCode(65 + index + 1)}
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
