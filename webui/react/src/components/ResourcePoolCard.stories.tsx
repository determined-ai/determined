import React from 'react';

import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { ResourcePool } from 'types';

import ResourcePoolCard from './ResourcePoolCard';

const rps = resourcePools as unknown as ResourcePool[];

export default {
  component: ResourcePoolCard,
  title: 'ResourcePoolCard',
};

export const Default = (): React.ReactNode => {
  return <ResourcePoolCard
    gpuContainerStates={[]}
    resourcePool={rps.random()}
    totalGpuSlots={3}
  />;
};
