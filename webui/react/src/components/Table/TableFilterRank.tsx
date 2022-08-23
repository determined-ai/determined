import { Button, Input, InputNumber, InputRef, Select } from 'antd';
import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import { V1OrderBy } from 'services/api-ts-sdk';

import css from './TableFilterSearch.module.scss';

interface Props extends FilterDropdownProps {
  column?: string;
  columns: string[]
  onReset?: () => void;
  onSet?: (column: string, order: V1OrderBy, rank?: string) => void;
  orderBy?: V1OrderBy;
  rank?: string;
}

const TableFilterRange: React.FC<Props> = ({
  clearFilters,
  column: _column,
  columns,
  confirm,
  onReset,
  onSet,
  orderBy: _orderBy = V1OrderBy.ASC,
  rank: _rank,
  visible,
}: Props) => {
  const [ column, setColumn ] = useState(_column);
  const [ orderBy, setOrderBy ] = useState(_orderBy);
  const [ rank, setRank ] = useState<number>(parseInt(_rank ?? '0'));

  useEffect(() => {
    const rankInt = parseInt(_rank ?? '0');
    setRank(rankInt);
  }, [ _rank ]);
  useEffect(() => {
    setColumn(_column);
  }, [ _column ]);
  useEffect(() => {
    setOrderBy(_orderBy);
  }, [ _orderBy ]);

  const handleReset = useCallback(() => {
    setColumn('searcherMetricValue');
    setOrderBy(V1OrderBy.ASC);
    setRank(0);
    if (onReset) onReset();
    if (clearFilters) clearFilters();
  }, [ clearFilters, onReset ]);

  const handleApply = useCallback(() => {
    if (column && orderBy && rank)
      onSet?.(column, orderBy, String(rank));
    confirm();
  }, [ confirm, onSet, column, orderBy, rank ]);

  // useEffect(() => {
  //   if (!visible) return;

  //   setTimeout(() => {
  //     inputMinRef.current?.focus({ cursor: 'all' });
  //   }, 0);
  // }, [ visible ]);

  return (
    <div className={css.base}>
      <div className={css.search}>
        <Select
          placeholder="select rank column"
          value={column}
          onChange={(newColumn) => setColumn(newColumn)}>
          {columns.map((column) => (
            <Select.Option key={column} value={column}>
              {column}
            </Select.Option>
          )) ?? []}
        </Select>
        <Select
          placeholder="Select Rank Order"
          value={orderBy}
          onChange={(newOrderBy) => setOrderBy(newOrderBy)}>
          <Select.Option key={V1OrderBy.ASC} value={V1OrderBy.ASC}>
            Ascending
          </Select.Option>
          <Select.Option key={V1OrderBy.DESC} value={V1OrderBy.DESC}>
            Descending
          </Select.Option>
        </Select>
        <InputNumber min={0} precision={0} value={rank} onChange={(newRank) => setRank(newRank)} />
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
          onClick={handleApply}>
          Apply
        </Button>
      </div>
    </div>
  );
};

export default TableFilterRange;
