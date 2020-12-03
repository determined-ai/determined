import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import SelectFilter from './SelectFilter';

interface LabeledValue {
  label: number|string;
  value: number|string;
}

interface Props {
  label: string;
  onChange?: (value: (number|string)[]) => void;
  options: (number|string|LabeledValue)[];
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

  const selectOptions = useMemo(() => {
    return options.map((item: number|string|LabeledValue) => (
      [ 'string', 'number' ].indexOf(typeof item) >= 0 ? (
        <Option key={item as number|string} value={item as string|number}>
          {item as string|number}
        </Option>
      ) : (
        <Option key={(item as LabeledValue).value} value={(item as LabeledValue).value}>
          {(item as LabeledValue).label}
        </Option>
      )
    ));
  }, [ options ]);

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
    {selectOptions}
  </SelectFilter>;
};

export default MultiSelect;
