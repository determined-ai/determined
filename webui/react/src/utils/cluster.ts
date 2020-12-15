import { Agent, ResourceState } from 'types';

export const getSlotContainerStates = (agents: Agent[], resourcePoolName?: string)
: ResourceState[] => {
  const targetAgents = agents;
  if (resourcePoolName) {
    // FIXME we turn this off so that the sample resource pool can match whatever
    // name resource pool agents are attached too.
    // targetAgents = targetAgents.filter(agent => agent.resourcePool);
  }
  const slotContainerStates = targetAgents.map(agent => agent.resources)
    .reduce((acc, cur) => {
      acc.push(...cur);
      return acc;
    }, [])
    .filter(res => res.enabled && res.container)
    .map(res => res.container?.state) as ResourceState[];

  return slotContainerStates;
};
