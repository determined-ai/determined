import { Radio } from 'antd';
import React, { useCallback, useState } from 'react';

import Icon from 'shared/components/Icon';

import RadioGroup from './RadioGroup';
import css from './RadioGroup.stories.module.scss';

export default {
  component: RadioGroup,
  parameters: { layout: 'centered' },
  title: 'Determined/Buttons/RadioGroup',
};

const DEFAULT_OPTIONS = [
  { icon: 'learning', id: 'learning-curve', label: 'Learning Curve' },
  { icon: 'parcoords', id: 'parcoords', label: 'Parallel Coordinates' },
  { icon: 'scatter-plot', id: 'scatter-plots', label: 'Scatter Plots' },
  { icon: 'heat', id: 'heatmap', label: 'Heat Map' },
];
const ICON_ONLY_OPTIONS = DEFAULT_OPTIONS.map((option) => ({ ...option, label: undefined }));
const LABELS_ONLY_OPTIONS = DEFAULT_OPTIONS.map((option) => ({ ...option, icon: undefined }));

export const Default = (): React.ReactNode => {
  const [value, setValue] = useState(DEFAULT_OPTIONS[0].id);
  const handleChange = useCallback((value: string) => setValue(value), []);
  return <RadioGroup options={DEFAULT_OPTIONS} value={value} onChange={handleChange} />;
};

export const IconsOnly = (): React.ReactNode => {
  const [value, setValue] = useState(ICON_ONLY_OPTIONS[0].id);
  const handleChange = useCallback((value: string) => setValue(value), []);
  return <RadioGroup options={ICON_ONLY_OPTIONS} value={value} onChange={handleChange} />;
};

export const LabelsOnly = (): React.ReactNode => {
  const [value, setValue] = useState(LABELS_ONLY_OPTIONS[0].id);
  const handleChange = useCallback((value: string) => setValue(value), []);
  return <RadioGroup options={LABELS_ONLY_OPTIONS} value={value} onChange={handleChange} />;
};

export const Large = (): React.ReactNode => {
  return (
    <div className={css.base}>
      <Radio.Group className={css.largeGroup} optionType="button">
        {new Array(3).fill(0).map((_item, idx) => (
          <Radio.Button key={idx} value={idx}>
            <div className={css.radioButton}>
              <Icon />
              <p>Option {idx + 1}</p>
            </div>
          </Radio.Button>
        ))}
      </Radio.Group>
    </div>
  );
};
