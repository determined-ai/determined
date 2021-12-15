import { Button } from 'antd';
import React from 'react';

import { capitalize } from 'utils/string';

import css from './FilterCounter.module.scss';

interface Props {
  activeFilterCount: number;
  onReset: () => void;
}

const FilterCounter: React.FC<Props> = ({ activeFilterCount, onReset }: Props) => {
  if (activeFilterCount === 0) return <></>;
  const phrase = `filter${activeFilterCount > 1 ? 's' : ''}`;
  return (
    <div className={css.base}>
      <div className={css.expanded}>
        <span>{activeFilterCount} active {phrase} </span>
        <Button
          className={css.launchButton}
          onClick={onReset}>Clear Filters
        </Button>
      </div>
      <div className={css.collapsed}>
        <Button
          className={css.launchButton}
          onClick={onReset}>Clear {activeFilterCount} {capitalize(phrase)}
        </Button>
      </div>
    </div>
  );
};

export default FilterCounter;
