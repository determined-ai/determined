import { Agent, deviceTypes, ResourceState, ResourceType } from 'types';

export const getSlotContainerStates = (
  agents: Agent[],
  resourceType: ResourceType,
  resourcePoolName?: string,
): ResourceState[] => {
  const slotContainerStates = agents
    .filter(agent => resourcePoolName ? agent.resourcePool === resourcePoolName : true)
    .map(agent => {
      return deviceTypes.has(resourceType)
        ? agent.resources.filter(res => res.type === resourceType)
        : agent.resources;
    })
    .reduce((acc, resource) => ([ ...acc, ...resource ]), [])
    .filter(resource => resource.enabled && resource.container)
    .map(resource => resource.container?.state) as ResourceState[];

  return slotContainerStates;
};
