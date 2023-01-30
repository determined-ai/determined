import React, { ReactElement, ReactNode } from 'react';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
// import {
//   observable,
//   Observable,
//   useObservable,
//   useValueMemoizedObservable,
//   WritableObservable,
// } from 'utils/observable';

import { AuthProvider } from './auth';
import { ClusterProvider } from './cluster';
import { DeterminedInfoProvider } from './determinedInfo';
import { KnownRolesProvider } from './knowRoles';
import { ProjectsProvider } from './projects';
import { TasksProvider } from './tasks';
import { UserRolesProvider } from './userRoles';
import { UsersProvider } from './users';
import { WorkspacesProvider } from './workspaces';

export const StoreProvider = ({ children }: { children: ReactNode }): ReactElement => (
  <UIProvider>
    <ClusterProvider>
      <UsersProvider>
        <AuthProvider>
          <TasksProvider>
            <WorkspacesProvider>
              <DeterminedInfoProvider>
                <UserRolesProvider>
                  <KnownRolesProvider>
                    <ProjectsProvider>{children}</ProjectsProvider>
                  </KnownRolesProvider>
                </UserRolesProvider>
              </DeterminedInfoProvider>
            </WorkspacesProvider>
          </TasksProvider>
        </AuthProvider>
      </UsersProvider>
    </ClusterProvider>
  </UIProvider>
);
