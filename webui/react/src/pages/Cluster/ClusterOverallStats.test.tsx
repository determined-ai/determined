import { render } from '@testing-library/react';

import { ClusterOverallStats } from 'pages/Cluster/ClusterOverallStats';
import { StoreProvider as UIProvider } from 'stores/contexts/UI';

vi.mock('services/api', () => ({
  getActiveTasks: () => Promise.resolve({ commands: 0, notebooks: 0, shells: 0, tensorboards: 0 }),
  getAgents: () => Promise.resolve([]),
  getExperiments: () => Promise.resolve({ experiments: [], pagination: { total: 0 } }),
  getResourcePools: () => Promise.resolve({}),
}));

const setup = () => {
  const view = render(
    <UIProvider>
      <ClusterOverallStats />
    </UIProvider>,
  );
  return { view };
};

describe('ClusterOverallStats', () => {
  it('displays cluster overall stats ', () => {
    const { view } = setup();
    expect(view.getByText('Connected Agents')).toBeInTheDocument();
  });
});
