import { makeAutoObservable } from 'mobx';
import { observer } from 'mobx-react';
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

const StoreObserver = observer(
  ({ children }: { children: ReactNode }): ReactElement => <>{children}</>,
);

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
  agents: Loadable<Agent[]> = NotLoaded;
  resourcePools: Loadable<ResourcePool[]> = NotLoaded;
  pollingHandle: NodeJS.Timer

  canceler: AbortController;

  constructor(canceler: AbortController) {
    this.canceler = canceler;
    makeAutoObservable(this);
  }

  get clusterOverview(): Loadable<ClusterOverview> {
    return Loadable.map(this.agents, (agents) => {
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
  }

  get clusterStatus(): string | undefined {
    return Loadable.match(Loadable.all([this.clusterOverview, this.resourcePools, this.agents]), {
      Loaded: ([overview, pools, agents]) => clusterStatusText(overview, pools, agents) ?? '',
      NotLoaded: () => undefined,
    });
  }
  fetchAgents = async (): Promise<void> => {
    try {
      const agents = await getAgents({}, { signal: this.canceler.signal });
      this.agents = Loaded(agents);
    } catch (e) {
      handleError(e);
    }
  };

 fetchResourcePools = async (): Promise<void> => {
    try {
      const response = await getResourcePools({}, { signal: this.canceler.signal });
      this.resourcePools = Loaded(response);
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
  }
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
                        <ProjectsProvider>
                          <StoreObserver>{children}</StoreObserver>
                        </ProjectsProvider>
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
  return store.clusterStatus;
};

export const useAgents = ():Loadable<Agent[]> => {
  const store = useStore();
  if (store === null) throw new Error('no store');
  return store.agents;
};

export const useResourcePools = (): Loadable<ResourcePool[]> => {
  const store = useStore();
  if (store === null) throw new Error('no store');
  return store.resourcePools;
};

export const useClusterOverview = (): Loadable<ClusterOverview> => {
  const store = useStore();
  if (store === null) throw new Error('no store');
  return store.clusterOverview;
};
