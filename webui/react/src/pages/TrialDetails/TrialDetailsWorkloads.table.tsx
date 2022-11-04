import { ColumnType } from 'antd/es/table';
import React from 'react';

import { Renderer } from 'components/Table/Table';
import { numericSorter } from 'shared/utils/sort';
import { Step } from 'types';

const batchRender: Renderer<Step> = (_, record) => {
  return <>{record.batchNum}</>;
};

export const columns: ColumnType<Step>[] = [
  {
    key: 'batches',
    render: batchRender,
    sorter: (a: Step, b: Step): number => {
      return numericSorter(a.batchNum, b.batchNum);
    },
    title: 'Batches',
  },
  {
    key: 'checkpoint',
    title: 'Checkpoint',
  },
];
