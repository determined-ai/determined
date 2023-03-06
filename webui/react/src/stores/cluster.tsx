import React, {
  createContext,
  ReactElement,
  ReactNode,
  useContext,
  useEffect,
  useState,
} from 'react';

import { clusterStatusText } from 'pages/Clusters/utils';
import { getAgents, getResourcePools } from 'services/api';
import { clone, isEqual } from 'shared/utils/data';
import { percent } from 'shared/utils/number';
import { selectIsAuthenticated } from 'stores/auth';
import { Agent, ClusterOverview, ClusterOverviewResource, ResourcePool, ResourceType } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { Observable, observable, useObservable, WritableObservable } from 'utils/observable';

const initResourceTally: ClusterOverviewResource = { allocation: 0, available: 0, total: 0 };
// TODO: dont export
export const initClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: clone(initResourceTally),
  [ResourceType.CUDA]: clone(initResourceTally),
  [ResourceType.ROCM]: clone(initResourceTally),
  [ResourceType.ALL]: clone(initResourceTally),
  [ResourceType.UNSPECIFIED]: clone(initResourceTally),
};

class ClusterService {
  agents: WritableObservable<Loadable<Agent[]>> = observable(NotLoaded);
  resourcePools: WritableObservable<Loadable<ResourcePool[]>> = observable(NotLoaded);
  pollingHandle: NodeJS.Timer;

  canceler?: AbortController;

  clusterOverview = this.agents.select((agents) =>
    Loadable.map(agents, (agents) => {
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
    }),
  );

  clusterStatus = Observable.select(
    [this.clusterOverview, this.resourcePools, this.agents],
    (overview, pools, agents) => {
      return Loadable.match(Loadable.all([overview, pools, agents]), {
        Loaded: ([overview, pools, agents]) => clusterStatusText(overview, pools, agents) ?? '',
        NotLoaded: () => undefined,
      });
    },
  );

  fetchAgents = async (signal: AbortSignal): Promise<void> => {
    try {
      const response = await getAgents({}, { signal });
      this.agents.update((prev) => (isEqual(prev, Loaded(response)) ? prev : Loaded(response)));
    } catch (e) {
      handleError(e);
    }
  };

  fetchResourcePools = async (signal: AbortSignal): Promise<void> => {
    try {
      const response = await getResourcePools({}, { signal });
      this.resourcePools.update((prev) =>
        isEqual(prev, Loaded(response)) ? prev : Loaded(response),
      );
    } catch (e) {
      handleError(e);
    }
  };

  startPolling = (): void => {
    this.cancelPolling();
    const canceler = new AbortController();
    const pollFn = () => {
      this.fetchResourcePools(canceler.signal);
      this.fetchAgents(canceler.signal);
    };

    this.canceler = canceler;
    pollFn();

    this.pollingHandle = setInterval(pollFn, 5000);
  };

  cancelPolling = (): void => {
    this.canceler?.abort();
    if (!this.pollingHandle) return;
    clearInterval(this.pollingHandle);
  };
}

const ClusterContext = createContext<ClusterService | null>(null);

export const ClusterProvider = ({ children }: { children: ReactNode }): ReactElement => {
  const [store] = useState(() => new ClusterService());
  const isAuthenticated = useObservable(selectIsAuthenticated);

  useEffect(() => {
    if (isAuthenticated) {
      store.startPolling();
    }
    return () => store.cancelPolling();
  }, [store, isAuthenticated]);

  return <ClusterContext.Provider value={store}>{children}</ClusterContext.Provider>;
};

export const useClusterStore = (): ClusterService => {
  const store = useContext(ClusterContext);
  if (store === null)
    throw new Error('attempted to use cluster store outside of a cluster context');
  return store;
};

export const useRefetchClusterData = (): void => {
  // kick off another round of polling to ensure fresh data
  // only use at top level pages to avoid redundant api calls
  const store = useClusterStore();
  useEffect(() => store.startPolling(), [store]);
};
