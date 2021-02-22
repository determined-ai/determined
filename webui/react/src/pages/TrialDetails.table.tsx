import { ColumnType } from 'antd/es/table';
import React from 'react';

import { Renderer, stateRenderer } from 'components/Table';
import { Step } from 'types';
import { numericSorter, runStateSorter } from 'utils/data';

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
    key: 'state',
    render: (text: string, record: Step, index: number): React.ReactNode =>
      stateRenderer(text, record.training, index),
    sorter: (a: Step, b: Step): number => {
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
