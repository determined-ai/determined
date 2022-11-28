import React, { ReactElement, ReactNode } from 'react';

import { AgentsProvider } from './agents';
import { WorkspacesProvider } from './workspaces';

export const StoreContext = ({ children }: { children: ReactNode }): ReactElement => (
  <AgentsProvider>
    <WorkspacesProvider>{children}</WorkspacesProvider>
  </AgentsProvider>
);
