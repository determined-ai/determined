import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React from 'react';

import SelectFilter from 'components/SelectFilter';
import { capitalize } from 'shared/utils/string';

const { Option } = Select;

interface Props {
  onChange: (value: string) => void;
  options: string[];
  value: string;
}

const XAxisFilter: React.FC<Props> = ({ options, onChange, value }: Props) => {
  return (
    <SelectFilter
      enableSearchFilter={false}
      label="X-Axis"
      showSearch={false}
      value={value}
      onSelect={(newValue: SelectValue) => onChange(newValue as string)}>
      {options.map((opt) => (
        <Option key={opt} value={opt}>
          {capitalize(opt)}
        </Option>
      ))}
    </SelectFilter>
  );
};

export default XAxisFilter;
