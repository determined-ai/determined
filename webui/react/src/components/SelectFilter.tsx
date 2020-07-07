import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { PropsWithChildren, useCallback } from 'react';

import Icon from './Icon';
import Label from './Label';
import css from './SelectFilter.module.scss';

interface Props {
  enableSearchFilter?: boolean;
  label: string;
  onSelect?: (value: SelectValue) => void;
  placeholder?: string | React.ReactNode;
  showSearch?: boolean;
  style?: React.CSSProperties;
  value?: SelectValue;
}

const defaultProps = {
  enableSearchFilter: true,
  showSearch: true,
};

export const ALL_VALUE = 'all';

const SelectFilter: React.FC<PropsWithChildren<Props>> = (props: PropsWithChildren<Props>) => {
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
        defaultValue={props.value}
        dropdownMatchSelectWidth={false}
        filterOption={props.enableSearchFilter ? handleFilter : true}
        placeholder={props.placeholder}
        showSearch={props.showSearch}
        style={props.style}
        suffixIcon={<Icon name="arrow-down" size="tiny" />}
        onSelect={props.onSelect}>
        {props.children}
      </Select>
    </div>
  );
};

SelectFilter.defaultProps = defaultProps;

export default SelectFilter;
