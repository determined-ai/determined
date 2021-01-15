import React from 'react';

import { getResourcePools } from 'services/api';

import ResourcePoolCard from './ResourcePoolCard';

const resourcePools = getResourcePools();

export default {
  component: ResourcePoolCard,
  title: 'ResourcePoolCard',
};

export const Default = (): React.ReactNode => {
  return <ResourcePoolCard
    containerStates={[]}
    resourcePool={resourcePools[Math.floor(Math.random()*resourcePools.length)]} />;
};
