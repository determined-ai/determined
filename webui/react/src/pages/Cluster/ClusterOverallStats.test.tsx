import { render } from '@testing-library/react';
import React from 'react';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { AuthProvider } from 'stores/auth';
import { ResourcePoolsProvider } from 'stores/resourcePools';
import { TasksProvider } from 'stores/tasks';
import { UserRolesProvider } from 'stores/userRoles';
import { UsersProvider } from 'stores/users';

import { ClusterOverallStats } from './ClusterOverallStats';

jest.mock('services/api', () => ({
  getActiveTasks: () => Promise.resolve({ commands: 0, notebooks: 0, shells: 0, tensorboards: 0 }),
  getExperiments: () => Promise.resolve({ experiments: [], pagination: { total: 0 } }),
}));

const setup = () => {
  const view = render(
    <UIProvider>
      <AuthProvider>
        <UsersProvider>
          <UserRolesProvider>
            <TasksProvider>
              <ResourcePoolsProvider>
                <ClusterOverallStats />
              </ResourcePoolsProvider>
            </TasksProvider>
          </UserRolesProvider>
        </UsersProvider>
      </AuthProvider>
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
