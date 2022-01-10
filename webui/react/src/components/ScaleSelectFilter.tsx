import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React from 'react';

import SelectFilter from 'components/SelectFilter';
import { capitalize } from 'utils/string';

const { Option } = Select;

export enum Scale {
  Linear = 'linear',
  Log = 'log',
}

interface Props {
  onChange: (value: Scale) => void;
  value: Scale;
}

const ScaleSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  return (
    <SelectFilter
      enableSearchFilter={false}
      label="Scale"
      showSearch={false}
      value={value}
      onSelect={(newValue: SelectValue) => onChange(newValue as Scale)}>
      {Object.values(Scale).map(scale => (
        <Option key={scale} value={scale}>{capitalize(scale)}</Option>
      ))}
    </SelectFilter>
  );
};

export default ScaleSelectFilter;
