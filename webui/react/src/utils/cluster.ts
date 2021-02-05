import { Agent, ResourceState, ResourceType } from 'types';

export const getSlotContainerStates = (
  agents: Agent[],
  resourceType: ResourceType,
  resourcePoolName?: string,
)
: ResourceState[] => {
  const targetAgents = agents.filter(agent =>
    resourcePoolName ? agent.resourcePool === resourcePoolName : true);
  const slotContainerStates = targetAgents
    .map(agent => agent.resources.filter(res => res.type === resourceType))
    .reduce((acc, cur) => {
      acc.push(...cur);
      return acc;
    }, [])
    .filter(res => res.enabled && res.container)
    .map(res => res.container?.state) as ResourceState[];
  return slotContainerStates;
};
