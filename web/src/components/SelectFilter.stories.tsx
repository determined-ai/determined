import { number, text, withKnobs } from '@storybook/addon-knobs';
import { Select } from 'antd';
import React from 'react';

import SelectFilter from './SelectFilter';

const { Option } = Select;

export default {
  component: SelectFilter,
  decorators: [ withKnobs ],
  title: 'SelectFilter',
};

export const Default = (): React.ReactNode => {
  const count = number('Number of Options', 5, { max: 26, min: 0, range: true, step: 1 });
  return (
    <SelectFilter
      label={text('Label', 'Default Label')}
      placeholder={text('Placeholder', 'Pick an option')}>
      {new Array(count).fill(null).map((v, index) => (
        <Option key={index} value={String.fromCharCode(65 + index)}>
          {'Option ' + String.fromCharCode(65 + index)}
        </Option>
      ))}
    </SelectFilter>
  );
};
