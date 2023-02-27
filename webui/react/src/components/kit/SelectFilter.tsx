import { Select } from 'antd';
import type { DefaultOptionType, LabeledValue, RefSelectProps, SelectValue } from 'antd/es/select';
import React, { forwardRef, useCallback, useMemo, useState } from 'react';

import Label, { LabelTypes } from 'components/Label';
import Icon from 'shared/components/Icon/Icon';

import css from './SelectFilter.module.scss';

const { OptGroup, Option } = Select;

type Options = DefaultOptionType | DefaultOptionType[];
export interface Props<T = SelectValue> {
  allowClear?: boolean;
  autoClearSearchValue?: boolean;
  defaultValue?: T;
  disableTags?: boolean;
  disabled?: boolean;
  enableSearchFilter?: boolean;
  filterOption?: (input: string, option: LabeledValue) => void;
  filterSort?: (a: LabeledValue, b: LabeledValue) => 1 | -1;
  id?: string;
  itemName?: string;
  label?: string;
  maxTagCount?: number;
  maxTagPlaceholderValue?: string;
  mode?: 'multiple';
  onBlur?: () => void;
  onChange?: (value: T, option: Options) => void;
  onDeselect?: (selected: SelectValue, option: Options) => void;
  onDropdownVisibleChange?: () => void;
  onSearch?: (searchInput: string) => void;
  onSelect?: (selected: SelectValue, option: Options) => void;
  options?: LabeledValue[];
  placeholder?: string;
  ref?: React.Ref<RefSelectProps>;
  showArrow?: boolean;
  size?: 'large';
  value?: T;
  verticalLayout?: boolean;
}

export const ALL_VALUE = 'all';

const countOptions = (children: React.ReactNode): number => {
  if (!children) return 0;

  let count = 0;
  if (Array.isArray(children)) {
    count += children.map((child) => countOptions(child)).reduce((acc, count) => acc + count, 0);
  }

  const childType = (children as React.ReactElement).type;
  const childProps = (children as React.ReactElement).props;
  const childList = (childProps as React.ReactPortal)?.children;
  if (childType === Option) count++;
  if (childType === OptGroup && childList) count += countOptions(childList);

  return count;
};

const SelectFilter: React.FC<React.PropsWithChildren<Props>> = forwardRef(function SelectFilter(
  {
    allowClear,
    autoClearSearchValue,
    defaultValue,
    disabled,
    disableTags = false,
    /*
     * Disabling `dropdownMatchSelectWidth` will disable virtual scroll within the dropdown options.
     * This should only be done if the option count is fairly low.
     */
    enableSearchFilter = true,
    filterSort,
    id,
    itemName,
    label,
    mode,
    onChange,
    onBlur,
    onDeselect,
    onSearch,
    onSelect,
    options,
    placeholder,
    verticalLayout = false,
    value,
    maxTagPlaceholderValue,
    children,
  }: React.PropsWithChildren<Props>,
  ref?: React.Ref<RefSelectProps>,
) {
  const [isOpen, setIsOpen] = useState(false);
  const classes = [css.base];

  if (disableTags) classes.push(css.disableTags);
  if (verticalLayout) classes.push(css.vertical);

  const optionsCount = useMemo(() => countOptions(children), [children]);

  const [maxTagCount, maxTagPlaceholder] = useMemo(() => {
    if (!disableTags) return [undefined, maxTagPlaceholderValue];

    const count = Array.isArray(value) ? value.length : value ? 1 : 0;
    const isPlural = count > 1;
    const itemLabel = itemName ? `${itemName}${isPlural ? 's' : ''}` : 'selected';
    const placeholder = count === optionsCount ? 'All' : `${count} ${itemLabel}`;
    return isOpen ? [0, ''] : [0, placeholder];
  }, [disableTags, isOpen, itemName, optionsCount, maxTagPlaceholderValue, value]);

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
      <Select
        allowClear={allowClear}
        autoClearSearchValue={autoClearSearchValue}
        defaultValue={defaultValue}
        disabled={disabled}
        dropdownMatchSelectWidth={250}
        filterOption={enableSearchFilter ? handleFilter : true}
        filterSort={filterSort}
        id={id}
        maxTagCount={maxTagCount}
        maxTagPlaceholder={maxTagPlaceholder}
        mode={mode}
        options={options ? options : undefined}
        placeholder={placeholder}
        ref={ref}
        showSearch={onSearch ? true : false}
        suffixIcon={<Icon name="arrow-down" size="tiny" />}
        value={value}
        onBlur={onBlur}
        onChange={onChange}
        onDeselect={onDeselect}
        onDropdownVisibleChange={handleDropdownVisibleChange}
        onSearch={onSearch}
        onSelect={onSelect}>
        {children}
      </Select>
    </div>
  );
});

export default SelectFilter;
