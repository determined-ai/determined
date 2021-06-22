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
  return <div className={css.base}>
    <span>{text} </span>
    <Button
      className={css.launchButton}
      onClick={onReset}>Clear Filters</Button>
  </div>;
};

export default FilterCounter;
