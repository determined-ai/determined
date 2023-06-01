import React from 'react';

import { ValueOf } from 'components/kit/internal/types';
import Select, { Option, SelectValue } from 'components/kit/Select';

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
      label="X-Axis"
      searchable={false}
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
