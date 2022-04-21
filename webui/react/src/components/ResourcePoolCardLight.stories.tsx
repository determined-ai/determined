import React from 'react';

import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { ResourcePool } from 'types';

import ResourcePoolCardLight from './ResourcePoolCardLight';

const rps = resourcePools as unknown as ResourcePool[];

export default {
  component: ResourcePoolCardLight,
  title: 'ResourcePoolCardLight',
};

export const Default = (): React.ReactNode => (
  <ResourcePoolCardLight
    resourcePool={rps.random()}
  />
);

export const CPU = (): React.ReactNode => (
  <ResourcePoolCardLight
    resourcePool={rps.random()}
  />
);

export const Aux = (): React.ReactNode => (
  <ResourcePoolCardLight
    resourcePool={rps.random()}
  />
);
