import React, { ReactElement, ReactNode } from 'react';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
// import {
//   observable,
//   Observable,
//   useObservable,
//   useValueMemoizedObservable,
//   WritableObservable,
// } from 'utils/observable';

import { AgentsProvider } from './agents';
import { AuthProvider } from './auth';
import { DeterminedInfoProvider } from './determinedInfo';
import { ExperimentsProvider } from './experiments';
import { KnownRolesProvider } from './knowRoles';
import { ProjectsProvider } from './projects';
import { ResourcePoolsProvider } from './resourcePools';
import { TasksProvider } from './tasks';
import { UserRolesProvider } from './userRoles';
import { UsersProvider } from './users';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <UIProvider>
    <AgentsProvider>
      <UsersProvider>
        <AuthProvider>
          <ExperimentsProvider>
            <TasksProvider>
              <WorkspacesProvider>
                <ResourcePoolsProvider>
                  <DeterminedInfoProvider>
                    <UserRolesProvider>
                      <KnownRolesProvider>
                        <ProjectsProvider>{children}</ProjectsProvider>
                      </KnownRolesProvider>
                    </UserRolesProvider>
                  </DeterminedInfoProvider>
                </ResourcePoolsProvider>
              </WorkspacesProvider>
            </TasksProvider>
          </ExperimentsProvider>
        </AuthProvider>
      </UsersProvider>
    </AgentsProvider>
  </UIProvider>
);
