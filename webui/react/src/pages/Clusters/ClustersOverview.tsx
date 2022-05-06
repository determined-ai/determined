import React, { useCallback, useEffect, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import ResourcePoolCardLight from 'components/ResourcePoolCardLight';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import Section from 'components/Section';
import { useStore } from 'contexts/Store';
import { useFetchAgents, useFetchResourcePools } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import { maxPoolSlotCapacity } from 'pages/Cluster/ClusterOverview';
import { ShirtSize } from 'themes';
import {
  ResourcePool,
} from 'types';
import { getSlotContainerStates } from 'utils/cluster';

import { ClusterOverallBar } from '../Cluster/ClusterOverallBar';

import css from './ClustersOverview.module.scss';

const ClusterOverview: React.FC = () => {

  const { agents, resourcePools } = useStore();
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
      <ClusterOverallBar />
      <Section
        title={'Resource Pools'}>
        <Grid gap={ShirtSize.medium} minItemWidth={300} mode={GridMode.AutoFill}>
          {resourcePools.map((rp, idx) => (
            <ResourcePoolCardLight
              computeContainerStates={
                getSlotContainerStates(agents || [], rp.slotType, rp.name)
              }
              key={idx}
              resourcePool={rp}
              resourceType={rp.slotType}
              totalComputeSlots={maxPoolSlotCapacity(rp)}
            />
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
