import { render, screen, waitFor } from '@testing-library/react';
import React, { Suspense } from 'react';

import { RenderAllocationBarResourcePool } from 'components/ResourcePoolCard';
import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { StoreProvider as UIProvider } from 'stores/contexts/UI';
import { ResourcePool } from 'types';

const rps = resourcePools as unknown as ResourcePool[];

vi.mock('services/api', () => ({
  getAgents: () => Promise.resolve([]),
  getResourcePools: () => Promise.resolve({}),
}));

const setup = (pool: ResourcePool) => {
  const view = render(
    <UIProvider>
      <Suspense>
        <RenderAllocationBarResourcePool resourcePool={pool} />
      </Suspense>
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
