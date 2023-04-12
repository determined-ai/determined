import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FixedSizeList, ListChildComponentProps } from 'react-window';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import Input, { InputRef } from 'components/kit/Input';
import usePrevious from 'shared/hooks/usePrevious';

import css from './TableFilterMultiSearch.module.scss';

export const nonDigits = /\D/;

interface Props extends FilterDropdownProps {
  multiple?: boolean;
  onFilter?: (keys: string[]) => void;
  onReset?: () => void;
  placeholder?: string;
  searchable?: boolean;
  validatorRegex?: RegExp;
  values?: string[];
  width?: number;
}

export const ARIA_LABEL_CONTAINER = 'table-filter-dropdown-container';
export const ARIA_LABEL_INPUT = 'table-filter-dropdown-input';
export const ARIA_LABEL_RESET = 'table-filter-reset';
export const ARIA_LABEL_APPLY = 'table-filter-apply';

const ITEM_HEIGHT = 28;

const TableFilterDropdown: React.FC<Props> = ({
  clearFilters,
  confirm,
  filters,
  multiple,
  onFilter,
  onReset,
  placeholder,
  searchable,
  validatorRegex,
  values = [],
  visible,
  width = 160,
}: Props) => {
  const inputRef = useRef<InputRef>(null);
  const [search, setSearch] = useState('');
  const [selectedMap, setSelectedMap] = useState<Record<string, boolean>>({});
  const prevVisible = usePrevious(visible, undefined);

  const filteredOptions = useMemo(() => {
    if (validatorRegex) {
      // we are not searching when we supply a regex
      // instead, return the currently active filters
      return Object.keys(selectedMap).map((v) => ({ text: v, value: v }));
    }
    const searchString = search.toLocaleLowerCase();
    return (filters || []).filter((filter) => {
      return (
        filter.value?.toString().toLocaleLowerCase().includes(searchString) ||
        filter.text?.toString().toLocaleLowerCase().includes(searchString)
      );
    });
  }, [filters, search, validatorRegex, selectedMap]);

  const listHeight = useMemo(() => {
    if (filteredOptions.length < 10) return ITEM_HEIGHT * filteredOptions.length;
    return ITEM_HEIGHT * 9;
  }, [filteredOptions.length]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleOptionSelect = useCallback(
    (value: string, metaKey?: boolean) => {
      setSelectedMap((prev) => {
        if (multiple) {
          // Support for using CMD + Click to select every option EXCEPT the selected option.
          if (metaKey && filters) {
            return filters.reduce((acc, filter) => {
              if (filter.value !== value) acc[filter.value as string] = true;
              return acc;
            }, {} as Record<string, boolean>);
          } else {
            const newMap = { ...prev };
            if (newMap[value]) delete newMap[value];
            else newMap[value] = true;
            return newMap;
          }
        }
        return prev[value] ? {} : { [value]: true };
      });
    },
    [filters, multiple],
  );

  const handleOptionClick = useCallback(
    (e: React.MouseEvent) => {
      const value = (e.target as HTMLDivElement).getAttribute('data-value');
      if (value) handleOptionSelect(value, e.metaKey);
    },
    [handleOptionSelect],
  );

  const handleReset = useCallback(() => {
    setSelectedMap({});
    if (onReset) onReset();
    if (clearFilters) clearFilters();
  }, [clearFilters, onReset]);

  const handleConfirm = useCallback(() => {
    const filters = Object.keys(selectedMap);
    onFilter?.(filters);
    confirm();
  }, [confirm, onFilter, selectedMap]);

  const handlePressEnter = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      e.stopPropagation();
      if (!inputRef.current?.input?.value) handleConfirm();
      if (validatorRegex) {
        const validatedInput = inputRef.current?.input?.value?.replace(validatorRegex, '');
        if (validatedInput) {
          setSelectedMap((m) => ({ ...m, [validatedInput]: true }));
          setSearch('');
        }
      } else {
        if (filteredOptions.length) {
          handleOptionSelect(filteredOptions[0].value as string, e.metaKey);
        }
      }
    },
    [validatorRegex, filteredOptions, handleOptionSelect, handleConfirm],
  );

  const OptionRow: React.FC<ListChildComponentProps> = useCallback(
    ({ data, index, style }) => {
      const classes = [css.option];
      const isSelected = validatorRegex || selectedMap[data[index].value];
      const isJSX = typeof data[index].text !== 'string';
      if (isSelected) classes.push(css.selected);
      return (
        <div
          className={classes.join(' ')}
          data-value={data[index].value}
          style={style}
          onClick={handleOptionClick}>
          {isJSX ? data[index].text : <span>{data[index].text}</span>}
          <Icon name="checkmark" />
        </div>
      );
    },
    [handleOptionClick, selectedMap, validatorRegex],
  );

  /*
   * Detect when filter dropdown is being shown and
   * proceed to initialize the selected map of which
   * options are selected.
   */
  useEffect(() => {
    if (prevVisible !== visible && visible) {
      setSearch('');

      const valuesAsList = Array.isArray(values) ? values : [values];
      setSelectedMap(
        valuesAsList.reduce((acc, value) => {
          acc[value] = true;
          return acc;
        }, {} as Record<string, boolean>),
      );

      setTimeout(() => {
        if (inputRef.current) inputRef.current.focus({ cursor: 'all' });
      }, 0);
    }
  }, [prevVisible, values, visible]);

  return (
    <div aria-label={ARIA_LABEL_CONTAINER} className={css.base} style={{ width }}>
      {searchable && (
        <div className={css.search}>
          <Input
            allowClear
            aria-label={ARIA_LABEL_INPUT}
            bordered={false}
            placeholder={placeholder ?? 'search filters'}
            prefix={<Icon name="search" size="tiny" />}
            ref={inputRef}
            value={search}
            onChange={handleSearchChange}
            onPressEnter={handlePressEnter}
          />
        </div>
      )}
      <FixedSizeList
        height={listHeight}
        itemCount={filteredOptions.length}
        itemData={filteredOptions}
        itemSize={ITEM_HEIGHT}
        width="100%">
        {OptionRow}
      </FixedSizeList>
      <div className={css.footer}>
        <Button
          aria-label={ARIA_LABEL_RESET}
          disabled={Object.keys(selectedMap).length === 0}
          size="small"
          type="link"
          onClick={handleReset}>
          Reset
        </Button>
        <Button aria-label={ARIA_LABEL_APPLY} size="small" type="primary" onClick={handleConfirm}>
          Ok
        </Button>
      </div>
    </div>
  );
};

export default TableFilterDropdown;
