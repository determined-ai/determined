import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React from 'react';

import SelectFilter from 'components/kit/SelectFilter';
import { ValueOf } from 'shared/types';

const { Option } = Select;

export const XAxisDomain = {
  Batches: 'Batches',
  Epochs: 'Epoch',
  Time: 'Time',
} as const;

export type XAxisDomain = ValueOf<typeof XAxisDomain>;

interface Props {
  onChange: (value: XAxisDomain) => void;
  options: string[];
  value: string;
}

export const XAxisFilter: React.FC<Props> = ({ options, onChange, value }: Props) => {
  return (
    <SelectFilter
      enableSearchFilter={false}
      label="X-Axis"
      showSearch={false}
      value={value}
      onSelect={(newValue: SelectValue) => onChange(newValue as XAxisDomain)}>
      {Object.values(XAxisDomain)
        .filter((opt) => options.includes(opt))
        .map((opt) => (
          <Option key={opt} value={opt}>
            {opt}
          </Option>
        ))}
    </SelectFilter>
  );
};
