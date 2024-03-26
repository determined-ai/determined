import { Agent, ResourceState, ResourceType } from 'types';

export const getSlotContainerStates = (
  agents: Agent[],
  resourceType: ResourceType,
  resourcePoolName?: string,
): ResourceState[] => {
  const slotContainerStates = agents
    .filter((agent) => (resourcePoolName ? agent.resourcePools?.includes(resourcePoolName) : true))
    .map((agent) => {
      const ids = Object.keys(agent.slotStats?.deviceTypeCounts ?? {})
        .map((deviceType) => deviceType.replace('TYPE_', ''))
        .filter((deviceType) => deviceType === resourceType);
      const states = ids
        .map((id) => agent.slotStats?.slotStates?.[id]?.replace('STATE_', '') as ResourceState)
        .flatMap((state) => (state === undefined ? [] : [state]));
      return states;
    })
    .flat();

  return slotContainerStates;
};
