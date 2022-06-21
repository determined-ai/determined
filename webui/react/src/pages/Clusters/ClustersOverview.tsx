import Grid, { GridMode } from 'components/Grid';
import Link from 'components/Link';
import ResourcePoolCardLight from 'components/ResourcePoolCardLight';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import Section from 'components/Section';
import { useStore } from 'contexts/Store';
import { useFetchAgents, useFetchResourcePools } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import React, { useCallback, useEffect, useState } from 'react';
import { paths } from 'routes/utils';
import { ShirtSize } from 'themes';
import { ClusterOverview as Overview, ResourcePool, ResourceType } from 'types';
import { percent } from '../../shared/utils/number';

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
export const clusterStatusText = (
    overview: Overview,
    pools: ResourcePool[],
): string | undefined => {
    if (overview[ResourceType.ALL].allocation === 0) return undefined;
    const totalSlots = pools.reduce((totalSlots, currentPool) => {
        return totalSlots + maxPoolSlotCapacity(currentPool);
    }, 0);
    if (totalSlots === 0) return `${overview[ResourceType.ALL].allocation}%`;
    return `${percent((overview[ResourceType.ALL].total - overview[ResourceType.ALL].available)
        / totalSlots)}%`;
};

const ClusterOverview: React.FC = () => {

  const { resourcePools } = useStore();
  const [ rpDetail, setRpDetail ] = useState<ResourcePool>();

  const [ canceler ] = useState(new AbortController());

  const fetchAgents = useFetchAgents(canceler);
  const fetchResourcePools = useFetchResourcePools(canceler);

  usePolling(fetchResourcePools, { interval: 10000 });

  const hideModal = useCallback(() => setRpDetail(undefined), []);

  useEffect(() => {
    fetchAgents();

    return () => canceler.abort();
  }, [ canceler, fetchAgents ]);

  return (
    <div className={css.base}>
      <ClusterOverallStats />
      <ClusterOverallBar />
      <Section
        title={'Resource Pools'}>
        <Grid gap={ShirtSize.large} minItemWidth={300} mode={GridMode.AutoFill}>
          {resourcePools.map((rp, idx) => (
            <Link key={idx} path={paths.resourcePool(rp.name)}>
              <ResourcePoolCardLight
                resourcePool={rp}
              />
            </Link>
          ))}
        </Grid>
      </Section>
      {!!rpDetail && (
        <ResourcePoolDetails
          finally={hideModal}
          resourcePool={rpDetail}
          visible={!!rpDetail}
        />
      )}
    </div>
  );
};

export default ClusterOverview;
