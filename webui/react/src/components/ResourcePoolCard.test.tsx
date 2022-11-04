import { render } from '@testing-library/react';
import React from 'react';

import StoreProvider from 'contexts/Store';
import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { ResourcePool } from 'types';

import { RenderAllocationBarResourcePool } from './ResourcePoolCard';

const rps = resourcePools as unknown as ResourcePool[];

const setup = (pool: ResourcePool) => {
  const view = render(
    <StoreProvider>
      <RenderAllocationBarResourcePool resourcePool={pool} />
    </StoreProvider>,
  );
  return { view };
};

describe('AllocationBarResourcePool', () => {
  it('displays resource pool slot allocation bar ', () => {
    rps.forEach((pool) => {
      const { view } = setup(pool);
      expect(view.getAllByText('Allocated', { exact: false }).length).toBeGreaterThan(0);
    });
  });
});
