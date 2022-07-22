import { number, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import TableBatch from './TableBatch';

export default {
  component: TableBatch,
  decorators: [ withKnobs ],
  parameters: { layout: 'padded' },
  title: 'TableBatch',
};

const batchOptions = [
  { label: 'Batch Operation 1', value: 'Action1' },
  { label: 'Batch Operation 2', value: 'Action2' },
  { label: 'Batch Operation 3', value: 'Action3' },
];

export const Default = (): React.ReactNode => (
  <TableBatch
    actions={batchOptions}
    selectedRowCount={1}
  />
);

export const Custom = (): React.ReactNode => (
  <TableBatch
    actions={batchOptions}
    selectedRowCount={number('selectedRowCount', 1)}
  />
);
