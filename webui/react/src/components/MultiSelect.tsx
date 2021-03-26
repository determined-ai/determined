import { Select } from 'antd';
import { LabeledValue, SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import { ALL_VALUE } from 'types';
import { clone, isObject } from 'utils/data';

import SelectFilter, { Props as SelectFilterProps } from './SelectFilter';

const { Option } = Select;

const MultiSelect: React.FC<SelectFilterProps> = ({ onChange, value, ...props }) => {
  const values = useMemo(() => {
    if (!value) return [];
    return Array.isArray(value) ? value : [ value ];
  }, [ value ]);

  const handleSelect = useCallback((selected: SelectValue, option) => {
    if (!onChange) return;

    if (selected === ALL_VALUE) {
      onChange([], option);
      if (document.activeElement) (document.activeElement as HTMLElement).blur();
    } else {
      const newValue = clone(values);
      const selectedValue = isObject(selected) ? (selected as LabeledValue).value : selected;

      if (!newValue.includes(selectedValue)) newValue.push(selectedValue);

      onChange(newValue as SelectValue, option);
    }
  }, [ onChange, values ]);

  const handleDeselect = useCallback((selected: SelectValue, option) => {
    if (!onChange) return;

    const selectedValue = isObject(selected) ? (selected as LabeledValue).value : selected;
    const newValue = (clone(values) as SelectValue[]).filter(item => item !== selectedValue);

    onChange(newValue as SelectValue, option);
  }, [ onChange, values ]);

  return (
    <SelectFilter
      disableTags
      dropdownMatchSelectWidth={200}
      mode="multiple"
      placeholder="All"
      showArrow
      style={{ width: 130 }}
      value={value}
      onDeselect={handleDeselect}
      onSelect={handleSelect}
      {...props}>
      <Option value={ALL_VALUE}>All</Option>
      {props.children}
    </SelectFilter>
  );
};

export default MultiSelect;
