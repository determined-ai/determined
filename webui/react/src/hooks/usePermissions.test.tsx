import { render, screen } from '@testing-library/react';
import React from 'react';

import usePermissions from 'hooks/usePermissions';
import { V1PermissionType } from 'services/api-ts-sdk/api';
import { GetWorkspaceParams } from 'services/types';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
// import { Loaded } from 'utils/loadable';
// import { observable } from 'utils/observable';

const workspace = {
  id: 10,
  name: 'Test Workspace',
};

vi.mock('services/api', () => ({
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
    <UIProvider>
      <PermissionRenderer workspaceId={1} />
    </UIProvider>,
  );
};

describe('usePermissions', () => {
  it('should have OSS permissions', async () => {
    // vi.doMock('stores/determinedInfo', async (importOriginal) => {
    //   const loadable = await import('utils/loadable');
    //   const observable = await import('utils/observable');
    //   const store = {
    //     info: observable.observable(loadable.Loaded({
    //       rbacEnabled: false,
    //     })),
    //   };
    //   return {
    //     ...(await importOriginal<typeof import('stores/determinedInfo')>()),
    //     default: store,
    //   };
    // });

    await setup();

    // any user permission in OSS
    expect(screen.queryByText('canCreateWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canCreateProject')).toBeInTheDocument();
    expect(screen.queryByText('canViewWorkspace')).toBeInTheDocument();

    expect(screen.queryByText('canModifyWorkspace')).not.toBeInTheDocument();
    expect(screen.queryByText('canDeleteWorkspace')).not.toBeInTheDocument();
  });

  // it('should have read permissions', async () => {
  //   vi.mocked(useFeature).mockReturnValue({
  //     isOn: (f: string) => ['rbac', 'mock_permissions_read'].includes(f),
  //   });
  //   await setup();
  //
  //   // read permissions available
  //   expect(screen.queryByText('canViewWorkspace')).toBeInTheDocument();
  //
  //   // create / update / delete permissions permissions not available
  //   expect(screen.queryByText('canCreateWorkspace')).not.toBeInTheDocument();
  //   expect(screen.queryByText('canCreateProject')).not.toBeInTheDocument();
  //   expect(screen.queryByText('canModifyWorkspace')).not.toBeInTheDocument();
  //   expect(screen.queryByText('canDeleteWorkspace')).not.toBeInTheDocument();
  // });

  it('should have create/read/update/delete permissions', async () => {
    vi.doMock('stores/permissions', async (importOriginal) => {
      const loadable = await import('utils/loadable');
      const observable = await import('utils/observable');
      const assigned = observable.observable(
        loadable.Loaded([
          {
            roleId: 1,
            scopeCluster: true,
          },
        ]),
      );
      const roles = observable.observable(
        loadable.Loaded([
          {
            id: 1,
            name: 'TestClusterAdmin',
            permissions: [
              {
                id: 'PERMISSION_TYPE_CREATE_WORKSPACE',
                scopeCluster: true,
                scopeWorkspace: false,
              },
              {
                id: V1PermissionType.DELETEWORKSPACE,
                scopeCluster: true,
                scopeWorkspace: false,
              },
              {
                id: 'PERMISSION_TYPE_UPDATE_WORKSPACE',
                scopeCluster: true,
                scopeWorkspace: false,
              },
            ],
            scopeCluster: true,
          },
        ]),
      );
      return {
        ...(await importOriginal<typeof import('stores/permissions')>()),
        default: {
          myAssignments: assigned,
          myRoles: roles,
          permissions: observable.observable([assigned, roles]),
        },
      };
    });

    vi.doMock('stores/determinedInfo', async (importOriginal) => {
      const loadable = await import('utils/loadable');
      const observable = await import('utils/observable');
      const store = {
        info: observable.observable(
          loadable.Loaded({
            rbacEnabled: true,
          }),
        ),
      };
      return {
        ...(await importOriginal<typeof import('stores/determinedInfo')>()),
        default: store,
      };
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
