import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback } from 'react';

import SelectFilter from './SelectFilter';

interface Props {
  label: string;
  onChange?: (value: (number|string)[]) => void;
  options: (number|string)[];
  value?: (number|string)[];
}

const ALL_VALUE = 'All';

const { Option } = Select;

const MultiSelect: React.FC<Props> = ({ label, onChange, options, value }: Props) => {

  const handleSelect = useCallback((option: SelectValue) => {
    if (!onChange) return;

    const optionString = option.toString();
    if (optionString === ALL_VALUE) {
      onChange([]);
      if (document && document.activeElement) {
        (document.activeElement as HTMLElement).blur();
      }
      return;
    }

    const newValue = Array.isArray(value) ? [ ...value ] : [];
    if (newValue.indexOf(optionString) === -1) newValue.push(optionString);
    onChange(newValue);
  }, [ onChange, value ]);

  const handleDeselect = useCallback((option: SelectValue) => {
    if (!onChange) return;

    const newValue = Array.isArray(value) ? [ ...value ] : [];
    const optionString = option.toString();
    const index = newValue.indexOf(optionString);
    if (index !== -1) newValue.splice(index, 1);
    onChange(newValue);
  }, [ onChange, value ]);

  return <SelectFilter
    disableTags
    dropdownMatchSelectWidth={200}
    label={label}
    mode="multiple"
    placeholder={'All'}
    showArrow
    style={{ width: 130 }}
    value={value}
    onDeselect={handleDeselect}
    onSelect={handleSelect}
  >
    <Option key={ALL_VALUE} value={ALL_VALUE}>
      {ALL_VALUE}
    </Option>
    {options.map((item: number|string) => (
      <Option key={item} value={item}>
        {item}
      </Option>
    ))}
    {options}
  </SelectFilter>;
};

export default MultiSelect;
