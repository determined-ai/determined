import { Observable, observable, useObservable, WritableObservable } from 'micro-observables';
import React, {
  createContext,
  ReactElement,
  ReactNode,
  useContext,
  useEffect,
  useState,
} from 'react';

import { clusterStatusText } from 'pages/Clusters/ClustersOverview';
import { getAgents, getResourcePools } from 'services/api';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { clone } from 'shared/utils/data';
import { percent } from 'shared/utils/number';
import { Agent, ClusterOverview, ClusterOverviewResource, ResourcePool, ResourceType } from 'types';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';

import { AuthProvider } from './auth';
import { DeterminedInfoProvider } from './determinedInfo';
import { ExperimentsProvider } from './experiments';
import { KnownRolesProvider } from './knowRoles';
import { ProjectsProvider } from './projects';
import { TasksProvider } from './tasks';
import { UserRolesProvider } from './userRoles';
import { UsersProvider } from './users';
import { WorkspacesProvider } from './workspaces';

const initResourceTally: ClusterOverviewResource = { allocation: 0, available: 0, total: 0 };
// TODO: dont export
export const initClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: clone(initResourceTally),
  [ResourceType.CUDA]: clone(initResourceTally),
  [ResourceType.ROCM]: clone(initResourceTally),
  [ResourceType.ALL]: clone(initResourceTally),
  [ResourceType.UNSPECIFIED]: clone(initResourceTally),
};

class StoreService {
  agents: WritableObservable<Loadable<Agent[]>> = observable(NotLoaded);
  resourcePools: WritableObservable<Loadable<ResourcePool[]>> = observable(NotLoaded);
  pollingHandle: NodeJS.Timer;

  canceler: AbortController;

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

  constructor(canceler: AbortController) {
    this.canceler = canceler;
  }

  fetchAgents = async (): Promise<void> => {
    try {
      const response = await getAgents({}, { signal: this.canceler.signal });
      this.agents.set(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  };

  fetchResourcePools = async (): Promise<void> => {
    try {
      const response = await getResourcePools({}, { signal: this.canceler.signal });
      this.resourcePools.set(Loaded(response));
    } catch (e) {
      handleError(e);
    }
  };

  poll = (): void => {
    const pollFn = () => {
      this.fetchResourcePools();
      this.fetchAgents();
    };

    pollFn();

    this.pollingHandle = setInterval(pollFn, 5000);
  };

  cancelPolling = (): void => {
    if (!this.pollingHandle) return;
    clearInterval(this.pollingHandle);
  };
}

const StoreContext = createContext<StoreService | null>(null);

export const useStore = (): StoreService => {
  const store = useContext(StoreContext);
  if (store === null) throw new Error('this is not a store');
  return store;
};

export const StoreProvider = ({ children }: { children: ReactNode }): ReactElement => {
  const [store] = useState(() => new StoreService(new AbortController()));

  store.poll();

  useEffect(() => store.cancelPolling, [store]);

  return (
    <StoreContext.Provider value={store}>
      <UIProvider>
        <UsersProvider>
          <AuthProvider>
            <ExperimentsProvider>
              <TasksProvider>
                <WorkspacesProvider>
                  <DeterminedInfoProvider>
                    <UserRolesProvider>
                      <KnownRolesProvider>
                        <ProjectsProvider>{children}</ProjectsProvider>
                      </KnownRolesProvider>
                    </UserRolesProvider>
                  </DeterminedInfoProvider>
                </WorkspacesProvider>
              </TasksProvider>
            </ExperimentsProvider>
          </AuthProvider>
        </UsersProvider>
      </UIProvider>
    </StoreContext.Provider>
  );
};

export const useClusterStatus = (): string | undefined => {
  const store = useStore();
  if (store === null) throw new Error('no store');
  return useObservable(store.clusterStatus);
};

export const useAgents = ():Loadable<Agent[]> => {
  const store = useStore();
  if (store === null) throw new Error('no store');
  return useObservable(store.agents);
};

export const useResourcePools = (): Loadable<ResourcePool[]> => {
  const store = useStore();
  if (store === null) throw new Error('no store');
  return useObservable(store.resourcePools);
};

export const useClusterOverview = (): Loadable<ClusterOverview> => {
  const store = useStore();
  if (store === null) throw new Error('no store');
  return useObservable(store.clusterOverview);
};
