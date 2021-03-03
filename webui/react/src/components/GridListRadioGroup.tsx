import { Radio } from 'antd';
import { RadioChangeEvent } from 'antd/lib/radio';
import React, { useCallback, useState } from 'react';

import Icon from 'components/Icon';

import css from './GridListRadioGroup.module.scss';

export enum GridListView {
  Grid,
  List,
}

interface Props {
  onChange?: (e: GridListView) => void;
  value: GridListView;
}

const GridListRadioGroup: React.FC<Props> = ({ onChange, value }: Props) => {
  const [ view, setView ] = useState(value);

  const handleChange = useCallback((e: RadioChangeEvent) => {
    if (onChange) onChange(e.target.value as GridListView);
    setView(e.target.value as GridListView);
  }, [ onChange ]);

  return (
    <Radio.Group className={css.base} value={view} onChange={handleChange}>
      <Radio.Button className={css.option} value={GridListView.Grid}>
        <Icon name="grid" size="large" title="Card View" />
      </Radio.Button>
      <Radio.Button className={css.option} value={GridListView.List}>
        <Icon name="list" size="large" title="List View" />
      </Radio.Button>
    </Radio.Group>
  );
};

export default GridListRadioGroup;
