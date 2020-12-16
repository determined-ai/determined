import { number, text, withKnobs } from '@storybook/addon-knobs';
import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useState } from 'react';

import MultiSelect from './MultiSelect';

const { Option } = Select;

export default {
  component: MultiSelect,
  decorators: [ withKnobs ],
  title: 'MultiSelect',
};

export const Default = (): React.ReactNode => {
  const [ value, setValue ] = useState<string[]>([]);
  const count = number('Number of Options', 5, { max: 26, min: 0, range: true, step: 1 });
  const onChange = useCallback((value: SelectValue) => {
    setValue(value as string[]);
  }, []);

  return (
    <MultiSelect
      label={text('Label', 'Default Label')}
      placeholder={text('Placeholder', 'All')}
      value={value}
      onChange={onChange}
    >
      {new Array(count).fill(null).map((v, index) => (
        <Option key={index} value={String.fromCharCode(65 + index)}>
          Option {String.fromCharCode(65 + index)}
        </Option>
      ))}
    </MultiSelect>
  );
};
