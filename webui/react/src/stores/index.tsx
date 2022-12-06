import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { OmnibarProvider } from './omnibar';
import { ProjectsProvider } from './projects';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <OmnibarProvider>
    <AgentsProvider>
      <WorkspacesProvider>
        <ProjectsProvider>{children}</ProjectsProvider>
      </WorkspacesProvider>
    </AgentsProvider>
  </OmnibarProvider>
);
