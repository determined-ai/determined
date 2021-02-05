import { number, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import resourcePoolResponse from 'fixtures/responses/resource-pools/a.json';
import { ResourcePool } from 'types';

import ResourcePoolDetails from './ResourcePoolDetails';

const resourcePools = resourcePoolResponse.resourcePools as unknown as ResourcePool[];

export default {
  component: ResourcePoolDetails,
  decorators: [ withKnobs ],
  title: 'ResourcePoolDetails',
};

export const Default = (): React.ReactNode => {
  return <ResourcePoolDetails
    resourcePool={resourcePools[number(
      'ResourcePool Index',
      0,
      { max: resourcePools.length - 1, min: 0, step: 1 },
    )]}
    visible={true}
  />;
};
