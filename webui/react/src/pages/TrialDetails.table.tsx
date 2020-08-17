import { ColumnType } from 'antd/es/table';
import React from 'react';

import { Renderer, stateRenderer } from 'components/Table';
import { Step } from 'types';
import { alphanumericSorter, runStateSorter } from 'utils/data';

const batchRender: Renderer<Step> = (_, record) => (
  <>{record.numBatches + record.priorBatchesProcessed}</>
);

export const columns: ColumnType<Step>[] = [
  {
    dataIndex: 'id',
    fixed: 'left',
    sorter: (a: Step, b: Step): number => alphanumericSorter(a.id, b.id),
    title: 'ID',
    width: 80,
  },
  {
    fixed: 'left',
    render: batchRender,
    sorter: (a: Step, b: Step): number => alphanumericSorter(
      a.numBatches + a.priorBatchesProcessed,
      b.numBatches + b.priorBatchesProcessed,
    ),
    title: 'Batches',
    width: 100,
  },
  {
    fixed: 'right',
    render: stateRenderer,
    sorter: (a: Step, b: Step): number => runStateSorter(a.state, b.state),
    title: 'State',
    width: 120,
  },
];
