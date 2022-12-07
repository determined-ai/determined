import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { ExperimentsProvider } from './experiments';
import { ProjectsProvider } from './projects';
<<<<<<< HEAD
import { UserRolesProvider } from './userRoles';
=======
import { ResourcePoolsProvider } from './resourcePools';
import { TasksProvider } from './tasks';
>>>>>>> master
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <AgentsProvider>
<<<<<<< HEAD
    <WorkspacesProvider>
      <ProjectsProvider>
        <UserRolesProvider>{children}</UserRolesProvider>
      </ProjectsProvider>
    </WorkspacesProvider>
=======
    <ExperimentsProvider>
      <TasksProvider>
        <WorkspacesProvider>
          <ResourcePoolsProvider>
            <ProjectsProvider>{children}</ProjectsProvider>
          </ResourcePoolsProvider>
        </WorkspacesProvider>
      </TasksProvider>
    </ExperimentsProvider>
>>>>>>> master
  </AgentsProvider>
);
