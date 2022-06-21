import { ResourcePool } from 'types';

import { maxPoolSlotCapacity } from '../Clusters/ClustersOverview';

const pools: Record<string, Partial<ResourcePool>> = {
  dynamic: { maxAgents: 3, slotsAvailable: 0, slotsPerAgent: 2 },
  dynamic2: { maxAgents: 3, slotsAvailable: 2, slotsPerAgent: -1 },
  onPrem: { maxAgents: 0, slotsAvailable: 1 },
};

describe('cluster overview', () => {
  describe('maxPoolSlotCapacity', () => {
    it('should calculate slot capacity for static pools', () => {
      expect(maxPoolSlotCapacity(pools.onPrem as ResourcePool)).toEqual(1);
    });
    it('should calculate slot capacity for dynamic pools', () => {
      expect(maxPoolSlotCapacity(pools.dynamic as ResourcePool)).toEqual(6);
    });
    it('should treat as a static pool with -1 slotsPerAgent', () => {
      expect(maxPoolSlotCapacity(pools.dynamic2 as ResourcePool)).toEqual(2);
    });
  });
});
