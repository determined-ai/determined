import React, { useCallback } from 'react';

import RadioGroup from 'components/RadioGroup';
import { ValueOf } from 'types';

export const GridListView = {
  Grid: 'grid',
  List: 'list',
} as const;

export type GridListView = ValueOf<typeof GridListView>;

interface Props {
  onChange?: (view: GridListView) => void;
  value: GridListView;
}

const GridListRadioGroup: React.FC<Props> = ({ onChange, value }: Props) => {
  const handleChange = useCallback(
    (id: string) => {
      if (onChange) onChange(id as GridListView);
    },
    [onChange],
  );

  return (
    <RadioGroup
      iconOnly
      options={[
        { icon: 'grid', id: GridListView.Grid, label: 'Grid View' },
        { icon: 'list', id: GridListView.List, label: 'List View' },
      ]}
      value={value}
      onChange={handleChange}
    />
  );
};

export default GridListRadioGroup;
