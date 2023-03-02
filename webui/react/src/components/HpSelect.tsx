import { DefaultOptionType, LabeledValue, SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import HumanReadableNumber from 'components/HumanReadableNumber';
import Select, { Option, SelectProps } from 'components/kit/Select';
import { clone, isObject } from 'shared/utils/data';
import { ALL_VALUE, HpImportance } from 'types';

import { hpImportanceSorter } from '../utils/experiment';

import css from './HpSelect.module.scss';

interface Props extends SelectProps {
  fullHParams: string[];
  hpImportance?: HpImportance;
}

const HpSelect: React.FC<Props> = ({
  fullHParams,
  hpImportance = {},
  onChange,
  value,
  ...props
}: Props) => {
  const values = useMemo(() => {
    if (!value) return [];
    return Array.isArray(value) ? value : [value];
  }, [value]);

  const sortedFullHParams = useMemo(() => {
    const hParams = clone(fullHParams) as string[];
    return hParams.sortAll((a, b) => hpImportanceSorter(a, b, hpImportance));
  }, [hpImportance, fullHParams]);

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
      onDeselect={handleDeselect}
      onSelect={handleSelect}
      {...props}>
      <Option key={ALL_VALUE} value={ALL_VALUE}>
        All
      </Option>
      {sortedFullHParams.map((hParam) => {
        const importance = hpImportance[hParam];
        return (
          <Option className={css.option} key={hParam} value={hParam}>
            {hParam}
            {importance && (
              <HumanReadableNumber num={importance} precision={1} tooltipPrefix="Importance: " />
            )}
          </Option>
        );
      })}
    </Select>
  );
};

export default HpSelect;
