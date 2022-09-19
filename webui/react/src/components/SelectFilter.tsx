import { Select } from 'antd';
import { RefSelectProps, SelectProps, SelectValue } from 'antd/es/select';
import React, {
  forwardRef,
  useCallback,
  useMemo,
  useState,
} from 'react';

import Icon from 'shared/components/Icon/Icon';

import Label, { LabelTypes } from './Label';
import css from './SelectFilter.module.scss';

const { OptGroup, Option } = Select;

export interface Props<T = SelectValue> extends SelectProps<T> {
  disableTags?: boolean;
  enableSearchFilter?: boolean;
  itemName?: string;
  label?: string;
  ref?: React.Ref<RefSelectProps>;
  style?: React.CSSProperties;
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

const SelectFilter: React.FC<Props> = forwardRef(function SelectFilter(
  {
    className = '',
    disableTags = false,
    /*
     * Disabling `dropdownMatchSelectWidth` will disable virtual scroll within the dropdown options.
     * This should only be done if the option count is fairly low.
     */
    dropdownMatchSelectWidth = true,
    enableSearchFilter = true,
    itemName,
    showSearch = true,
    verticalLayout = false,
    ...props
  }: Props,
  ref?: React.Ref<RefSelectProps>,
) {
  const [ isOpen, setIsOpen ] = useState(false);
  const classes = [ className, css.base ];

  if (disableTags) classes.push(css.disableTags);
  if (verticalLayout) classes.push(css.vertical);

  const optionsCount = useMemo(() => countOptions(props.children), [ props.children ]);

  const [ maxTagCount, maxTagPlaceholder ] = useMemo(() => {
    if (!disableTags) return [ undefined, props.maxTagPlaceholder ];

    const count = Array.isArray(props.value) ? props.value.length : (props.value ? 1 : 0);
    const isPlural = count > 1;
    const itemLabel = itemName ? `${itemName}${isPlural ? 's' : ''}` : 'selected';
    const placeholder = count === optionsCount ? 'All' : `${count} ${itemLabel}`;
    return isOpen ? [ 0, '' ] : [ 0, placeholder ];
  }, [ disableTags, isOpen, itemName, optionsCount, props.maxTagPlaceholder, props.value ]);

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
      {props.label && <Label type={LabelTypes.TextOnly}>{props.label}</Label>}
      <Select
        dropdownMatchSelectWidth={dropdownMatchSelectWidth}
        filterOption={enableSearchFilter ? handleFilter : true}
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
