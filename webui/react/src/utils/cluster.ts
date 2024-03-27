import { Agent, ResourceState, ResourceType } from 'types';

export const getSlotContainerStates = (
  agents: Agent[],
  resourceType: ResourceType,
  resourcePoolName?: string,
): ResourceState[] => {
  const slotContainerStates = agents
    .filter((agent) => {
      if (resourcePoolName === undefined) {
        return true;
      }
      for (const agentResourcePool of agent.resourcePools) {
        if (!resourcePoolName.includes(agentResourcePool)) {
          return false;
        }
      }
      return true;
    })
    .flatMap((agent) => {
      const arr: ResourceState[] = Object.entries(agent.slotStats.typeStats ?? {})
        .filter(([type]) => {
          return resourceType === ResourceType.ALL ? true : type === `TYPE_${resourceType}`;
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
