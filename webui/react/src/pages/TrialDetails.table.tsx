import { ColumnType } from 'antd/es/table';
import React from 'react';

import { Renderer, stateRenderer } from 'components/Table';
import { MetricsWorkload, WorkloadWrapper } from 'types';
import { alphanumericSorter, runStateSorter } from 'utils/data';
import { getWorkload } from 'utils/step';

const batchRender: Renderer<WorkloadWrapper> = (_, record) => {
  const wl = getWorkload(record);
  return <>{wl.numBatches + wl.priorBatchesProcessed}</>;
};

export const columns: ColumnType<WorkloadWrapper>[] = [
  {
    fixed: 'left',
    key: 'batches',
    render: batchRender,
    sorter: (a: WorkloadWrapper, b: WorkloadWrapper): number => {
      const wlA = getWorkload(a);
      const wlB = getWorkload(b);
      return alphanumericSorter(
        wlA.numBatches + wlA.priorBatchesProcessed,
        wlB.numBatches + wlB.priorBatchesProcessed,
      );
    },
    title: 'Batches',
    width: 100,
  },
  {
    fixed: 'right',
    key: 'state',
    render: (text, record, index) => stateRenderer(text, getWorkload(record), index),
    sorter: (a: WorkloadWrapper, b: WorkloadWrapper): number => {
      // Previously the list of steps would list steps by training steps which
      // all have RunState type as their state.
      const wlA = getWorkload(a) as MetricsWorkload;
      const wlB = getWorkload(b) as MetricsWorkload;
      return runStateSorter(wlA.state, wlB.state);
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
