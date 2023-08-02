import React, { useCallback, useState } from 'react';

import Card from 'components/kit/Card';
import Icon from 'components/kit/Icon';
import ResourcePoolCard from 'components/ResourcePoolCard';
import ResourcePoolDetails from 'components/ResourcePoolDetails';
import Section from 'components/Section';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import clusterStore from 'stores/cluster';
import { ResourcePool } from 'types';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import { ClusterOverallBar } from '../Cluster/ClusterOverallBar';
import { ClusterOverallStats } from '../Cluster/ClusterOverallStats';

const ClusterOverview: React.FC = () => {
  const resourcePools = useObservable(clusterStore.resourcePools);
  const rpBindingFlagOn = useFeature().isOn('rp_binding');
  const { canManageResourcePoolBindings } = usePermissions();

  const [rpDetail, setRpDetail] = useState<ResourcePool>();

  const hideModal = useCallback(() => setRpDetail(undefined), []);

  const actionMenu = useCallback(
    (pool: ResourcePool) =>
      rpBindingFlagOn && canManageResourcePoolBindings
        ? [
            {
              disabled: pool.defaultAuxPool || pool.defaultComputePool,
              icon: <Icon name="four-squares" title="manage-bindings" />,
              key: 'bindings',
              label: 'Manage bindings',
            },
          ]
        : [],
    [canManageResourcePoolBindings, rpBindingFlagOn],
  );

  return (
    <>
      <ClusterOverallStats />
      <ClusterOverallBar />
      <Section title="Resource Pools">
        <Card.Group size="medium">
          {Loadable.isLoaded(resourcePools) &&
            resourcePools.data.map((rp, idx) => (
              <ResourcePoolCard actionMenu={actionMenu(rp)} key={idx} resourcePool={rp} />
            ))}
        </Card.Group>
      </Section>
      {!!rpDetail && (
        <ResourcePoolDetails finally={hideModal} resourcePool={rpDetail} visible={!!rpDetail} />
      )}
    </>
  );
};

export default ClusterOverview;
