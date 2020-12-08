import { ColumnType } from 'antd/es/table';
import React from 'react';

import { Renderer, stateRenderer } from 'components/Table';
import { MetricsWorkload, Step2 } from 'types';
import { alphanumericSorter, runStateSorter } from 'utils/data';
import { getWorkload } from 'utils/step';

const batchRender: Renderer<Step2> = (_, record) => {
  return <>{record.batchNum}</>;
};

export const columns: ColumnType<Step2>[] = [
  {
    fixed: 'left',
    key: 'batches',
    render: batchRender,
    sorter: (a: Step2, b: Step2): number => {
      return alphanumericSorter(a.batchNum, b.batchNum);
    },
    title: 'Batches',
    width: 100,
  },
  {
    fixed: 'right',
    key: 'state',
    render: (text, record, index) => stateRenderer(text, record.training, index),
    sorter: (a: Step2, b: Step2): number => {
      // Previously the list of steps would list steps by training steps which
      // all have RunState type as their state.
      return runStateSorter(a.training.state, b.training.state);
    },
    title: 'State',
    width: 120,
  },
  {
    fixed: 'right',
    key: 'checkpoint',
    title: 'Checkpoint',
    width: 100,
  },
];
