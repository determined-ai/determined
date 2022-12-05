import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { ProjectsProvider } from './projects';
import { ResourcePoolsProvider } from './resourcePools';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <AgentsProvider>
    <WorkspacesProvider>
      <ResourcePoolsProvider>
        <ProjectsProvider>{children}</ProjectsProvider>
      </ResourcePoolsProvider>
    </WorkspacesProvider>
  </AgentsProvider>
);
