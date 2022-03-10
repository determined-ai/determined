import React from 'react';

import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { ResourcePool, ResourceType } from 'types';

import ResourcePoolCardLight from './ResourcePoolCardLight';

const rps = resourcePools as unknown as ResourcePool[];

export default {
  component: ResourcePoolCardLight,
  title: 'ResourcePoolCardLight',
};

export const Default = (): React.ReactNode => (
  <ResourcePoolCardLight
    computeContainerStates={[]}
    resourcePool={rps.random()}
    resourceType={ResourceType.CUDA}
    totalComputeSlots={3}
  />
);

export const CPU = (): React.ReactNode => (
  <ResourcePoolCardLight
    computeContainerStates={[]}
    resourcePool={rps.random()}
    resourceType={ResourceType.CPU}
    totalComputeSlots={3}
  />
);

export const Aux = (): React.ReactNode => (
  <ResourcePoolCardLight
    computeContainerStates={[]}
    resourcePool={rps.random()}
    resourceType={ResourceType.UNSPECIFIED}
    totalComputeSlots={0}
  />
);
