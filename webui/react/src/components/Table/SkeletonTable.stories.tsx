import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import SkeletonTable from './SkeletonTable';

export default {
  component: SkeletonTable,
  parameters: { layout: 'fullscreen' },
  title: 'Determined/Skeleton/SkeletonTable',
} as Meta<typeof SkeletonTable>;

export const Default = (): React.ReactNode => <SkeletonTable />;

export const WithVariableColumns = (): React.ReactNode => (
  <SkeletonTable columns={[{ flexGrow: 0.5 }, { flexGrow: 4 }, { flexGrow: 2 }, { flexGrow: 1 }]} />
);

export const Custom: ComponentStory<typeof SkeletonTable> = (args) => {
  return <SkeletonTable {...args} />;
};

Custom.args = {
  columns: 10,
  rows: 10,
  title: true,
};
