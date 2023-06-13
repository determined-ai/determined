import React from 'react';

import Select, { Option, SelectValue } from 'components/kit/Select';
import { capitalize } from 'utils/string';

import { Scale } from '../types';

interface Props {
  onChange: (value: Scale) => void;
  value: Scale;
}

const ScaleSelect: React.FC<Props> = ({ onChange, value }: Props) => {
  return (
    <Select
      label="Scale"
      searchable={false}
      value={value}
      width={90}
      onSelect={(newValue: SelectValue) => onChange(newValue as Scale)}>
      {Object.values(Scale).map((scale) => (
        <Option key={scale} value={scale}>
          {capitalize(scale)}
        </Option>
      ))}
    </Select>
  );
};

export default ScaleSelect;
