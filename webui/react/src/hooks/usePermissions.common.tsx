import { render, RenderResult } from '@testing-library/react';
import React from 'react';

import usePermissions from 'hooks/usePermissions';
import { ActionWorkspaceParams } from 'services/types';
import { StoreProvider as UIProvider } from 'stores/contexts/UI';

export const workspace = {
  id: 10,
  name: 'Test Workspace',
};

vi.mock('services/api', () => ({
  getWorkspace: (params: ActionWorkspaceParams) => {
    return {
      ...workspace,
      id: params.workspaceId,
    };
  },
}));

vi.mock('stores/users', async (importOriginal) => {
  const loadable = await import('utils/loadable');
  const observable = await import('utils/observable');
  const store = {
    currentUser: observable.observable(
      loadable.Loaded({
        admin: false,
        id: 101,
      }),
    ),
  };
  return {
    ...(await importOriginal<typeof import('stores/users')>()),
    default: store,
  };
});

interface Props {
  workspaceId: number;
}

const PermissionRenderer: React.FC<Props> = () => {
  const {
    canCreateProject,
    canCreateWorkspace,
    canDeleteWorkspace,
    canModifyWorkspace,
    canViewWorkspace,
  } = usePermissions();

  return (
    <ul>
      <li>{canCreateProject({ workspace }) && 'canCreateProject'}</li>
      <li>{canCreateWorkspace && 'canCreateWorkspace'}</li>
      <li>{canDeleteWorkspace({ workspace }) && 'canDeleteWorkspace'}</li>
      <li>{canModifyWorkspace({ workspace }) && 'canModifyWorkspace'}</li>
      <li>{canViewWorkspace({ workspace }) && 'canViewWorkspace'}</li>
    </ul>
  );
};

export const setup = async (): Promise<RenderResult> => {
  return await render(
    <UIProvider>
      <PermissionRenderer workspaceId={1} />
    </UIProvider>,
  );
};
