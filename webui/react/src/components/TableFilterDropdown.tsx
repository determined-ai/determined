import { Button, Input } from 'antd';
import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { FixedSizeList, ListChildComponentProps } from 'react-window';

import usePrevious from 'hooks/usePrevious';

import Icon from './Icon';
import css from './TableFilterDropdown.module.scss';

interface Props extends FilterDropdownProps {
  onFilter?: (keys: string[]) => void;
  onReset?: () => void;
  searchable?: boolean;
  values?: string[];
  width?: number;
}

const ITEM_HEIGHT = 28;

const TableFilterDropdown: React.FC<Props> = ({
  clearFilters,
  confirm,
  filters,
  onFilter,
  onReset,
  searchable,
  values = [],
  visible,
  width = 160,
}: Props) => {
  const [ search, setSearch ] = useState('');
  const [ selectedMap, setSelectedMap ] = useState<Record<string, boolean>>({});
  const prevVisible = usePrevious(visible, undefined);

  const filteredOptions = useMemo(() => {
    const searchString = search.toLocaleLowerCase();
    return (filters || []).filter(filter => {
      return filter.value?.toString().toLocaleLowerCase().includes(searchString);
    });
  }, [ filters, search ]);

  const listHeight = useMemo(() => {
    if (filteredOptions.length < 10) return ITEM_HEIGHT * filteredOptions.length;
    return ITEM_HEIGHT * 9;
  }, [ filteredOptions.length ]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleOptionClick = useCallback((e: React.MouseEvent) => {
    const value = (e.target as HTMLDivElement).getAttribute('data-value');
    if (!value) return;

    setSelectedMap(prev => {
      const newMap = { ...prev };
      if (newMap[value]) delete newMap[value];
      else newMap[value] = true;
      return newMap;
    });
  }, []);

  const handleReset = useCallback(() => {
    setSelectedMap({});
    if (onReset) onReset();
    if (clearFilters) clearFilters();
  }, [ clearFilters, onReset ]);

  const handleFilter = useCallback(() => {
    if (onFilter) onFilter(Object.keys(selectedMap));
    confirm();
  }, [ confirm, onFilter, selectedMap ]);

  const OptionRow: React.FC<ListChildComponentProps> = useCallback(({ data, index, style }) => {
    const classes = [ css.option ];
    const isSelected = selectedMap[data[index].value];
    if (isSelected) classes.push(css.selected);
    return (
      <div
        className={classes.join(' ')}
        data-value={data[index].value}
        style={style}
        onClick={handleOptionClick}>
        <span>{data[index].text}</span>
        <Icon name="checkmark" />
      </div>
    );
  }, [ handleOptionClick, selectedMap ]);

  /*
   * Detect when filter dropdown is being shown and
   * proceed to initialize the selected map of which
   * options are selected.
   */
  useEffect(() => {
    if (prevVisible !== visible && visible) {
      setSearch('');
      setSelectedMap(values.reduce((acc, value) => {
        acc[value] = true;
        return acc;
      }, {} as Record<string, boolean>));
    }
  }, [ prevVisible, values, visible ]);

  return (
    <div className={css.base} style={{ width }}>
      {searchable && (
        <div className={css.search}>
          <Input
            allowClear
            bordered={false}
            placeholder="search filters"
            prefix={<Icon name="search" size="tiny" />}
            value={search}
            onChange={handleSearchChange}
          />
        </div>
      )}
      <FixedSizeList
        height={listHeight}
        itemCount={filteredOptions.length}
        itemData={filteredOptions}
        itemSize={ITEM_HEIGHT}
        width='100%'>
        {OptionRow}
      </FixedSizeList>
      <div className={css.footer}>
        <Button
          aria-label="Reset Filter"
          disabled={Object.keys(selectedMap).length === 0}
          size="small"
          type="link"
          onClick={handleReset}>Reset</Button>
        <Button
          aria-label="Apply Filter"
          size="small"
          type="primary"
          onClick={handleFilter}>Ok</Button>
      </div>
    </div>
  );
};

export default TableFilterDropdown;
