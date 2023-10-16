import { render, screen, waitFor } from '@testing-library/react';
import React, { Suspense } from 'react';

import { UIProvider, ThemeProvider } from 'components/kit/Theme';
import resourcePools from 'fixtures/responses/cluster/resource-pools.json';
import { ResourcePool } from 'types';
import { theme, isDarkMode } from 'utils/tests/getTheme';
import { RenderAllocationBarResourcePool } from './ResourcePoolCard';

const rps = resourcePools as unknown as ResourcePool[];

vi.mock('services/api', () => ({
  getAgents: () => Promise.resolve([]),
  getResourcePools: () => Promise.resolve({}),
}));

const setup = (pool: ResourcePool) => {
  const view = render(
    <ThemeProvider>
      <UIProvider theme={theme} darkMode={isDarkMode}>
        <Suspense>
          <RenderAllocationBarResourcePool resourcePool={pool} />
        </Suspense>
      </UIProvider>
    </ThemeProvider>
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
