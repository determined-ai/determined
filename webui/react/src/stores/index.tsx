import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { ExperimentsProvider } from './experiments';
import { OmnibarProvider } from './omnibar';
import { ProjectsProvider } from './projects';
import { ResourcePoolsProvider } from './resourcePools';
import { TasksProvider } from './tasks';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <OmnibarProvider>
    <AgentsProvider>
      <ExperimentsProvider>
        <TasksProvider>
          <WorkspacesProvider>
            <ResourcePoolsProvider>
              <ProjectsProvider>{children}</ProjectsProvider>
            </ResourcePoolsProvider>
          </WorkspacesProvider>
        </TasksProvider>
      </ExperimentsProvider>
    </AgentsProvider>
  </OmnibarProvider>
);
