import React from 'react';

import { getResourcePoolSamples } from 'services/api';

import ResourcePoolCard from './ResourcePoolCard';

const resourcePools = getResourcePoolSamples();

export default {
  component: ResourcePoolCard,
  title: 'ResourcePoolCard',
};

export const Default = (): React.ReactNode => {
  return <ResourcePoolCard
    gpuContainerStates={[]}
    resourcePool={resourcePools.random()}
    totalGpuSlots={3}
  />;
};
