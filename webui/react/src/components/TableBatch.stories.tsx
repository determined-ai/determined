import { number, withKnobs } from '@storybook/addon-knobs';
import { Button } from 'antd';
import React from 'react';

import TableBatch from './TableBatch';

export default {
  component: TableBatch,
  decorators: [ withKnobs ],
  parameters: { layout: 'padded' },
  title: 'TableBatch',
};

export const Default = (): React.ReactNode => (
  <TableBatch selectedRowCount={1}>
    <Button danger type="primary">Batch Operation 1</Button>
    <Button>Batch Operation 2</Button>
    <Button>Batch Operation 3</Button>
  </TableBatch>
);

export const Custom = (): React.ReactNode => (
  <TableBatch selectedRowCount={number('selectedRowCount', 1)}>
    <Button danger type="primary">Batch Operation 1</Button>
    <Button>Batch Operation 2</Button>
    <Button>Batch Operation 3</Button>
  </TableBatch>
);
