import { screen } from '@testing-library/react';

import { V1PermissionType } from 'services/api-ts-sdk/api';

import { setup } from './usePermissions.common';

vi.mock('stores/determinedInfo', async (importOriginal) => {
  const observable = await import('utils/observable');
  const store = {
    info: observable.observable({
      rbacEnabled: true,
    }),
  };
  return {
    ...(await importOriginal<typeof import('stores/determinedInfo')>()),
    default: store,
  };
});

vi.mock('stores/permissions', async (importOriginal) => {
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
            id: V1PermissionType.CREATEWORKSPACE,
          },
          {
            id: V1PermissionType.CREATEPROJECT,
          },
          {
            id: V1PermissionType.DELETEWORKSPACE,
          },
          {
            id: V1PermissionType.UPDATEWORKSPACE,
          },
          {
            id: V1PermissionType.VIEWWORKSPACE,
          },
        ],
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

describe('usePermissions for RBAC admin user', () => {
  it('should have create/read/update/delete permissions', async () => {
    await setup();

    // sample create / read / update / delete permissions all available
    expect(screen.queryByText('canCreateWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canCreateProject')).toBeInTheDocument();
    expect(screen.queryByText('canModifyWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canDeleteWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canViewWorkspace')).toBeInTheDocument();
  });
});
