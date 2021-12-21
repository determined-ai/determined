import { boolean, number, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import SkeletonTable from './SkeletonTable';

export default {
  component: SkeletonTable,
  decorators: [ withKnobs ],
  parameters: { layout: 'fullscreen' },
  title: 'Skeleton/SkeletonTable',
};

export const Default = (): React.ReactNode => <SkeletonTable />;

export const WithVariableColumns = (): React.ReactNode => (
  <SkeletonTable
    columns={[
      { flexGrow: 0.5 },
      { flexGrow: 4 },
      { flexGrow: 2 },
      { flexGrow: 1 },
    ]}
  />
);

export const WithTitleAndFilters = (): React.ReactNode => (
  <SkeletonTable filters={number('number of filters', 2)} title={boolean('show title', true)} />
);

export const Custom = (): React.ReactNode => {
  return (
    <SkeletonTable
      columns={number('columns', 10)}
      rows={number('rows', 10)}
    />
  );
};
