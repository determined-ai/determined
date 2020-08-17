import { Select } from 'antd';
import { SelectProps, SelectValue } from 'antd/es/select';
import React, { PropsWithChildren, useCallback } from 'react';

import Icon from './Icon';
import Label from './Label';
import css from './SelectFilter.module.scss';

interface Props<T = SelectValue> extends SelectProps<T> {
  enableSearchFilter?: boolean;
  label: string;
  style?: React.CSSProperties;
}

export const ALL_VALUE = 'all';

const SelectFilter: React.FC<PropsWithChildren<Props>> = ({
  dropdownMatchSelectWidth = false,
  enableSearchFilter = true,
  showSearch = true,
  ...props
}: PropsWithChildren<Props>) => {

  const getPopupContainer = useCallback((triggerNode) => triggerNode, []);

  const handleFilter = useCallback((search: string, option) => {
    /*
     * `option.children` is one of the following:
     * - undefined
     * - string
     * - string[]
     */
    let label = null;
    if (option.children) {
      if (Array.isArray(option.children)) label = option.children.join(' ').toLocaleLowerCase();
      else label = option.children.toLocaleLowerCase();
    }
    return label && label.indexOf(search.toLocaleLowerCase()) !== -1;
  }, []);

  return (
    <div className={css.base}>
      <Label>{props.label}</Label>
      <Select
        dropdownMatchSelectWidth={dropdownMatchSelectWidth}
        filterOption={enableSearchFilter ? handleFilter : true}
        getPopupContainer={getPopupContainer}
        showSearch={showSearch}
        suffixIcon={<Icon name="arrow-down" size="tiny" />}
        {...props}>
        {props.children}
      </Select>
    </div>
  );
};

export default SelectFilter;
