import * as Type from 'types';
import * as utils from 'utils/cluster';

const AGENTS = [
  {
    id: 'Calebs-MacBook-Pro.local',
    registeredTime: 1637797899,
    resourcePools: ['aux-pool'],
    resources: [
      {
        container: {
          id: '',
          state: Type.ResourceState.Assigned,
        },
        enabled: true,
        id: '0',
        name: 'Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz x 8 physical cores',
        type: Type.ResourceType.CPU,
        uuid: 'GenuineIntel',
      },
    ],
  },
  {
    id: 'i-05caeddde60b7bb2a',
    registeredTime: 1638250166,
    resourcePools: ['compute-pool'],
    resources: [
      {
        container: {
          id: '9d6eaaa0-5ffb-491f-8fc8-3970a2a2b2b8',
          state: Type.ResourceState.Running,
        },
        enabled: true,
        id: '0',
        name: 'Tesla K80',
        type: Type.ResourceType.CUDA,
        uuid: 'GPU-d3a502f5-2637-3e09-6a6c-b56efa07288e',
      },
    ],
  },
  {
    id: 'i-08c04ab8ca93366c4',
    registeredTime: 1638250636,
    resourcePools: ['compute-pool'],
    resources: [
      {
        container: {
          id: '0192e222-51a0-11ec-bf63-0242ac130002',
          state: Type.ResourceState.Terminated,
        },
        enabled: false,
        id: '1',
        name: 'Tesla K80',
        type: Type.ResourceType.CUDA,
        uuid: 'CPU-0b4ced76-51a0-11ec-bf63-0242ac130002',
      },
    ],
  },
];

describe('Cluster Utilities', () => {
  describe('getSlotContainerStates', () => {
    it('should convert all agents into slot container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.ALL);
      const expected = [Type.ResourceState.Assigned, Type.ResourceState.Running];
      expect(result).toStrictEqual(expected);
    });

    it('should convert enabled CPU agents into slot container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.CPU);
      const expected = [Type.ResourceState.Assigned];
      expect(result).toStrictEqual(expected);
    });

    it('should convert enabled GPU agents into slot container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.CUDA);
      const expected = [Type.ResourceState.Running];
      expect(result).toStrictEqual(expected);
    });

    it('should convert specified resource pool agents into container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.CUDA, 'compute-pool');
      const expected = [Type.ResourceState.Running];
      expect(result).toStrictEqual(expected);
    });
  });
});
