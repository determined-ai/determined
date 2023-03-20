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
import { ProjectsProvider } from './projects';
import { WorkspacesProvider } from './workspaces';

export const StoreProvider = ({ children }: { children: ReactNode }): ReactElement => (
  <UIProvider>
    <ClusterProvider>
      <WorkspacesProvider>
        <ProjectsProvider>{children}</ProjectsProvider>
      </WorkspacesProvider>
    </ClusterProvider>
  </UIProvider>
);
