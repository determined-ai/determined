import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo } from 'react';

import Card from 'components/kit/Card';
import Icon from 'components/kit/Icon';
import ResourcePoolCard from 'components/ResourcePoolCard';
import Section from 'components/Section';
import usePermissions from 'hooks/usePermissions';
import { patchWorkspace } from 'services/api';
import clusterStore from 'stores/cluster';
import workspaceStore from 'stores/workspaces';
import { ResourcePool, Workspace } from 'types';
import { Loadable } from 'utils/loadable';

interface Props {
  workspace: Workspace;
}

const ResourcePoolsBound: React.FC<Props> = ({ workspace }) => {
  const resourcePools = useObservable(clusterStore.resourcePools);
  const unBoundResourcePools = useObservable(clusterStore.unBoundResourcePools);
  const boundResourcePoolIds = useObservable(workspaceStore.boundResourcePools(workspace.id));
  const { canManageResourcePoolBindings } = usePermissions();

  useEffect(() => {
    workspaceStore.fetchAvailableResourcePools(workspace.id);
    clusterStore.fetchUnboundResourcePools();
  }, [workspace.id]);

  const boundResourcePools: ResourcePool[] = useMemo(() => {
    if (!Loadable.isLoaded(resourcePools) || !boundResourcePoolIds) return [];
    const unBoundResourcePoolIds = Loadable.isLoaded(unBoundResourcePools)
      ? unBoundResourcePools.data.map((p) => p.name)
      : [];
    return resourcePools.data.filter(
      (rp) => boundResourcePoolIds.includes(rp.name) && !unBoundResourcePoolIds.includes(rp.name),
    );
  }, [resourcePools, boundResourcePoolIds, unBoundResourcePools]);

  const renderDefaultLabel = useCallback(
    (pool: ResourcePool) => {
      if (pool.name === workspace.defaultAuxPool && pool.name === workspace.defaultComputePool)
        return 'Default';
      if (pool.name === workspace.defaultAuxPool) return 'Default Aux';
      if (pool.name === workspace.defaultComputePool) return 'Default Compute';
      return ' ';
    },
    [workspace.defaultComputePool, workspace.defaultAuxPool],
  );

  const actionMenu = useCallback(
    (pool: ResourcePool) =>
      canManageResourcePoolBindings
        ? [
            {
              disabled: workspace.defaultAuxPool === pool.name,
              icon: <Icon name="four-squares" title="set-default" />,
              key: 'set-default-aux',
              label: 'Set as Default Aux Resource Pool',
              onClick: async () => {
                await patchWorkspace({
                  defaultAuxPool: pool.name,
                  id: workspace.id,
                });
                workspaceStore.fetch(undefined, true);
              },
            },
            {
              disabled: workspace.defaultComputePool === pool.name,
              icon: <Icon name="four-squares" title="set-default" />,
              key: 'set-default-compute',
              label: 'Set as Default Compute Resource Pool',
              onClick: async () => {
                await patchWorkspace({
                  defaultComputePool: pool.name,
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
            {boundResourcePools.map((rp, idx) => (
              <ResourcePoolCard
                actionMenu={actionMenu(rp)}
                defaultLabel={renderDefaultLabel(rp)}
                key={idx}
                resourcePool={rp}
              />
            ))}
          </Card.Group>
        </Section>
      )}
      {Loadable.isLoaded(unBoundResourcePools) && (
        <Section title="Shared Resource Pools">
          <Card.Group size="medium">
            {unBoundResourcePools.data.map((rp: ResourcePool, idx: number) => (
              <ResourcePoolCard defaultLabel={' '} key={idx} resourcePool={rp} />
            ))}
          </Card.Group>
        </Section>
      )}
    </>
  );
};

export default ResourcePoolsBound;
