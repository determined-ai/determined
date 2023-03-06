import { render, screen, waitFor } from '@testing-library/react';
import React, { Suspense } from 'react';

import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { ClusterProvider } from 'stores/cluster';
import { ResourcePool } from 'types';

import { RenderAllocationBarResourcePool } from './ResourcePoolCard';

const rps = resourcePools as unknown as ResourcePool[];

jest.mock('services/api', () => ({
  getAgents: () => Promise.resolve([]),
  getResourcePools: () => Promise.resolve({}),
}));

const setup = (pool: ResourcePool) => {
  const view = render(
    <UIProvider>
      <ClusterProvider>
        <Suspense>
          <RenderAllocationBarResourcePool resourcePool={pool} />
        </Suspense>
      </ClusterProvider>
    </UIProvider>,
  );
  return { view };
};

describe('AllocationBarResourcePool', () => {
  it('displays resource pool slot allocation bar ', async () => {
    await rps.forEach(async (pool) => {
      const { view } = setup(pool);
      await waitFor(() => expect(screen.getByText('Allocated')).toBeInTheDocument());
      expect(view.getAllByText('Allocated', { exact: false }).length).toBeGreaterThan(0);
    });
  });
});
