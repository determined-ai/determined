import { getAgents, getResourcePools } from 'services/api';
import { V1ResourcePoolType } from 'services/api-ts-sdk';
import { Agent, ClusterOverview, ClusterOverviewResource, ResourcePool, ResourceType } from 'types';
import { clone, isEqual } from 'utils/data';
import handleError from 'utils/error';
import { Loadable, Loaded, NotLoaded } from 'utils/loadable';
import { percent } from 'utils/number';
import { Observable, observable, WritableObservable } from 'utils/observable';

import PollingStore from './polling';

const initResourceTally: ClusterOverviewResource = { allocation: 0, available: 0, total: 0 };
const initClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: clone(initResourceTally),
  [ResourceType.CUDA]: clone(initResourceTally),
  [ResourceType.ROCM]: clone(initResourceTally),
  [ResourceType.ALL]: clone(initResourceTally),
  [ResourceType.UNSPECIFIED]: clone(initResourceTally),
};

/**
 * maximum theoretcial capacity of the resource pool in terms of the advertised
 * compute slot type.
 * @param pool resource pool
 */
export const maxPoolSlotCapacity = (pool: ResourcePool): number => {
  if (pool.maxAgents > 0 && pool.slotsPerAgent && pool.slotsPerAgent > 0)
    return pool.maxAgents * pool.slotsPerAgent;
  // on-premise deployments don't have dynamic agents and we don't know how many
  // agents might connect.
  return pool.slotsAvailable;
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
  const allPoolsStatic = pools.reduce((acc, pool) => {
    return acc && (pool.type === V1ResourcePoolType.STATIC || pool.type === V1ResourcePoolType.K8S);
  }, true);

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

const updateIfChanged = <T, V extends WritableObservable<T>>(o: V, next: T) =>
  o.update((prev) => (isEqual(prev, next) ? prev : next));

class ClusterStore extends PollingStore {
  #agents: WritableObservable<Loadable<Agent[]>> = observable(NotLoaded);
  #resourcePools: WritableObservable<Loadable<ResourcePool[]>> = observable(NotLoaded);

  public readonly agents = this.#agents.readOnly();
  public readonly resourcePools = this.#resourcePools.readOnly();

  public readonly clusterOverview = this.#agents.select((agents) =>
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

  public readonly clusterStatus = Observable.select(
    [this.clusterOverview, this.#resourcePools, this.#agents],
    (overview, pools, agents) => {
      return Loadable.match(Loadable.all([overview, pools, agents]), {
        Loaded: ([overview, pools, agents]) => clusterStatusText(overview, pools, agents) ?? '',
        NotLoaded: () => undefined,
      });
    },
  );

  public fetchAgents(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getAgents({}, { signal: signal ?? canceler.signal })
      .then((response) => {
        updateIfChanged(this.#agents, Loaded(response));
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public fetchResourcePools(signal?: AbortSignal): () => void {
    const canceler = new AbortController();

    getResourcePools({}, { signal: signal ?? canceler.signal })
      .then((response) => {
        updateIfChanged(this.#resourcePools, Loaded(response));
      })
      .catch(handleError);

    return () => canceler.abort();
  }

  public async poll() {
    const agentRequest = getAgents({}, { signal: this.canceler?.signal });
    const poolsRequest = getResourcePools({}, { signal: this.canceler?.signal });
    const [agents, resourcePools] = await Promise.all([agentRequest, poolsRequest]);
    updateIfChanged(this.#resourcePools, Loaded(resourcePools));
    updateIfChanged(this.#agents, Loaded(agents));
  }
}

export default new ClusterStore();
