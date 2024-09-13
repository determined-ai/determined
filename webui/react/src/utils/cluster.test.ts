import * as Type from 'types';

import * as utils from './cluster';

// DISCLAIMER
// Blank unnused fields for this test to reduce complexity,
// so this whole data is not necessarily accurate other than required fields
// Required fields: `resourcePools`, `slotStats.typeStats`
const DEFAULT = 'default';
const COMPUTE_POOL = 'compute-pool';
const NOT_EXIST = 'not-exist';
const AGENTS: Type.Agent[] = [
  {
    enabled: true,
    id: '',
    registeredTime: 0,
    resourcePools: [DEFAULT],
    resources: [],
    slotStats: {
      brandStats: {},
      typeStats: {
        TYPE_CPU: { disabled: 0, draining: 0, states: { STATE_RUNNING: 1 }, total: 2 },
      },
    },
  },
  {
    enabled: true,
    id: '',
    registeredTime: 0,
    resourcePools: [COMPUTE_POOL],
    resources: [],
    slotStats: {
      brandStats: {},
      typeStats: {
        TYPE_CPU: { disabled: 0, draining: 0, states: { STATE_RUNNING: 2 }, total: 2 },
        TYPE_CUDA: {
          disabled: 0,
          draining: 0,
          states: { STATE_PULLING: 3, STATE_RUNNING: 1 },
          total: 5,
        },
      },
    },
  },
];

describe('Cluster Utilities', () => {
  describe('getSlotContainerStates', () => {
    it('should return empty array when agent list is empty', () => {
      const result = utils.getSlotContainerStates([], Type.ResourceType.ALL);
      const expected: Type.ResourceState[] = [];
      expect(result).toStrictEqual(expected);
    });

    it('should convert all agents into slot container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.ALL);
      const expected: Type.ResourceState[] = [
        Type.ResourceState.Running,
        Type.ResourceState.Running,
        Type.ResourceState.Running,
        Type.ResourceState.Pulling,
        Type.ResourceState.Pulling,
        Type.ResourceState.Pulling,
        Type.ResourceState.Running,
      ];
      expect(result).toStrictEqual(expected);
    });

    it('should convert all agents in `compute-pool` into slot container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.ALL, COMPUTE_POOL);
      const expected: Type.ResourceState[] = [
        Type.ResourceState.Running,
        Type.ResourceState.Running,
        Type.ResourceState.Pulling,
        Type.ResourceState.Pulling,
        Type.ResourceState.Pulling,
        Type.ResourceState.Running,
      ];
      expect(result).toStrictEqual(expected);
    });

    it('should convert enabled CPU agents into slot container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.CPU);
      const expected: Type.ResourceState[] = [
        Type.ResourceState.Running,
        Type.ResourceState.Running,
        Type.ResourceState.Running,
      ];
      expect(result).toStrictEqual(expected);
    });

    it('should convert enabled GPU (CUDA) agents into slot container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.CUDA);
      const expected: Type.ResourceState[] = [
        Type.ResourceState.Pulling,
        Type.ResourceState.Pulling,
        Type.ResourceState.Pulling,
        Type.ResourceState.Running,
      ];
      expect(result).toStrictEqual(expected);
    });

    it('should convert enabled UNSPECIFIED agents into slot container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.UNSPECIFIED);
      const expected: Type.ResourceState[] = [
        Type.ResourceState.Running,
        Type.ResourceState.Running,
        Type.ResourceState.Running,
        Type.ResourceState.Pulling,
        Type.ResourceState.Pulling,
        Type.ResourceState.Pulling,
        Type.ResourceState.Running,
      ];
      expect(result).toStrictEqual(expected);
    });

    it('should convert `default` resource pool agents into container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.CPU, DEFAULT);
      const expected: Type.ResourceState[] = [Type.ResourceState.Running];
      expect(result).toStrictEqual(expected);
    });

    it('should convert `compute-pool` resource pool agents into container states', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.CPU, COMPUTE_POOL);
      const expected: Type.ResourceState[] = [
        Type.ResourceState.Running,
        Type.ResourceState.Running,
      ];
      expect(result).toStrictEqual(expected);
    });

    it('should return empty list when resource pool does not exist', () => {
      const result = utils.getSlotContainerStates(AGENTS, Type.ResourceType.CPU, NOT_EXIST);
      const expected: Type.ResourceState[] = [];
      expect(result).toStrictEqual(expected);
    });
  });
});
