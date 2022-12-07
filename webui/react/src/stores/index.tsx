import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { ExperimentsProvider } from './experiments';
import { ProjectsProvider } from './projects';
import { ResourcePoolsProvider } from './resourcePools';
import { TasksProvider } from './tasks';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
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
);
