import { Agent, ResourceState, ResourceType } from 'types';

const unspecifiedResourceTypes = new Set([ ResourceType.ALL, ResourceType.UNSPECIFIED ]);

export const getSlotContainerStates = (
  agents: Agent[],
  resourceType: ResourceType,
  resourcePoolName?: string,
)
: ResourceState[] => {
  const targetAgents = agents.filter(agent =>
    resourcePoolName ? agent.resourcePool === resourcePoolName : true);
  const slotContainerStates = targetAgents
    .map(agent => unspecifiedResourceTypes.has(resourceType) ?
      agent.resources.filter(res => res.type === resourceType) : agent.resources)
    .reduce((acc, cur) => {
      acc.push(...cur);
      return acc;
    }, [])
    .filter(res => res.enabled && res.container)
    .map(res => res.container?.state) as ResourceState[];
  return slotContainerStates;
};
