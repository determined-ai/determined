import { Meta, Story } from '@storybook/react';
import React from 'react';

import resourcePoolResponse from 'fixtures/responses/resource-pools/a.json';
import { ResourcePool } from 'types';

import ResourcePoolDetails from './ResourcePoolDetails';

const resourcePools = resourcePoolResponse.resourcePools as unknown as ResourcePool[];

type ResourcePoolDetailsProps = React.ComponentProps<typeof ResourcePoolDetails>;

export default {
  argTypes: {
    poolNumber: { control: { max: resourcePools.length - 1, min: 0, step: 1, type: 'number' } },
    resourcePool: { control: { type: null } },
    visible: { control: { type: null } },
  },
  component: ResourcePoolDetails,
  title: 'Determined/ResourcePoolDetails',
} as Meta<typeof ResourcePoolDetails>;

export const Default: Story<ResourcePoolDetailsProps & { poolNumber: number }> = ({
  poolNumber,
  ...args
}) => <ResourcePoolDetails {...args} resourcePool={resourcePools[poolNumber]} visible={true} />;

Default.args = { poolNumber: 0 };
