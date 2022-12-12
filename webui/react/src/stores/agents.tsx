import React, { createContext, PropsWithChildren, useCallback, useContext, useState } from 'react';

import { getAgents } from 'services/api';
import { clone } from 'shared/utils/data';
import { percent } from 'shared/utils/number';
import { noOp } from 'shared/utils/service';
import { Agent, ClusterOverview, ClusterOverviewResource, ResourceType } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

type AgentContext = {
  agents: Loadable<Agent[]>;
  updateAgents: (a: Loadable<Agent[]>) => void;
};

const initResourceTally: ClusterOverviewResource = { allocation: 0, available: 0, total: 0 };
// TODO: dont export
export const initClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: clone(initResourceTally),
  [ResourceType.CUDA]: clone(initResourceTally),
  [ResourceType.ROCM]: clone(initResourceTally),
  [ResourceType.ALL]: clone(initResourceTally),
  [ResourceType.UNSPECIFIED]: clone(initResourceTally),
};

const AgentsContext = createContext<AgentContext>({ agents: NotLoaded, updateAgents: noOp });

export const AgentsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [state, setState] = useState<Loadable<Agent[]>>(NotLoaded);
  return (
    <AgentsContext.Provider value={{ agents: state, updateAgents: setState }}>
      {children}
    </AgentsContext.Provider>
  );
};

export const useFetchAgents = (canceler: AbortController): (() => Promise<void>) => {
  const { updateAgents } = useContext(AgentsContext);

  return useCallback(async (): Promise<void> => {
    try {
      const agents = await getAgents({}, { signal: canceler.signal });
      updateAgents(Loaded(agents));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateAgents]);
};

export const useEnsureAgentsFetched = (canceler: AbortController): (() => Promise<void>) => {
  const { agents, updateAgents } = useContext(AgentsContext);

  return useCallback(async (): Promise<void> => {
    if (agents !== NotLoaded) return;
    try {
      const agents = await getAgents({ signal: canceler.signal });
      updateAgents(Loaded(agents));
    } catch (e) {
      handleError(e);
    }
  }, [canceler, updateAgents, agents]);
};
export const useAgents = (): Loadable<Agent[]> => {
  const context = useContext(AgentsContext);
  if (context === null) {
    throw new Error('Attempted to use useAgents outside of Agent Context');
  }
  return context.agents;
};

export const useClusterOverview = (): Loadable<ClusterOverview> => {
  // Deep clone for render detection.
  const agents = useAgents();
  // TODO: memoize
  return Loadable.map(agents, (agents) => {
    const overview: ClusterOverview = clone(initClusterOverview);

    agents.forEach((agent) => {
      agent.resources
        .filter((resource) => resource.enabled)
        .forEach((resource) => {
          const isResourceFree = resource.container == null;
          const availableResource = isResourceFree ? 1 : 0;
          overview[resource.type].available += availableResource;
          overview[resource.type].total++;
          overview[ResourceType.ALL].available += availableResource;
          overview[ResourceType.ALL].total++;
        });
    });

    for (const key in overview) {
      const rt = key as ResourceType;
      overview[rt].allocation =
        overview[rt].total !== 0
          ? percent((overview[rt].total - overview[rt].available) / overview[rt].total)
          : 0;
    }

    return overview;
  });
};
