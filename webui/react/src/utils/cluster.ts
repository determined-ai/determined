import { Agent, ResourceState, ResourceType } from 'types';

export const getSlotContainerStates = (
  agents: Agent[],
  resourceType: ResourceType,
  resourcePoolName?: string,
): ResourceState[] => {
  const slotContainerStates = agents
    .filter((agent) => {
      return resourcePoolName === undefined || agent.resourcePools.includes(resourcePoolName);
    })
    .flatMap((agent) => {
      const arr: ResourceState[] = Object.entries(agent.slotStats.typeStats ?? {})
        .filter(([type]) => {
          return (
            resourceType === ResourceType.ALL ||
            type === `TYPE_${resourceType}` ||
            resourceType === 'UNSPECIFIED'
          );
        })
        .flatMap(([, val]) => {
          const tempArr = Object.entries(val.states ?? {}).flatMap(([state, count]) => {
            return Array(count).fill(state.replace('STATE_', ''));
          });
          return tempArr;
        });
      return arr;
    });
  return slotContainerStates;
};
