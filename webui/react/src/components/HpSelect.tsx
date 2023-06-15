import { DefaultOptionType, LabeledValue, SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import Select, { Option, SelectProps } from 'components/kit/Select';
import { ALL_VALUE } from 'types';
import { clone, isObject } from 'utils/data';

import css from './HpSelect.module.scss';

interface Props extends SelectProps {
  fullHParams: string[];
}

const HpSelect: React.FC<Props> = ({ fullHParams, onChange, value, ...props }: Props) => {
  const values = useMemo(() => {
    if (!value) return [];
    return Array.isArray(value) ? value : [value];
  }, [value]);

  const handleSelect = useCallback(
    (selected: SelectValue, option: DefaultOptionType | DefaultOptionType[]) => {
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
    },
    [onChange, values],
  );

  const handleDeselect = useCallback(
    (selected: SelectValue, option: DefaultOptionType | DefaultOptionType[]) => {
      if (!onChange) return;

      const selectedValue = isObject(selected) ? (selected as LabeledValue).value : selected;
      const newValue = (clone(values) as SelectValue[]).filter((item) => item !== selectedValue);

      onChange(newValue as SelectValue, option);
    },
    [onChange, values],
  );

  return (
    <Select
      disableTags
      mode="multiple"
      placeholder={ALL_VALUE}
      value={value}
      width={200}
      onDeselect={handleDeselect}
      onSelect={handleSelect}
      {...props}>
      <Option key={ALL_VALUE} value={ALL_VALUE}>
        All
      </Option>
      {fullHParams.map((hParam) => {
        return (
          <Option className={css.option} key={hParam} value={hParam}>
            {hParam}
          </Option>
        );
      })}
    </Select>
  );
};

export default HpSelect;
