import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { DeterminedInfoProvider } from './determinedInfo';
import { ExperimentsProvider } from './experiments';
import { ProjectsProvider } from './projects';
import { ResourcePoolsProvider } from './resourcePools';
import { TasksProvider } from './tasks';
import { UserRolesProvider } from './userRoles';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <AgentsProvider>
    <ExperimentsProvider>
      <TasksProvider>
        <WorkspacesProvider>
          <ResourcePoolsProvider>
            <DeterminedInfoProvider>
            <UserRolesProvider>
              <ProjectsProvider>{children}</ProjectsProvider>
              </UserRolesProvider>
            </DeterminedInfoProvider>
          </ResourcePoolsProvider>
        </WorkspacesProvider>
      </TasksProvider>
    </ExperimentsProvider>
  </AgentsProvider>
);
