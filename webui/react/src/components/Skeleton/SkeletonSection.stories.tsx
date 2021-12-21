import { boolean, number, withKnobs } from '@storybook/addon-knobs';
import { Skeleton } from 'antd';
import React from 'react';

import SkeletonSection from './SkeletonSection';

export default {
  component: SkeletonSection,
  decorators: [ withKnobs ],
  parameters: { layout: 'fullscreen' },
  title: 'Skeleton/SkeletonSection',
};

export const Default = (): React.ReactNode => <SkeletonSection />;

export const WithTitleAndFilters = (): React.ReactNode => <SkeletonSection filters title />;

export const WithTitleProperties = (): React.ReactNode => (
  <SkeletonSection title={{ style: { background: 'red' }, width: 150 }} />
);

export const WithFilterProperties = (): React.ReactNode => (
  <SkeletonSection
    filters={[
      { width: 100 },
      { width: 200 },
      { width: 300 },
    ]}
    title
  />
);

export const WithCustomChildren = (): React.ReactNode => (
  <SkeletonSection filters title>
    <Skeleton active />
  </SkeletonSection>
);

export const Custom = (): React.ReactNode => {
  return (
    <SkeletonSection
      filters={number('number of filters', 2)}
      maxHeight={boolean('max height', false)}
      title={boolean('show title', true)}
    />
  );
};
