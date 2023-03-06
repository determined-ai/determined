import { Select } from 'antd';
import { FilterDropdownProps } from 'antd/es/table/interface';
import React, { useCallback, useEffect, useState } from 'react';

import Button from 'components/kit/Button';
import InputNumber from 'components/kit/InputNumber';

import css from './TableFilterRank.module.scss';

interface Props extends FilterDropdownProps {
  column?: string;
  columns: string[];
  onReset?: () => void;
  onSet?: (column: string, sortDesc: boolean, rank?: string) => void;
  rank?: string;
  sortDesc?: boolean;
}

const TableFilterRange: React.FC<Props> = ({
  clearFilters,
  column: _column,
  columns,
  confirm,
  onReset,
  onSet,
  sortDesc: _sortDesc = false,
  rank: _rank,
}: Props) => {
  const [column, setColumn] = useState(_column);
  const [sortDesc, setSortDesc] = useState(_sortDesc);
  const [rank, setRank] = useState<number | undefined>(parseInt(_rank ?? '0') || undefined);

  useEffect(() => {
    const rankInt = parseInt(_rank ?? '0') || undefined;
    setRank(rankInt);
  }, [_rank]);
  useEffect(() => {
    setColumn(_column);
  }, [_column]);
  useEffect(() => {
    setSortDesc(_sortDesc);
  }, [_sortDesc]);

  const handleReset = useCallback(() => {
    setColumn('searcherMetricValue');
    setSortDesc(false);
    setRank(undefined);
    if (onReset) onReset();
    if (clearFilters) clearFilters();
  }, [clearFilters, onReset]);

  const handleApply = useCallback(() => {
    if (column && sortDesc != null && rank != null) onSet?.(column, sortDesc, String(rank));
    confirm();
  }, [confirm, onSet, column, sortDesc, rank]);

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
          value={sortDesc}
          onChange={(newOrderBy) => setSortDesc(newOrderBy)}>
          <Select.Option key="false" value={false}>
            {column === 'searcherMetricValue' ? 'Best' : 'Ascending'}
          </Select.Option>
          <Select.Option key="true" value={true}>
            {column === 'searcherMetricValue' ? 'Worst' : 'Descending'}
          </Select.Option>
        </Select>
        <InputNumber
          min={0}
          precision={0}
          value={rank}
          onChange={(newRank) => {
            setRank(newRank ? (newRank as number) : undefined);
          }}
        />
      </div>
      <div className={css.footer}>
        <Button aria-label="Reset Search" size="small" type="link" onClick={handleReset}>
          Reset
        </Button>
        <Button aria-label="Apply Search" size="small" type="primary" onClick={handleApply}>
          Apply
        </Button>
      </div>
    </div>
  );
};

export default TableFilterRange;
