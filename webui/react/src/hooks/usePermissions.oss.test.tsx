import { screen } from '@testing-library/react';

import { setup } from './usePermissions.common';

vi.mock('stores/determinedInfo', async (importOriginal) => {
  const observable = await import('utils/observable');
  const store = {
    info: observable.observable({
      rbacEnabled: false,
    }),
  };
  return {
    ...(await importOriginal<typeof import('stores/determinedInfo')>()),
    default: store,
  };
});

describe('usePermissions for OSS', () => {
  it('should have OSS permissions', async () => {
    await setup();

    // any user permission in OSS
    expect(screen.queryByText('canCreateWorkspace')).toBeInTheDocument();
    expect(screen.queryByText('canCreateProject')).toBeInTheDocument();
    expect(screen.queryByText('canViewWorkspace')).toBeInTheDocument();

    expect(screen.queryByText('canModifyWorkspace')).not.toBeInTheDocument();
    expect(screen.queryByText('canDeleteWorkspace')).not.toBeInTheDocument();
  });
});
