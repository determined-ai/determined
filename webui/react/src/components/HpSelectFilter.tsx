import { Select } from 'antd';
import { LabeledValue, SelectValue } from 'antd/es/select';
import React, { useCallback, useMemo } from 'react';

import HumanReadableFloat from 'components/HumanReadableFloat';
import { ALL_VALUE, HpImportance } from 'types';
import { clone, isObject } from 'utils/data';
import { hpImportanceSorter } from 'utils/sort';

import css from './HpSelectFilter.module.scss';
import SelectFilter, { Props as SelectFilterProps } from './SelectFilter';

const { Option } = Select;

interface Props extends SelectFilterProps {
  fullHParams: string[];
  hpImportance?: HpImportance;
}

const HpSelectFilter: React.FC<Props> = ({
  fullHParams,
  hpImportance = {},
  onChange,
  value,
  ...props
}: Props) => {
  const values = useMemo(() => {
    if (!value) return [];
    return Array.isArray(value) ? value : [ value ];
  }, [ value ]);

  const sortedFullHParams = useMemo(() => {
    const hParams = clone(fullHParams) as string[];
    return hParams.sortAll((a, b) => hpImportanceSorter(a, b, hpImportance));
  }, [ hpImportance, fullHParams ]);

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
      className={css.base}
      disableTags
      dropdownMatchSelectWidth={300}
      mode="multiple"
      placeholder={ALL_VALUE}
      showArrow
      style={{ width: 130 }}
      value={value}
      onDeselect={handleDeselect}
      onSelect={handleSelect}
      {...props}>
      <Option value={ALL_VALUE}>{ALL_VALUE}</Option>
      {sortedFullHParams.map(hParam => {
        const importance = hpImportance[hParam];
        return (
          <Option className={css.option} key={hParam} value={hParam}>
            {hParam}
            {importance && (
              <HumanReadableFloat num={importance} precision={1} tooltipPrefix="Importance: " />
            )}
          </Option>
        );
      })}
    </SelectFilter>
  );
};

export default HpSelectFilter;
