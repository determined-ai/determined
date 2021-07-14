import React from 'react';

import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { ResourcePool, ResourceType } from 'types';

import ResourcePoolCard from './ResourcePoolCard';

const rps = resourcePools as unknown as ResourcePool[];

export default {
  component: ResourcePoolCard,
  title: 'ResourcePoolCard',
};

export const Default = (): React.ReactNode => {
  return <ResourcePoolCard
    computeContainerStates={[]}
    resourcePool={rps.random()}
    resourceType={ResourceType.GPU}
    totalComputeSlots={3}
  />;
};

export const CPU = (): React.ReactNode => {
  return <ResourcePoolCard
    computeContainerStates={[]}
    resourcePool={rps.random()}
    resourceType={ResourceType.CPU}
    totalComputeSlots={3}
  />;
};

export const Aux = (): React.ReactNode => {
  return <ResourcePoolCard
    computeContainerStates={[]}
    resourcePool={rps.random()}
    resourceType={ResourceType.UNSPECIFIED}
    totalComputeSlots={0}
  />;
};
