import { ClearOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import React from 'react';

import css from './FilterCounter.module.scss';

interface Props {
  activeFilterCount: number;
  onReset: () => void;
}

const FilterCounter: React.FC<Props> = ({ activeFilterCount, onReset }: Props) => {
  if (activeFilterCount === 0) return <></>;
  const text = `${activeFilterCount} active filter${activeFilterCount > 1 ? 's' : ''}`;
  return <div>
    <span>{text} </span>
    <div className={css.launchBlock}>
      <Button
        className={css.launchButton}
        onClick={onReset}>Clear Filters</Button>
    </div>
    <ClearOutlined
      title="Clear filters"
      onClick={onReset} />
  </div>;
};

export default FilterCounter;
