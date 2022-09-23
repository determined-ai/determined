import { Meta, Story } from '@storybook/react';
import React from 'react';

import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { ResourcePool } from 'types';

import ResourcePoolCard from './ResourcePoolCard';

const rps = resourcePools as unknown as ResourcePool[];

export default {
  argTypes: { pool: { control: { max: rps.length - 1, min: 0, step: 1, type: 'range' } } },
  component: ResourcePoolCard,
  title: 'Determined/Cards/ResourcePoolCard',
} as Meta<typeof ResourcePoolCard>;

type ResourcePoolCardProps = React.ComponentProps<typeof ResourcePoolCard>;

export const Default: Story<ResourcePoolCardProps & { pool: number }> = ({ pool, ...args }) => (
  <ResourcePoolCard {...args} resourcePool={rps[pool]} />
);

Default.args = { pool: 0 };
