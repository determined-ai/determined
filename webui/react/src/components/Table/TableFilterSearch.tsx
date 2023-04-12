import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Button from 'components/kit/Button';
import Icon from 'components/kit/Icon';
import Input, { InputRef } from 'components/kit/Input';

import css from './TableFilterSearch.module.scss';

interface Props extends FilterDropdownProps {
  onReset?: () => void;
  onSearch?: (search: string) => void;
  value: string;
}

const TableFilterSearch: React.FC<Props> = ({
  clearFilters,
  confirm,
  onReset,
  onSearch,
  value,
  visible,
}: Props) => {
  const inputRef = useRef<InputRef>(null);
  const [search, setSearch] = useState(value);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleReset = useCallback(() => {
    setSearch('');
    if (onReset) onReset();
    if (clearFilters) clearFilters();
  }, [clearFilters, onReset]);

  const handleSearch = useCallback(() => {
    if (onSearch) onSearch(search);
    confirm();
  }, [confirm, onSearch, search]);

  useEffect(() => {
    if (!visible) return;

    setTimeout(() => {
      if (inputRef.current) inputRef.current.focus({ cursor: 'all' });
    }, 0);
  }, [visible]);

  return (
    <div className={css.base}>
      <div className={css.search}>
        <Input
          allowClear
          bordered={false}
          placeholder="search"
          prefix={<Icon name="search" size="tiny" />}
          ref={inputRef}
          value={search}
          onChange={handleSearchChange}
          onPressEnter={handleSearch}
        />
      </div>
      <div className={css.footer}>
        <Button aria-label="Reset Search" size="small" type="link" onClick={handleReset}>
          Reset
        </Button>
        <Button aria-label="Apply Search" size="small" type="primary" onClick={handleSearch}>
          Search
        </Button>
      </div>
    </div>
  );
};

export default TableFilterSearch;
