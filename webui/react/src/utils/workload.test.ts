import { CheckpointState, WorkloadGroup } from 'types';

import * as utils from './workload';

const WORKLOADS: WorkloadGroup[] = [
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
  },
  {
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
  },
  {
    training: {
      metrics: { accuracy: 0.91, loss: 0.09 },
      totalBatches: 300,
    },
  },
  {
    validation: {
      metrics: { accuracy: 0.81, loss: 0.19 },
      totalBatches: 300,
    },
  },
];

describe('Workload Utilities', () => {
  describe('checkpointSize', () => {
    it('should return checkpoint sizes from resources', () => {
      const resources = {
        abc: 1024,
        def: 2048,
        ghi: 4096,
      };
      expect(utils.checkpointSize({ resources })).toBe(7168);
    });

    it('should return 0 for invalid checkpoints or checkpoint resources', () => {
      expect(utils.checkpointSize()).toBe(0);
      expect(utils.checkpointSize({ resources: undefined })).toBe(0);
      expect(utils.checkpointSize({ resources: {} })).toBe(0);
    });
  });

  describe('getWorkload', () => {
    it('should extract first available training workload', () => {
      expect(utils.getWorkload(WORKLOADS[0])).toStrictEqual(WORKLOADS[0].training);
    });

    it('should extract first available validation workload', () => {
      expect(utils.getWorkload(WORKLOADS[2])).toStrictEqual(WORKLOADS[2].validation);
    });

    it('should extract first available checkpoint workload', () => {
      expect(utils.getWorkload(WORKLOADS[3])).toStrictEqual(WORKLOADS[3].checkpoint);
    });
  });

  describe('hasCheckpoint', () => {
    it('should detect checkpoint from workload', () => {
      const workload = {
        checkpoint: {
          state: CheckpointState.Active,
          totalBatches: 100,
        },
      };
      expect(utils.hasCheckpoint(workload)).toBe(true);
    });

    it('should reject workload with deleted checkpoint', () => {
      const workload = {
        checkpoint: {
          state: CheckpointState.Deleted,
          totalBatches: 100,
        },
      };
      expect(utils.hasCheckpoint(workload)).toBe(false);
    });
  });
});
