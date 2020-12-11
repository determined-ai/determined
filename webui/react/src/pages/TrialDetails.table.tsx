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
    key: 'batches',
    render: batchRender,
    sorter: (a: Step, b: Step): number => alphanumericSorter(
      a.numBatches + a.priorBatchesProcessed,
      b.numBatches + b.priorBatchesProcessed,
    ),
    title: 'Batches',
  },
  {
    key: 'state',
    render: stateRenderer,
    sorter: (a: Step, b: Step): number => runStateSorter(a.state, b.state),
    title: 'State',
  },
  {
    key: 'checkpoint',
    title: 'Checkpoint',
  },
];
