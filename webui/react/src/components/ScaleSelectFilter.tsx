import { Select as AntdSelect } from 'antd';
import { SelectValue } from 'antd/es/select';
import React from 'react';

import Select from 'components/kit/Select';
import { capitalize } from 'shared/utils/string';

import { Scale } from '../types';

const { Option } = AntdSelect;

interface Props {
  onChange: (value: Scale) => void;
  value: Scale;
}

const ScaleSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  return (
    <Select
      enableSearchFilter={false}
      label="Scale"
      value={value}
      onSelect={(newValue: SelectValue) => onChange(newValue as Scale)}>
      {Object.values(Scale).map((scale) => (
        <Option key={scale} value={scale}>
          {capitalize(scale)}
        </Option>
      ))}
    </Select>
  );
};

export default ScaleSelectFilter;
