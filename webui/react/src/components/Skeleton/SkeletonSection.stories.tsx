import { ComponentStory, Meta } from '@storybook/react';
import { Skeleton } from 'antd';
import React from 'react';

import SkeletonSection, { ContentType } from './SkeletonSection';

export default {
  argTypes: {
    contentType: { control: { options: [...Object.keys(ContentType), undefined] } },
    size: { control: { options: ['small', 'medium', 'large', 'max'], type: 'select' } },
  },
  component: SkeletonSection,
  parameters: { layout: 'fullscreen' },
  title: 'Determined/Skeleton/SkeletonSection',
} as Meta<typeof SkeletonSection>;

export const Default = (): React.ReactNode => <SkeletonSection />;

export const WithTitleAndFilters = (): React.ReactNode => <SkeletonSection filters title />;

export const WithTitleProperties = (): React.ReactNode => (
  <SkeletonSection title={{ style: { background: 'red' }, width: 150 }} />
);

export const WithFilterProperties = (): React.ReactNode => (
  <SkeletonSection filters={[{ width: 100 }, { width: 200 }, { width: 300 }]} title />
);

export const WithCustomChildren = (): React.ReactNode => (
  <SkeletonSection filters title>
    <Skeleton active />
  </SkeletonSection>
);

export const Custom: ComponentStory<typeof SkeletonSection> = (args) => {
  return (
    <div style={{ height: '100vh' }}>
      <SkeletonSection {...args} />
    </div>
  );
};

Custom.args = { contentType: undefined, filters: 2, size: 'medium', title: true };
