import { ClearOutlined } from '@ant-design/icons';
import { Badge, Button } from 'antd';
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
  return <div className={css.base}>
    <div className={css.expanded}>
      <span>{activeFilterCount} active {phrase} </span>
      <Button
        className={css.launchButton}
        onClick={onReset}>Clear Filters</Button>
    </div>
    <div className={css.collapsed}>
      <Badge count={activeFilterCount}>
        <ClearOutlined
          className={css.launchButton}
          title={`Clear ${activeFilterCount} ${capitalize(phrase)}`}
          onClick={onReset} />
      </Badge>
    </div>
  </div>;
};

export default FilterCounter;
