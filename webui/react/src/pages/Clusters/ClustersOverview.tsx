import React, { useCallback, useEffect, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Link from 'components/Link';
import ResourcePoolCard from 'components/ResourcePoolCard';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import Section from 'components/Section';
import { useStore } from 'contexts/Store';
import {
  useFetchActiveExperiments,
  useFetchActiveTasks,
  useFetchResourcePools,
} from 'hooks/useFetch';
import { paths } from 'routes/utils';
import { V1ResourcePoolType } from 'services/api-ts-sdk';
import usePolling from 'shared/hooks/usePolling';
import { percent } from 'shared/utils/number';
import { useFetchAgents } from 'stores/agents';
import { ShirtSize } from 'themes';
import { Agent, ClusterOverview as Overview, ResourcePool, ResourceType } from 'types';

import { ClusterOverallBar } from '../Cluster/ClusterOverallBar';
import { ClusterOverallStats } from '../Cluster/ClusterOverallStats';

import css from './ClustersOverview.module.scss';

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

/** maximum theoretical capacity of the cluster, by advertised compute slot type. if all pools are
 * static pools, we just tally the agent slots. this method returns a correct cluster-wide total for
 * slurm where pools can have overlapping sets of agents.
 */
export const maxClusterSlotCapacity = (
  pools: ResourcePool[],
  agents: Agent[],
): { [key in ResourceType]: number } => {
  const allPoolsStatic = pools.reduce((acc, pool) => {
    return acc && pool.type === V1ResourcePoolType.STATIC;
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

export const clusterStatusText = (
  overview: Overview,
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

const ClusterOverview: React.FC = () => {
  const { resourcePools } = useStore();

  const [rpDetail, setRpDetail] = useState<ResourcePool>();

  const [canceler] = useState(new AbortController());

  const fetchActiveExperiments = useFetchActiveExperiments(canceler);
  const fetchActiveTasks = useFetchActiveTasks(canceler);
  const fetchAgents = useFetchAgents(canceler);
  const fetchResourcePools = useFetchResourcePools(canceler);

  const fetchActiveRunning = useCallback(async () => {
    await fetchActiveExperiments();
    await fetchActiveTasks();
  }, [fetchActiveExperiments, fetchActiveTasks]);

  usePolling(fetchActiveRunning);
  usePolling(fetchResourcePools, { interval: 10000 });

  const hideModal = useCallback(() => setRpDetail(undefined), []);

  useEffect(() => {
    fetchAgents();

    return () => canceler.abort();
  }, [canceler, fetchAgents]);

  return (
    <div className={css.base}>
      <ClusterOverallStats />
      <ClusterOverallBar />
      <Section title="Resource Pools">
        <Grid gap={ShirtSize.Large} minItemWidth={300} mode={GridMode.AutoFill}>
          {resourcePools.map((rp, idx) => (
            <Link key={idx} path={paths.resourcePool(rp.name)}>
              <ResourcePoolCard resourcePool={rp} />
            </Link>
          ))}
        </Grid>
      </Section>
      {!!rpDetail && (
        <ResourcePoolDetails finally={hideModal} resourcePool={rpDetail} visible={!!rpDetail} />
      )}
    </div>
  );
};

export default ClusterOverview;
