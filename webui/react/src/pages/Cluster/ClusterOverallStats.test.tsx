import { render } from '@testing-library/react';
import React from 'react';

import StoreProvider from 'contexts/Store';
import { ExperimentsProvider } from 'stores/experiments';
import { ResourcePoolsProvider } from 'stores/resourcePools';
import { TasksProvider } from 'stores/tasks';

import { ClusterOverallStats } from './ClusterOverallStats';

jest.mock('services/api', () => ({
  getActiveTasks: () => Promise.resolve({ commands: 0, notebooks: 0, shells: 0, tensorboards: 0 }),
  getExperiments: () => Promise.resolve({ experiments: [], pagination: { total: 0 } }),
}));

const setup = () => {
  const view = render(
    <StoreProvider>
      <ExperimentsProvider>
        <TasksProvider>
          <ResourcePoolsProvider>
            <ClusterOverallStats />
          </ResourcePoolsProvider>
        </TasksProvider>
      </ExperimentsProvider>
    </StoreProvider>,
  );
  return { view };
};

describe('ClusterOverallStats', () => {
  it('displays cluster overall stats ', () => {
    const { view } = setup();
    expect(view.getByText('Connected Agents')).toBeInTheDocument();
  });
});
