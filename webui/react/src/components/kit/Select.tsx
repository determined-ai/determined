import { Select as AntdSelect, SelectProps as AntdSelectProps } from 'antd';
import type { DefaultOptionType, RefSelectProps, SelectValue } from 'antd/es/select';
import React, { forwardRef, useCallback, useMemo, useState } from 'react';

import Icon from 'components/kit/Icon';
import Label, { LabelTypes } from 'components/kit/internal/Label';

import css from './Select.module.scss';

const { OptGroup, Option } = AntdSelect;

export { Option, OptGroup, SelectValue };

type Options = DefaultOptionType | DefaultOptionType[];
export interface SelectProps<T extends SelectValue = SelectValue> {
  allowClear?: boolean;
  autoFocus?: boolean;
  defaultValue?: T;
  disableTags?: boolean;
  disabled?: boolean;
  filterOption?: AntdSelectProps['filterOption'];
  filterSort?: AntdSelectProps['filterSort'];
  id?: string;
  label?: string;
  loading?: boolean;
  mode?: 'multiple' | 'tags';
  onBlur?: () => void;
  onChange?: (value: T, option: Options) => void;
  onDeselect?: (selected: SelectValue, option: Options) => void;
  onSearch?: (searchInput: string) => void;
  onSelect?: (selected: SelectValue, option: Options) => void;
  options?: AntdSelectProps['options'];
  placeholder?: string;
  ref?: React.Ref<RefSelectProps>;
  dropdownMatchSelectWidth?: boolean | number;
  searchable?: boolean;
  value?: T;
  width?: React.CSSProperties['width'];
  onDropdownVisibleChange?: (open: boolean) => void;
}

const countOptions = (children: React.ReactNode, options?: Options): number => {
  let count = 0;

  if (options) return options.length;
  if (!children) return count;

  if (Array.isArray(children)) {
    count += children
      .map((child) => countOptions(child, options))
      .reduce((acc, count) => acc + count, 0);
  }

  const childType = (children as React.ReactElement).type;
  const childProps = (children as React.ReactElement).props;
  const childList = (childProps as React.ReactPortal)?.children;
  if (childType === Option) count++;
  if (childType === OptGroup && childList) count += countOptions(childList, options);

  return count;
};

const Select: React.FC<React.PropsWithChildren<SelectProps>> = forwardRef(function Select(
  {
    disabled,
    disableTags = false,
    searchable = true,
    dropdownMatchSelectWidth = true,
    filterOption,
    label,
    loading,
    onSearch,
    options,
    width,
    value,
    children,
    ...passthrough
  }: React.PropsWithChildren<SelectProps>,
  ref?: React.Ref<RefSelectProps>,
) {
  const [isOpen, setIsOpen] = useState(false);
  const classes = [css.base];

  if (disableTags) classes.push(css.disableTags);

  const optionsCount = useMemo(() => countOptions(children, options), [children, options]);
  const [maxTagCount, maxTagPlaceholder] = useMemo((): [0 | undefined, string] => {
    if (!disableTags) return [undefined, ''];
    const count = Array.isArray(value) ? value.length : value ? 1 : 0;
    const itemLabel = 'selected';
    const placeholder = count === optionsCount ? 'All' : `${count} ${itemLabel}`;
    return isOpen ? [0, ''] : [0, placeholder];
  }, [disableTags, isOpen, optionsCount, value]);

  const handleDropdownVisibleChange = useCallback((open: boolean) => {
    setIsOpen(open);
  }, []);
  const handleFilter = useCallback((search: string, option?: DefaultOptionType): boolean => {
    let label: string | null = null;

    if (!option?.children && !option?.label) return false;

    if (Array.isArray(option.children)) {
      label = option.children.join(' ');
    } else if (option.label) {
      label = option.label.toString();
    } else if (typeof option.children === 'string') {
      label = option.children;
    }
    return !!label && label.toLocaleLowerCase().includes(search.toLocaleLowerCase());
  }, []);

  return (
    <div className={classes.join(' ')}>
      {label && <Label type={LabelTypes.TextOnly}>{label}</Label>}
      <AntdSelect
        disabled={disabled || loading}
        dropdownMatchSelectWidth={dropdownMatchSelectWidth}
        filterOption={filterOption ?? (searchable ? handleFilter : true)}
        maxTagCount={maxTagCount}
        maxTagPlaceholder={maxTagPlaceholder}
        options={options}
        ref={ref}
        showSearch={!!onSearch || !!filterOption || searchable}
        style={{ width }}
        suffixIcon={!loading ? <Icon name="arrow-down" size="tiny" title="Open" /> : undefined}
        value={value}
        onDropdownVisibleChange={handleDropdownVisibleChange}
        onSearch={onSearch}
        {...passthrough}>
        {children}
      </AntdSelect>
    </div>
  );
});

export default Select;
