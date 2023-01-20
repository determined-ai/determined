import React from 'react';

import Button from 'components/kit/Button';
interface Props {
  activeFilterCount: number;
  onReset: () => void;
}

const FilterCounter: React.FC<Props> = ({ activeFilterCount, onReset }: Props) => {
  if (activeFilterCount === 0) return <></>;
  return <Button onClick={onReset}>Clear Filters ({activeFilterCount})</Button>;
};

export default FilterCounter;
