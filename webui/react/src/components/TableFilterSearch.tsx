import { Button, Input } from 'antd';
import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useState } from 'react';

import Icon from './Icon';
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
}: Props) => {
  const [ search, setSearch ] = useState(value);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value || '');
  }, []);

  const handleReset = useCallback(() => {
    setSearch('');
    if (onReset) onReset();
    if (clearFilters) clearFilters();
  }, [ clearFilters, onReset ]);

  const handleSearch = useCallback(() => {
    if (onSearch) onSearch(search);
    confirm();
  }, [ confirm, onSearch, search ]);

  return (
    <div className={css.base}>
      <div className={css.search}>
        <Input
          allowClear
          bordered={false}
          placeholder="search"
          prefix={<Icon name="search" size="tiny" />}
          value={search}
          onChange={handleSearchChange}
        />
      </div>
      <div className={css.footer}>
        <Button
          aria-label="Reset Search"
          size="small"
          type="link"
          onClick={handleReset}>Reset</Button>
        <Button
          aria-label="Apply Search"
          size="small"
          type="primary"
          onClick={handleSearch}>Search</Button>
      </div>
    </div>
  );
};

export default TableFilterSearch;
