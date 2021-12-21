import { boolean, number, select, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import SkeletonChart from './SkeletonChart';

export default {
  component: SkeletonChart,
  decorators: [ withKnobs ],
  parameters: { layout: 'fullscreen' },
  title: 'Skeleton/SkeletonChart',
};

export const Default = (): React.ReactNode => <SkeletonChart />;

export const WithTitleAndFilters = (): React.ReactNode => (
  <SkeletonChart filters={number('number of filters', 2)} title={boolean('show title', true)} />
);

export const Custom = (): React.ReactNode => {
  return (
    <SkeletonChart size={select('size', {
      '': undefined,
      'large': 'large',
      'medium': 'medium',
      'small': 'small',
    }, undefined)}
    />
  );
};
