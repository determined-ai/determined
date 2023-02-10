import React, { ReactElement, ReactNode } from 'react';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
// import {
//   observable,
//   Observable,
//   useObservable,
//   useValueMemoizedObservable,
//   WritableObservable,
// } from 'utils/observable';

import { ClusterProvider } from './cluster';
import { KnownRolesProvider } from './knowRoles';
import { ProjectsProvider } from './projects';
import { UsersProvider } from './users';
import { WorkspacesProvider } from './workspaces';

export const StoreProvider = ({ children }: { children: ReactNode }): ReactElement => (
  <UIProvider>
    <ClusterProvider>
      <UsersProvider>
        <WorkspacesProvider>
          <KnownRolesProvider>
            <ProjectsProvider>{children}</ProjectsProvider>
          </KnownRolesProvider>
        </WorkspacesProvider>
      </UsersProvider>
    </ClusterProvider>
  </UIProvider>
);
