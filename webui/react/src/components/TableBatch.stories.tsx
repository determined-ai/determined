import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import TableBatch from './TableBatch';

export default {
  component: TableBatch,
  parameters: { layout: 'padded' },
  title: 'Determined/TableBatch',
} as Meta<typeof TableBatch>;

const batchOptions = [
  { label: 'Batch Operation 1', value: 'Action1' },
  { label: 'Batch Operation 2', value: 'Action2' },
  { label: 'Batch Operation 3', value: 'Action3' },
];

export const Default: ComponentStory<typeof TableBatch> = (args) => (
  <TableBatch actions={batchOptions} {...args} />
);

Default.args = { selectedRowCount: 1 };
