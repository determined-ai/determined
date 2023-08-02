import { useObservable } from 'micro-observables';
import React, { useCallback } from 'react';

import Card from 'components/kit/Card';
import Icon from 'components/kit/Icon';
import ResourcePoolCard from 'components/ResourcePoolCard';
import Section from 'components/Section';
import usePermissions from 'hooks/usePermissions';
import clusterStore from 'stores/cluster';
import { ResourcePool } from 'types';
import { Loadable } from 'utils/loadable';

const ResourcePoolsBound: React.FC = () => {
  const resourcePools = useObservable(clusterStore.resourcePools);
  const { canManageResourcePoolBindings } = usePermissions();

  const actionMenu = useCallback(
    (pool: ResourcePool) =>
      canManageResourcePoolBindings
        ? [
            {
              disabled: pool.defaultAuxPool || pool.defaultComputePool,
              icon: <Icon name="four-squares" title="set-default" />,
              key: 'set-default',
              label: 'Set as Default Resource Pool',
            },
          ]
        : [],
    [canManageResourcePoolBindings],
  );

  return (
    <>
      <Section title="Bound Resource Pools">
        <Card.Group size="medium">
          {Loadable.isLoaded(resourcePools) &&
            resourcePools.data.map((rp, idx) => (
              <ResourcePoolCard actionMenu={actionMenu(rp)} key={idx} resourcePool={rp} />
            ))}
        </Card.Group>
      </Section>
      <Section title="Shared Resource Pools">
        <Card.Group size="medium">
          {Loadable.isLoaded(resourcePools) &&
            resourcePools.data.map((rp, idx) => (
              <ResourcePoolCard actionMenu={actionMenu(rp)} key={idx} resourcePool={rp} />
            ))}
        </Card.Group>
      </Section>
    </>
  );
};

export default ResourcePoolsBound;
