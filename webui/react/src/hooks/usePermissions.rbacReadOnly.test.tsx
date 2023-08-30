import { screen } from '@testing-library/react';

import { setup } from 'hooks/usePermissions.common';
import { V1PermissionType } from 'services/api-ts-sdk/api';

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
        name: 'TestReadOnly',
        permissions: [
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

describe('usePermissions for RBAC read-only user', () => {
  it('should have read permissions', async () => {
    await setup();

    // read permissions available
    expect(screen.queryByText('canViewWorkspace')).toBeInTheDocument();

    // create / update / delete permissions permissions not available
    expect(screen.queryByText('canCreateWorkspace')).not.toBeInTheDocument();
    expect(screen.queryByText('canCreateProject')).not.toBeInTheDocument();
    expect(screen.queryByText('canModifyWorkspace')).not.toBeInTheDocument();
    expect(screen.queryByText('canDeleteWorkspace')).not.toBeInTheDocument();
  });
});
