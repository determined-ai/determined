import React, { useCallback, useState } from 'react';

import RadioGroup from './RadioGroup';

export default {
  component: RadioGroup,
  parameters: { layout: 'centered' },
  title: 'RadioGroup',
};

const DEFAULT_OPTIONS = [
  { icon: 'learning', id: 'learning-curve', tag: 'Learning Curve' },
  { icon: 'parcoords', id: 'parcoords', tag: 'Parallel Coordinates' },
  { icon: 'scatter-plot', id: 'scatter-plots', tag: 'Scatter Plots' },
  { icon: 'heat', id: 'heatmap', tag: 'Heat Map' },
];
const ICON_ONLY_OPTIONS = DEFAULT_OPTIONS.map(option => ({ ...option, tag: undefined }));
const LABELS_ONLY_OPTIONS = DEFAULT_OPTIONS.map(option => ({ ...option, icon: undefined }));

export const Default = (): React.ReactNode => {
  const [ value, setValue ] = useState(DEFAULT_OPTIONS[0].id);
  const handleChange = useCallback((value: string) => setValue(value), []);
  return <RadioGroup options={DEFAULT_OPTIONS} value={value} onChange={handleChange} />;
};

export const IconsOnly = (): React.ReactNode => {
  const [ value, setValue ] = useState(ICON_ONLY_OPTIONS[0].id);
  const handleChange = useCallback((value: string) => setValue(value), []);
  return <RadioGroup options={ICON_ONLY_OPTIONS} value={value} onChange={handleChange} />;
};

export const LabelsOnly = (): React.ReactNode => {
  const [ value, setValue ] = useState(LABELS_ONLY_OPTIONS[0].id);
  const handleChange = useCallback((value: string) => setValue(value), []);
  return <RadioGroup options={LABELS_ONLY_OPTIONS} value={value} onChange={handleChange} />;
};
