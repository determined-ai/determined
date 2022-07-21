import { Button, Input } from 'antd';
import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import Icon from '../shared/components/Icon/Icon';

import css from './TableFilterSearch.module.scss';

interface Props extends FilterDropdownProps {
  onReset?: () => void;
  onSet?: (min: string, max: string) => void;
  min: string;
  max: string;
}

const TableFilterRange: React.FC<Props> = ({
  clearFilters,
  confirm,
  onReset,
  onSet,
  min,
  max,
  visible,
}: Props) => {
  const inputMinRef = useRef<Input>(null);
  const inputMaxRef = useRef<Input>(null);
  const [ inputMin, setInputMin ] = useState(min);
  const [ inputMax, setInputMax ] = useState(max);

  console.log(min, inputMin)


  const handleMinChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setInputMin(e.target.value || '');
  }, []);
  const handleMaxChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setInputMax(e.target.value || '');
  }, []);

  const handleReset = useCallback(() => {
    setInputMin('')
    setInputMax('')
    if (onReset) onReset();
    if (clearFilters) clearFilters();
  }, [ clearFilters, onReset ]);

  const handleSearch = useCallback(() => {
    onSet?.(inputMin, inputMax);
    confirm();
  }, [ confirm, onSet, inputMin, inputMax ]);

  useEffect(() => {
    if (!visible) return;

    setTimeout(() => {
      inputMinRef.current?.focus({ cursor: 'all' });
    }, 0);
  }, [ visible ]);

  return (
    <div className={css.base}>
      <div className={css.search}>
        <Input
          allowClear
          bordered={false}
          placeholder="min"
          // prefix={<Icon name="search" size="tiny" />}
          ref={inputMinRef}
          value={inputMin}
          onChange={handleMinChange}
          onPressEnter={() => inputMaxRef.current?.focus({ cursor: 'all' })}
        />
        <Input
          allowClear
          bordered={false}
          placeholder="max"
          // prefix={<Icon name="search" size="tiny" />}
          ref={inputMaxRef}
          value={inputMax}
          onChange={handleMaxChange}
          onPressEnter={handleSearch}
        />
      </div>
      <div className={css.footer}>
        <Button
          aria-label="Reset Search"
          size="small"
          type="link"
          onClick={handleReset}>
          Reset
        </Button>
        <Button
          aria-label="Apply Search"
          size="small"
          type="primary"
          onClick={handleSearch}>
          Search
        </Button>
      </div>
    </div>
  );
};

export default TableFilterRange;
