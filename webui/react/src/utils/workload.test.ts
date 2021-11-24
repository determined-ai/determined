import { CheckpointState, WorkloadGroup } from 'types';

import * as utils from './workload';

const workloads: WorkloadGroup[] = [
  {
    training: {
      metrics: { accuracy: 0.9, loss: 0.1 },
      totalBatches: 100,
    },
  },
  {
    training: {
      metrics: { accuracy: 0.91, loss: 0.09 },
      totalBatches: 200,
    },
    validation: {
      metrics: { accuracy: 0.81, loss: 0.19 },
      totalBatches: 200,
    },
  },
  {
    checkpoint: {
      state: CheckpointState.Active,
      totalBatches: 300,
    },
    training: {
      metrics: { accuracy: 0.91, loss: 0.09 },
      totalBatches: 300,
    },
    validation: {
      metrics: { accuracy: 0.81, loss: 0.19 },
      totalBatches: 300,
    },
  },
];

describe('workload utilities', () => {
  describe('getWorkload', () => {
    it('should extract one available workload', () => {
      expect(utils.getWorkload(workloads[0])).toStrictEqual(workloads[0].training);
    });

    it('should extract multiple available workloads', () => {
      // Why does getWorkload only return the first available workload?
      expect(utils.getWorkload(workloads[2])).toStrictEqual(workloads[2].checkpoint);
    });
  });
});
