import React, { useCallback, useEffect, useState } from 'react';

import Grid, { GridMode } from 'components/Grid';
import Link from 'components/Link';
import ResourcePoolCardLight from 'components/ResourcePoolCardLight';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import Section from 'components/Section';
import { useStore } from 'contexts/Store';
import { useFetchAgents, useFetchResourcePools } from 'hooks/useFetch';
import usePolling from 'hooks/usePolling';
import { paths } from 'routes/utils';
import { ShirtSize } from 'themes';
import {
  ResourcePool,
} from 'types';

import { ClusterOverallBar } from '../Cluster/ClusterOverallBar';
import { ClusterOverallStats } from '../Cluster/ClusterOverallStats';

import css from './ClustersOverview.module.scss';

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
