import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { Map, OrderedSet } from 'immutable';

import {
  addResourcePoolBindings,
  deleteResourcePoolBindings,
  getAgents,
  getResourcePoolBindings,
  getResourcePools,
  overwriteResourcePoolBindings,
} from 'services/api';
import { V1ResourcePoolType, V1SchedulerType } from 'services/api-ts-sdk';
import { Agent, ClusterOverview, ClusterOverviewResource, ResourcePool, ResourceType } from 'types';
import handleError from 'utils/error';
import { percent } from 'utils/number';
import { deepObservable, immutableObservable, Observable } from 'utils/observable';

import 'core-js/actual/structured-clone'; // TODO: investigate why structuredClone is breaking if we remove this import.
import PollingStore from './polling';

const initResourceTally: ClusterOverviewResource = { allocation: 0, available: 0, total: 0 };
const initClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: structuredClone(initResourceTally),
  [ResourceType.CUDA]: structuredClone(initResourceTally),
  [ResourceType.ROCM]: structuredClone(initResourceTally),
  [ResourceType.ALL]: structuredClone(initResourceTally),
  [ResourceType.UNSPECIFIED]: structuredClone(initResourceTally),
};

const flexSchedulers: Set<V1SchedulerType> = new Set([V1SchedulerType.PBS, V1SchedulerType.SLURM]);

/**
 * maximum theoretical capacity of the resource pool in terms of the advertised
 * compute slot type.
 * @param pool resource pool
 */
export const maxPoolSlotCapacity = (pool: ResourcePool): number => {
  if (flexSchedulers.has(pool.schedulerType) && pool.slotsAvailable > 0) {
    return pool.slotsAvailable; // The case for HPC Slurm & PBS clusters
  }
  if (pool.maxAgents > 0 && pool.slotsPerAgent && pool.slotsPerAgent > 0) {
    return pool.maxAgents * pool.slotsPerAgent;
  }
  /**
   * On-premise deployments don't have dynamic agents and we don't know how many
   * agents might connect.
   *
   * This is a work around for dynamic agents such as Kubernetes where `slotsAvailable`,
   * `slotsPerAgents` and `maxAgents` are all zero. This value is used for form
   * validation and it is too strict to allow anything to run experiments. Intentially
   * generalized and not matching against Kubernetes, in case other schedulers return
   * zeroes, and this would at least unblock experiments, and the backend would be able
   * to return capacity issues.
   */
  return pool.slotsAvailable || Infinity;
};

/**
 * maximum theoretical capacity of the cluster, by advertised compute slot type. if all pools are
 * static pools, we just tally the agent slots. this method returns a correct cluster-wide total for
 * slurm where pools can have overlapping sets of agents.
 */
export const maxClusterSlotCapacity = (
  pools: ResourcePool[],
  agents: Agent[],
): { [key in ResourceType]: number } => {
  const allPoolsStatic = pools.every(
    ({ type }) => type === V1ResourcePoolType.STATIC || type === V1ResourcePoolType.K8S,
  );

  if (allPoolsStatic) {
    return agents.reduce(
      (acc, agent) => {
        agent.resources.forEach((resource) => {
          if (!(resource.type in acc)) acc[resource.type] = 0;
          acc[resource.type] += 1;
          acc[ResourceType.ALL] += 1;
        });
        return acc;
      },
      { ALL: 0 } as { [key in ResourceType]: number },
    );
  } else {
    return pools.reduce(
      (acc, pool) => {
        if (!(pool.slotType in acc)) acc[pool.slotType] = 0;
        const maxPoolSlots = maxPoolSlotCapacity(pool);
        acc[pool.slotType] += maxPoolSlots;
        acc[ResourceType.ALL] += maxPoolSlots;
        return acc;
      },
      { ALL: 0 } as { [key in ResourceType]: number },
    );
  }
};

const clusterStatusText = (
  overview: ClusterOverview,
  pools: ResourcePool[],
  agents: Agent[],
): string | undefined => {
  if (overview[ResourceType.ALL].allocation === 0) return undefined;
  const totalSlots = maxClusterSlotCapacity(pools, agents)[ResourceType.ALL];
  if (totalSlots === 0) return `${overview[ResourceType.ALL].allocation}%`;
  return `${percent(
    (overview[ResourceType.ALL].total - overview[ResourceType.ALL].available) / totalSlots,
  )}%`;
};

class ClusterStore extends PollingStore {
  #agents = deepObservable<Loadable<Agent[]>>(NotLoaded);
  #resourcePools = deepObservable<Loadable<ResourcePool[]>>(NotLoaded);
  #unboundResourcePools = deepObservable<Loadable<ResourcePool[]>>(NotLoaded);
  #resourcePoolBindings = immutableObservable<Map<string, OrderedSet<number>>>(Map());

  public readonly agents = this.#agents.readOnly();
  public readonly resourcePoolBindings = this.#resourcePoolBindings.select((bindings) =>
    bindings.map((workspaceIds) => workspaceIds.toJS()),
  );

  public readonly resourcePools = this.#resourcePools.select((loadable) => {
    return Loadable.map(loadable, (pools) => {
      return pools.sort((a, b) => a.name.localeCompare(b.name));
    });
  });

  public readonly unboundResourcePools = this.#unboundResourcePools.readOnly();

  public readonly clusterOverview = this.#agents.select((agents) =>
    Loadable.map(agents, (agents) => {
      const overview: ClusterOverview = structuredClone(initClusterOverview);

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

  public readonly clusterStatus = Observable.select(
    [this.clusterOverview, this.#resourcePools, this.#agents],
    (overview, pools, agents) => {
      return Loadable.match(Loadable.all([overview, pools, agents]), {
        _: () => undefined,
        Loaded: ([overview, pools, agents]) => clusterStatusText(overview, pools, agents) ?? '',
      });
    },
  );

  public fetchAgents(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getAgents({}, { signal: signal ?? canceler.signal })
      .then((response) => {
        this.#agents.set(Loaded(response));
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public fetchResourcePools(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getResourcePools({}, { signal: signal ?? canceler.signal })
      .then((response) => {
        this.#resourcePools.set(Loaded(response));
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public fetchUnboundResourcePools(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getResourcePools({ unbound: true }, { signal: signal ?? canceler.signal })
      .then((response) => {
        this.#unboundResourcePools.set(Loaded(response));
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public async poll() {
    const agentRequest = getAgents({}, { signal: this.canceler?.signal });
    const poolsRequest = getResourcePools({}, { signal: this.canceler?.signal });
    const [agents, resourcePools] = await Promise.all([agentRequest, poolsRequest]);
    this.#resourcePools.set(Loaded(resourcePools));
    this.#agents.set(Loaded(agents));
  }

  public readonly boundWorkspaces = (resourcePool: string) =>
    this.#resourcePoolBindings.select((map) => map.get(resourcePool));

  public fetchResourcePoolBindings(resourcePoolName: string, signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getResourcePoolBindings({ resourcePoolName }, { signal: signal ?? canceler.signal })
      .then((response) => {
        this.#resourcePoolBindings.update((map) =>
          map.set(resourcePoolName, OrderedSet(response.workspaceIds ?? [])),
        );
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public addResourcePoolBindings(
    resourcePool: string,
    workspaceIds: number[],
    signal?: AbortSignal,
  ): () => void {
    const canceler = new AbortController();

    addResourcePoolBindings(
      { resourcePoolName: resourcePool, workspaceIds },
      { signal: signal ?? canceler.signal },
    )
      .then(() => {
        this.#resourcePoolBindings.update((map) =>
          map.update(resourcePool, (prevWorkspaceIds) =>
            (prevWorkspaceIds ?? OrderedSet()).union(workspaceIds),
          ),
        );
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public deleteResourcePoolBindings(
    resourcePool: string,
    workspaceIds: number[],
    signal?: AbortSignal,
  ): () => void {
    const canceler = new AbortController();

    deleteResourcePoolBindings(
      { resourcePoolName: resourcePool, workspaceIds },
      { signal: signal ?? canceler.signal },
    )
      .then(() => {
        this.#resourcePoolBindings.update((map) =>
          map.update(
            resourcePool,
            (oldWorkspaceIds) => oldWorkspaceIds?.filter((id) => !workspaceIds.includes(id)),
          ),
        );
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public overwriteResourcePoolBindings(
    resourcePool: string,
    workspaceIds: number[],
    signal?: AbortSignal,
  ): () => void {
    const canceler = new AbortController();

    overwriteResourcePoolBindings(
      { resourcePoolName: resourcePool, workspaceIds },
      { signal: signal ?? canceler.signal },
    )
      .then(() => {
        this.#resourcePoolBindings.update((map) => map.set(resourcePool, OrderedSet(workspaceIds)));
      })
      .catch(handleError);

    return () => canceler.abort();
  }
}

export default new ClusterStore();
