import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { AuthProvider } from './auth';
import { ProjectsProvider } from './projects';
import { UsersProvider } from './users';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <AgentsProvider>
    <UsersProvider>
      <AuthProvider>
        <WorkspacesProvider>
          <ProjectsProvider>{children}</ProjectsProvider>
        </WorkspacesProvider>
      </AuthProvider>
    </UsersProvider>
  </AgentsProvider>
);
