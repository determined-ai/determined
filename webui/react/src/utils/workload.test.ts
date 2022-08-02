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

  describe('hasCheckpointStep', () => {
    it('should detect checkpoint from step', () => {
      const step = {
        batchNum: 100,
        checkpoint: { state: CheckpointState.Active, totalBatches: 100 },
        startTime: '2021-11-29T00:00:00Z',
        training: { totalBatches: 100 },
      };
      expect(utils.hasCheckpointStep(step)).toBe(true);
    });

    it('should reject step with deleted checkpoint', () => {
      const step = {
        batchNum: 100,
        checkpoint: { state: CheckpointState.Deleted, totalBatches: 100 },
        startTime: '2021-11-29T00:00:00Z',
        training: { totalBatches: 100 },
      };
      expect(utils.hasCheckpointStep(step)).toBe(false);
    });
  });

  describe('isMetricsWorkload', () => {
    it('should validate metric workload', () => {
      const metricWorkload = { metrics: {}, totalBatches: 100 };
      expect(utils.isMetricsWorkload(metricWorkload)).toBe(true);
    });

    it('should invalidate checkpoint workload', () => {
      const checkpointWorkload = { state: CheckpointState.Active, totalBatches: 100, uuid: 'abc' };
      expect(utils.isMetricsWorkload(checkpointWorkload)).toBe(false);
    });

    it('should invalidate unknown workload', () => {
      const unknownWorkload = { totalBatches: 100 };
      expect(utils.isMetricsWorkload(unknownWorkload)).toBe(false);
    });
  });

  describe('workloadsToStep', () => {
    it('should convert workloads to steps', () => {
      const results = utils.workloadsToSteps(WORKLOADS);
      const expected = [
        {
          batchNum: 100,
          training: {
            metrics: { accuracy: 0.9, loss: 0.1 },
            totalBatches: 100,
          },
        },
        {
          batchNum: 200,
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
          batchNum: 300,
          checkpoint: { state: 'ACTIVE', totalBatches: 300 },
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
      expect(results).toStrictEqual(expected);
    });
  });
});
