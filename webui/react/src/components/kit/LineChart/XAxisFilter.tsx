import { Select as AntdSelect } from 'antd';
import { SelectValue } from 'antd/es/select';
import React from 'react';

import Select from 'components/kit/Select';
import { ValueOf } from 'shared/types';

const { Option } = AntdSelect;

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
    <Select
      enableSearchFilter={false}
      label="X-Axis"
      value={value}
      onSelect={(newValue: SelectValue) => onChange(newValue as XAxisDomain)}>
      {Object.values(XAxisDomain)
        .filter((opt) => options.includes(opt))
        .map((opt) => (
          <Option key={opt} value={opt}>
            {opt}
          </Option>
        ))}
    </Select>
  );
};
