import { Select as AntdSelect } from 'antd';
import type { DefaultOptionType, LabeledValue, RefSelectProps, SelectValue } from 'antd/es/select';
import React, { forwardRef, useCallback, useMemo, useState } from 'react';

import Label, { LabelTypes } from 'components/Label';
import Icon from 'shared/components/Icon/Icon';

import css from './Select.module.scss';

const { OptGroup, Option } = AntdSelect;

type Options = DefaultOptionType | DefaultOptionType[];
export interface Props<T = SelectValue> {
  allowClear?: boolean;
  defaultValue?: T;
  disableTags?: boolean;
  disabled?: boolean;
  enableSearchFilter?: boolean;
  filterOption?: boolean | ((inputValue: string, option: LabeledValue | undefined) => boolean);
  filterSort?: (a: LabeledValue, b: LabeledValue) => 1 | -1;
  id?: string;
  label?: string;
  maxTagCount?: -1 | 0 | 'responsive';
  maxTagPlaceholder?: string;
  mode?: 'multiple' | 'tags';
  onBlur?: () => void;
  onChange?: (value: T, option: Options) => void;
  onDeselect?: (selected: SelectValue, option: Options) => void;
  onSearch?: (searchInput: string) => void;
  onSelect?: (selected: SelectValue, option: Options) => void;
  options?: LabeledValue[];
  placeholder?: string;
  placement?: 'bottomLeft' | 'bottomRight' | 'topLeft' | 'topRight';
  ref?: React.Ref<RefSelectProps>;
  showSearch?: boolean;
  value?: T;
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

const Select: React.FC<React.PropsWithChildren<Props>> = forwardRef(function Select(
  {
    allowClear,
    defaultValue,
    disabled,
    disableTags = false,
    enableSearchFilter = true,
    filterOption,
    filterSort,
    id,
    label,
    mode,
    onChange,
    onBlur,
    onDeselect,
    onSearch,
    onSelect,
    options,
    placeholder,
    placement,
    showSearch,
    value,
    maxTagPlaceholder,
    children,
  }: React.PropsWithChildren<Props>,
  ref?: React.Ref<RefSelectProps>,
) {
  const [isOpen, setIsOpen] = useState(false);
  const classes = [css.base];

  if (disableTags) classes.push(css.disableTags);
  if (mode === 'multiple') {
    classes.push(css.multiple);
  }
  const optionsCount = useMemo(() => countOptions(children, options), [children, options]);

  const [maxTagCount, maxTagPlaceholderValue] = useMemo((): [
    -1 | 0 | undefined | 'responsive',
    string,
  ] => {
    if (!disableTags) return [undefined, maxTagPlaceholder ? maxTagPlaceholder : ''];
    const count = Array.isArray(value) ? value.length : value ? 1 : 0;
    const itemLabel = 'selected';
    const placeholder = count === optionsCount ? 'All' : `${count} ${itemLabel}`;
    return isOpen ? [0, ''] : [0, placeholder];
  }, [disableTags, isOpen, optionsCount, maxTagPlaceholder, value]);

  const handleDropdownVisibleChange = useCallback((open: boolean) => {
    setIsOpen(open);
  }, []);
  const handleFilter = useCallback((search: string, option?: DefaultOptionType): boolean => {
    let label: string | null = null;
    if (option?.children) {
      if (Array.isArray(option.children)) {
        label = option.children.join(' ');
      } else if (option.label) {
        label = option.label.toString();
      } else if (typeof option.children === 'string') {
        label = option.children;
      }
    }

    return !!label && label.toLocaleLowerCase().indexOf(search.toLocaleLowerCase()) !== -1;
  }, []);
  return (
    <div className={classes.join(' ')}>
      {label && <Label type={LabelTypes.TextOnly}>{label}</Label>}
      <AntdSelect
        allowClear={allowClear}
        defaultValue={defaultValue}
        disabled={disabled}
        dropdownMatchSelectWidth={250}
        filterOption={
          enableSearchFilter || filterOption ? (filterOption ? filterOption : handleFilter) : true
        }
        filterSort={filterSort}
        id={id}
        maxTagCount={maxTagCount}
        maxTagPlaceholder={maxTagPlaceholderValue}
        mode={mode}
        options={options ? options : undefined}
        placeholder={placeholder}
        placement={placement}
        ref={ref}
        showSearch={!!onSearch || !!filterOption || showSearch}
        suffixIcon={<Icon name="arrow-down" size="tiny" />}
        value={value}
        onBlur={onBlur}
        onChange={onChange}
        onDeselect={onDeselect}
        onDropdownVisibleChange={handleDropdownVisibleChange}
        onSearch={onSearch}
        onSelect={onSelect}>
        {children}
      </AntdSelect>
    </div>
  );
});

export default Select;
