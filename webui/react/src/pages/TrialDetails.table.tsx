import { ColumnType } from 'antd/es/table';
import React from 'react';

import { Renderer, stateRenderer } from 'components/Table';
import { Step2 } from 'types';
import { alphanumericSorter, runStateSorter } from 'utils/data';

const batchRender: Renderer<Step2> = (_, record) => {
  return <>{record.batchNum}</>;
};

export const columns: ColumnType<Step2>[] = [
  {
    key: 'batches',
    render: batchRender,
    sorter: (a: Step2, b: Step2): number => {
      return alphanumericSorter(a.batchNum, b.batchNum);
    },
    title: 'Batches',
  },
  {
    key: 'state',
    render: (text: string, record: Step2, index: number): React.ReactNode =>
      stateRenderer(text, record.training, index),
    sorter: (a: Step2, b: Step2): number => {
      // Previously the list of steps would list steps by training steps which
      // all have RunState type as their state.
      return runStateSorter(a.training.state, b.training.state);
    },
    title: 'State',
  },
  {
    key: 'checkpoint',
    title: 'Checkpoint',
  },
];
