import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { ProjectsProvider } from './projects';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <AgentsProvider>
    <WorkspacesProvider>
      <ProjectsProvider>{children}</ProjectsProvider>
    </WorkspacesProvider>
  </AgentsProvider>
);
