import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { AuthProvider } from './auth';
import { DeterminedInfoProvider } from './determinedInfo';
import { ExperimentsProvider } from './experiments';
import { ProjectsProvider } from './projects';
import { ResourcePoolsProvider } from './resourcePools';
import { TasksProvider } from './tasks';
import { UsersProvider } from './users';
import { UserRolesProvider } from './userRoles';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <AgentsProvider>
    <UsersProvider>
      <AuthProvider>
        <ExperimentsProvider>
          <TasksProvider>
            <WorkspacesProvider>
              <ResourcePoolsProvider>
                <DeterminedInfoProvider>
                  <UserRolesProvider>
                    <ProjectsProvider>{children}</ProjectsProvider>
                  </UserRolesProvider>
                </DeterminedInfoProvider>
              </ResourcePoolsProvider>
            </WorkspacesProvider>
          </TasksProvider>
        </ExperimentsProvider>
      </AuthProvider>
    </UsersProvider>
  </AgentsProvider>
);
