import { ComponentStory, Meta } from '@storybook/react';
import React, { useState } from 'react';

import AutoComplete from './AutoComplete';

export default {
  component: AutoComplete,
  title: 'Determined/AutoComplete',
} as Meta<typeof AutoComplete>;

const mockVal = (str: string, repeat = 1) => ({
  label: str.repeat(repeat),
  value: str.repeat(repeat),
});

export const Default: ComponentStory<typeof AutoComplete> = (args) => {
  const [options, setOptions] = useState<{ label: string; value: string }[]>([]);

  const onSearch = (searchText: string) => {
    setOptions(
      !searchText ? [] : [mockVal(searchText), mockVal(searchText, 2), mockVal(searchText, 3)],
    );
  };
  return (
    <div style={{ width: 200 }}>
      <AutoComplete {...args} options={options} onSearch={onSearch} />
    </div>
  );
};

Default.args = { allowClear: false, placeholder: 'Placeholder' };
