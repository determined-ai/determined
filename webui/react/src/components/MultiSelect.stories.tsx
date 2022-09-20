import { Meta, Story } from '@storybook/react';
import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useState } from 'react';

import MultiSelect from './MultiSelect';

const { Option } = Select;

export default {
  argTypes: { count: { control: { max: 26, min: 0, step: 1, type: 'range' } } },
  component: MultiSelect,
  title: 'Determined/Dropdowns/MultiSelect',
} as Meta<typeof MultiSelect>;

type MultiSelectProps = React.ComponentProps<typeof MultiSelect>;

export const Default: Story<MultiSelectProps & { count: number }> = ({ count, ...args }) => {
  const [value, setValue] = useState<string[]>([]);
  const onChange = useCallback((value: SelectValue) => {
    setValue(value as string[]);
  }, []);

  return (
    <MultiSelect {...args} value={value} onChange={onChange}>
      {new Array(count).fill(null).map((v, index) => (
        <Option key={index} value={String.fromCharCode(65 + index)}>
          Option {String.fromCharCode(65 + index)}
        </Option>
      ))}
    </MultiSelect>
  );
};

Default.args = {
  count: 5,
  label: 'Default Label',
  placeholder: 'All',
};
