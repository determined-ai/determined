import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo } from 'react';

import Card from 'components/kit/Card';
import Icon from 'components/kit/Icon';
import ResourcePoolCard from 'components/ResourcePoolCard';
import Section from 'components/Section';
import usePermissions from 'hooks/usePermissions';
import clusterStore from 'stores/cluster';
import workspaceStore from 'stores/workspaces';
import { ResourcePool } from 'types';
import { Loadable } from 'utils/loadable';

interface Props {
  workspaceId: number;
}

const ResourcePoolsBound: React.FC<Props> = ({ workspaceId }) => {
  const resourcePools = useObservable(clusterStore.resourcePools);
  const boundResourcePoolIds = useObservable(workspaceStore.boundResourcePools(workspaceId));
  const { canManageResourcePoolBindings } = usePermissions();

  useEffect(() => {
    workspaceStore.fetchAvailableResourcePools(workspaceId);
  }, [workspaceId]);

  const boundResourcePools = useMemo(() => {
    if (!Loadable.isLoaded(resourcePools) || !boundResourcePoolIds) return [];
    return resourcePools.data.filter((rp) => boundResourcePoolIds.includes(rp.name));
  }, [resourcePools, boundResourcePoolIds]);

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
          {boundResourcePools.map((rp, idx) => (
            <ResourcePoolCard actionMenu={actionMenu(rp)} key={idx} resourcePool={rp} />
          ))}
        </Card.Group>
      </Section>
      <Section title="Shared Resource Pools">
        <Card.Group size="medium">
          {boundResourcePools.map((rp, idx) => (
            <ResourcePoolCard actionMenu={actionMenu(rp)} key={idx} resourcePool={rp} />
          ))}
        </Card.Group>
      </Section>
    </>
  );
};

export default ResourcePoolsBound;
