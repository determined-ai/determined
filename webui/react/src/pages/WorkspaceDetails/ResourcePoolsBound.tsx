import Card from 'hew/Card';
import Icon from 'hew/Icon';
import { Loadable } from 'hew/utils/loadable';
import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo } from 'react';

import ResourcePoolCard from 'components/ResourcePoolCard';
import Section from 'components/Section';
import usePermissions from 'hooks/usePermissions';
import { patchWorkspace } from 'services/api';
import clusterStore from 'stores/cluster';
import workspaceStore from 'stores/workspaces';
import { ResourcePool, Workspace } from 'types';

interface Props {
  workspace: Workspace;
}

const ResourcePoolsBound: React.FC<Props> = ({ workspace }) => {
  const resourcePools = useObservable(clusterStore.resourcePools);
  const unboundResourcePools = useObservable(clusterStore.unboundResourcePools);
  const boundResourcePoolNames = useObservable(workspaceStore.boundResourcePools(workspace.id));
  const { canManageResourcePoolBindings } = usePermissions();

  useEffect(() => {
    workspaceStore.fetchAvailableResourcePools(workspace.id);
    clusterStore.fetchUnboundResourcePools();
  }, [workspace.id]);

  const boundResourcePools: ResourcePool[] = useMemo(() => {
    if (!Loadable.isLoaded(resourcePools) || !boundResourcePoolNames) return [];
    const unboundResourcePoolNames = Loadable.getOrElse([], unboundResourcePools).map(
      (p) => p.name,
    );
    return resourcePools.data.filter(
      (rp) =>
        boundResourcePoolNames.includes(rp.name) && !unboundResourcePoolNames.includes(rp.name),
    );
  }, [resourcePools, boundResourcePoolNames, unboundResourcePools]);

  const actionMenu = useCallback(
    (pool: ResourcePool) =>
      canManageResourcePoolBindings
        ? [
            {
              icon: <Icon decorative name="four-squares" />,
              key: 'set-default-aux',
              label: `${
                workspace.defaultAuxPool === pool.name ? 'Unset' : 'Set'
              } as Default Aux Resource Pool`,
              onClick: async () => {
                await patchWorkspace({
                  defaultAuxPool: workspace.defaultAuxPool === pool.name ? '' : pool.name,
                  id: workspace.id,
                });
                workspaceStore.fetch(undefined, true);
              },
            },
            {
              icon: <Icon decorative name="four-squares" />,
              key: 'set-default-compute',
              label: `${
                workspace.defaultComputePool === pool.name ? 'Unset' : 'Set'
              } as Default Compute Resource Pool`,
              onClick: async () => {
                await patchWorkspace({
                  defaultComputePool: workspace.defaultComputePool === pool.name ? '' : pool.name,
                  id: workspace.id,
                });
                workspaceStore.fetch(undefined, true);
              },
            },
          ]
        : [],
    [
      canManageResourcePoolBindings,
      workspace.id,
      workspace.defaultComputePool,
      workspace.defaultAuxPool,
    ],
  );

  return (
    <>
      {boundResourcePools.length > 0 && (
        <Section title="Bound Resource Pools">
          <Card.Group size="medium">
            {boundResourcePools.map((rp) => (
              <ResourcePoolCard
                actionMenu={actionMenu(rp)}
                defaultAux={rp.name === workspace.defaultAuxPool}
                defaultCompute={rp.name === workspace.defaultComputePool}
                key={rp.name}
                resourcePool={rp}
              />
            ))}
          </Card.Group>
        </Section>
      )}
      {Loadable.getOrElse([], unboundResourcePools).length > 0 && (
        <Section title="Shared Resource Pools">
          <Card.Group size="medium">
            {Loadable.getOrElse([], unboundResourcePools).map((rp: ResourcePool) => (
              <ResourcePoolCard key={rp.name} resourcePool={rp} />
            ))}
          </Card.Group>
        </Section>
      )}
    </>
  );
};

export default ResourcePoolsBound;
