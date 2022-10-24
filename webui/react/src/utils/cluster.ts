import { Agent, isDeviceType, ResourceState, ResourceType } from 'types';

export const getSlotContainerStates = (
  agents: Agent[],
  resourceType: ResourceType,
  resourcePoolName?: string,
): ResourceState[] => {
  // agents of k8s clusters do not have resource pool name
  // assume that k8s clusters only have 1 resource pool named 'kubernetes'
  const slotContainerStates = agents
    .filter((agent) =>
      resourcePoolName && resourcePoolName !== 'kubernetes'
        ? agent.resourcePools?.includes(resourcePoolName)
        : true,
    )
    .map((agent) => {
      return isDeviceType(resourceType)
        ? agent.resources.filter((res) => res.type === resourceType)
        : agent.resources;
    })
    .reduce((acc, resource) => [...acc, ...resource], [])
    .filter((resource) => resource.enabled && resource.container)
    .map((resource) => resource.container?.state) as ResourceState[];

  return slotContainerStates;
};
