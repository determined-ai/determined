import { Select } from 'antd';
import { LabeledValue, SelectValue } from 'antd/es/select';
import React, { useCallback } from 'react';

import SelectFilter, { Props as SelectFilterProps } from './SelectFilter';

const ALL_VALUE = 'All';

const { Option } = Select;

const MultiSelect: React.FC<SelectFilterProps> = (
  { children, onChange, value, ...props }: SelectFilterProps,
) => {

  const handleSelect = useCallback((selectedValue: SelectValue, option) => {
    if (!onChange) return;

    if (selectedValue === ALL_VALUE) {
      onChange([], option);
      if (document && document.activeElement) {
        (document.activeElement as HTMLElement).blur();
      }
      return;
    }

    const newValue = Array.isArray(value) ? [ ...value ] : [];
    if (typeof selectedValue === 'object') {
      if (newValue.indexOf((selectedValue as LabeledValue).value) === -1) {
        newValue.push((selectedValue as LabeledValue).value);
      }
    } else {
      if (newValue.indexOf(selectedValue) === -1) {
        newValue.push(selectedValue);
      }
    }
    onChange(newValue as SelectValue, option);
  }, [ onChange, value ]);

  const handleDeselect = useCallback((selectedValue: SelectValue, option) => {
    if (!onChange) return;

    let newValue = Array.isArray(value) ? [ ...value ] : [];
    if (typeof selectedValue === 'object') {
      newValue = newValue.filter((item) => item !== (selectedValue as LabeledValue).value);
    } else {
      newValue = newValue.filter((item) => item !== selectedValue);
    }

    onChange(newValue as SelectValue, option);
  }, [ onChange, value ]);

  return <SelectFilter
    disableTags
    dropdownMatchSelectWidth={200}
    mode="multiple"
    placeholder={'All'}
    showArrow
    style={{ width: 130 }}
    value={value}
    onDeselect={handleDeselect}
    onSelect={handleSelect}
    {...props}
  >
    <Option key={ALL_VALUE} value={ALL_VALUE}>
      {ALL_VALUE}
    </Option>
    {children}
  </SelectFilter>;
};

export default MultiSelect;
