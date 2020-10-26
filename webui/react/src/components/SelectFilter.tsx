import { Select } from 'antd';
import { SelectProps, SelectValue } from 'antd/es/select';
import React, {
  forwardRef,
  PropsWithChildren,
  useCallback,
  useMemo,
  useState,
} from 'react';

import Icon from './Icon';
import Label from './Label';
import css from './SelectFilter.module.scss';

const { OptGroup, Option } = Select;

interface Props<T = SelectValue> extends SelectProps<T> {
  disableTags?: boolean;
  enableSearchFilter?: boolean;
  label: string;
  style?: React.CSSProperties;
  ref?: React.Ref<Select<SelectValue>>;
}

export interface SelectFilterHandles {
  blur: () => void;
}

export const ALL_VALUE = 'all';

const countOptions = (children: React.ReactNode): number => {
  let count = 0;
  if (Array.isArray(children)) {
    count += children.map(child => countOptions(child)).reduce((acc, count) => acc + count, 0);
  }

  const childType = (children as React.ReactElement).type;
  const childProps = (children as React.ReactElement).props;
  const childList = (childProps as React.ReactPortal)?.children;
  if (childType === Option) count++;
  if (childType === OptGroup && childList) count += countOptions(childList);

  return count;
};

const SelectFilter: React.FC<PropsWithChildren<Props>> = forwardRef((
  {
    className = '',
    disableTags = false,
    dropdownMatchSelectWidth = false,
    enableSearchFilter = true,
    showSearch = true,
    ...props
  }: PropsWithChildren<Props>,
  ref?: React.Ref<Select<SelectValue>>,
) => {

  const [ isOpen, setIsOpen ] = useState(false);
  const classes = [ css.base ];

  if (disableTags) classes.push(css.disableTags);

  const optionsCount = useMemo(() => countOptions(props.children), [ props.children ]);

  const [ maxTagCount, maxTagPlaceholder ] = useMemo(() => {
    if (!disableTags) return [ undefined, props.maxTagPlaceholder ];

    const count = Array.isArray(props.value) ? props.value.length : (props.value ? 1 : 0);
    const placeholder = count === optionsCount ? 'All' : `${count} selected`;
    return isOpen ? [ 0, '' ] : [ 0, placeholder ];
  }, [ disableTags, isOpen, optionsCount, props.maxTagPlaceholder, props.value ]);

  const getPopupContainer = useCallback((triggerNode) => triggerNode, []);

  const handleDropdownVisibleChange = useCallback((open: boolean) => {
    setIsOpen(open);
  }, []);

  const handleFilter = useCallback((search: string, option) => {
    /*
     * `option.children` is one of the following:
     * - undefined
     * - string
     * - string[]
     */
    let label = null;
    if (option.children) {
      if (Array.isArray(option.children)) {
        label = option.children.join(' ');
      } else if (option.children.props?.label) {
        label = option.children.props?.label;
      } else if (typeof option.children === 'string') {
        label = option.children;
      }
    }
    return label && label.toLocaleLowerCase().indexOf(search.toLocaleLowerCase()) !== -1;
  }, []);

  return (
    <div className={classes.join(' ')}>
      <Label>{props.label}</Label>
      <Select
        dropdownMatchSelectWidth={dropdownMatchSelectWidth}
        filterOption={enableSearchFilter ? handleFilter : true}
        getPopupContainer={getPopupContainer}
        maxTagCount={maxTagCount}
        maxTagPlaceholder={maxTagPlaceholder}
        ref={ref}
        showSearch={showSearch}
        suffixIcon={<Icon name="arrow-down" size="tiny" />}
        onDropdownVisibleChange={handleDropdownVisibleChange}
        {...props}>
        {props.children}
      </Select>
    </div>
  );
});

export default SelectFilter;
