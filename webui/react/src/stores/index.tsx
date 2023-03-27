import React, { ReactElement, ReactNode } from 'react';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';

import { ClusterProvider } from './cluster';
import { WorkspacesProvider } from './workspaces';

export const StoreProvider = ({ children }: { children: ReactNode }): ReactElement => (
  <UIProvider>
    <ClusterProvider>
      <WorkspacesProvider>{children}</WorkspacesProvider>
    </ClusterProvider>
  </UIProvider>
);
