import { render, screen } from '@testing-library/react';
import React from 'react';

import StoreProvider from 'contexts/Store';
import { GetWorkspaceParams } from 'services/types';
import { AuthProvider } from 'stores/auth';
import { UsersProvider } from 'stores/users';
import { UserRolesProvider } from 'stores/userRoles';

import useFeature from './useFeature';
import usePermissions from './usePermissions';

const workspace = {
  id: 10,
  name: 'Test Workspace',
};
jest.mock('hooks/useFeature');
jest.mock('services/api', () => ({
  getWorkspace: (params: GetWorkspaceParams) => {
    return {
      ...workspace,
      id: params.id,
    };
  },
}));

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

const setup = async () => {
  return await render(
    <StoreProvider>
      <UsersProvider>
        <AuthProvider>
          <UserRolesProvider>
            <PermissionRenderer workspaceId={1} />
          </UserRolesProvider>
        </AuthProvider>
      </UsersProvider>
    </StoreProvider>,
  );
};

describe('usePermissions', () => {
  it('should have OSS permissions', async () => {
    (useFeature as jest.Mock).mockReturnValue({
      isOn: () => false,
    });
    await setup();

    // any user permission in OSS
    expect(screen.queryByText('canCreateWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canCreateProject')).toBeInTheDocument();
    expect(screen.queryByText('canViewWorkspace')).toBeInTheDocument();

    expect(screen.queryByText('canModifyWorkspace')).not.toBeInTheDocument();
    expect(screen.queryByText('canDeleteWorkspace')).not.toBeInTheDocument();
  });

  it('should have read permissions', async () => {
    (useFeature as jest.Mock).mockReturnValue({
      isOn: (f: string) => ['rbac', 'mock_permissions_read'].includes(f),
    });
    await setup();

    // read permissions available
    expect(screen.queryByText('canViewWorkspace')).toBeInTheDocument();

    // create / update / delete permissions permissions not available
    expect(screen.queryByText('canCreateWorkspace')).not.toBeInTheDocument();
    expect(screen.queryByText('canCreateProject')).not.toBeInTheDocument();
    expect(screen.queryByText('canModifyWorkspace')).not.toBeInTheDocument();
    expect(screen.queryByText('canDeleteWorkspace')).not.toBeInTheDocument();
  });

  it('should have create/read/update/delete permissions', async () => {
    (useFeature as jest.Mock).mockReturnValue({
      isOn: (f: string) => ['rbac', 'mock_permissions_all'].includes(f),
    });
    await setup();

    // sample create / read / update / delete permissions all available
    expect(screen.queryByText('canCreateWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canCreateProject')).toBeInTheDocument();
    expect(screen.queryByText('canModifyWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canDeleteWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canViewWorkspace')).toBeInTheDocument();
  });
});
