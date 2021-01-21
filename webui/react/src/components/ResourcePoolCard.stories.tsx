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
    containerStates={[]}
    resourcePool={resourcePools[Math.floor(Math.random()*resourcePools.length)]} />;
};
