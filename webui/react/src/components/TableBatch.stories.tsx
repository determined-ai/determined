import { Button } from 'antd';
import React from 'react';

import TableBatch from './TableBatch';

export default {
  component: TableBatch,
  title: 'TableBatch',
};

export const SampleTableBatch = (): React.ReactNode => {
  return (
    <TableBatch message="Apply batch operations to multiple items.">
      <Button danger type="primary">Batch Operation 1</Button>
      <Button>Batch Operation 2</Button>
      <Button>Batch Operation 3</Button>
    </TableBatch>
  );
};
