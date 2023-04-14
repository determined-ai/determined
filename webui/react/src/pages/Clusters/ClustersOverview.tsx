import React, { useCallback, useState } from 'react';

import Card from 'components/kit/Card';
import ResourcePoolCard from 'components/ResourcePoolCard';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import Section from 'components/Section';
import clusterStore from 'stores/cluster';
import { ResourcePool } from 'types';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import { ClusterOverallBar } from '../Cluster/ClusterOverallBar';
import { ClusterOverallStats } from '../Cluster/ClusterOverallStats';

const ClusterOverview: React.FC = () => {
  const resourcePools = useObservable(clusterStore.resourcePools);

  const [rpDetail, setRpDetail] = useState<ResourcePool>();

  const hideModal = useCallback(() => setRpDetail(undefined), []);

  return (
    <>
      <ClusterOverallStats />
      <ClusterOverallBar />
      <Section title="Resource Pools">
        <Card.Group size="medium">
          {Loadable.isLoaded(resourcePools) &&
            resourcePools.data.map((rp, idx) => <ResourcePoolCard key={idx} resourcePool={rp} />)}
        </Card.Group>
      </Section>
      {!!rpDetail && (
        <ResourcePoolDetails finally={hideModal} resourcePool={rpDetail} visible={!!rpDetail} />
      )}
    </>
  );
};

export default ClusterOverview;
