import { waitFor } from '@testing-library/dom';
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TooltipProps } from 'antd/es/tooltip';
import React, { useCallback, useEffect, useState } from 'react';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import usersStore from 'stores/users';
import { DetailedUser } from 'types';

import UserAvatar, { Props } from './UserAvatar';

const testUsers: DetailedUser[] = [
  {
    displayName: 'Bugs Bunny',
    id: 44,
    isActive: true,
    isAdmin: true,
    username: 'elmerFudd01',
  },
];

jest.mock('services/api', () => ({
  getUsers: () => Promise.resolve({ users: testUsers }),
}));

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  /**
   * We need to mock Tooltip in order to override getPopupContainer to null. getPopupContainer
   * sets the DOM container and if this prop is set, the popup div may not be available in the body
   */
  const Tooltip = (props: TooltipProps) => {
    return (
      <antd.Tooltip
        {...props}
        getPopupContainer={(trigger: HTMLElement) => trigger}
        mouseEnterDelay={0}
      />
    );
  };

  return {
    __esModule: true,
    ...antd,
    Tooltip,
  };
});

const Component = ({ user }: Partial<Props> = {}) => {
  const [canceler] = useState(new AbortController());
  const asyncFetch = useCallback(async () => {
    await usersStore.ensureUsersFetched(canceler);
  }, [canceler]);

  useEffect(() => {
    asyncFetch();
    usersStore.updateCurrentUser(44);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return <UserAvatar hideTooltip={false} user={user} />;
};

const setup = (testUser: DetailedUser) => {
  const user = userEvent.setup();

  const view = render(
    <UIProvider>
      <Component user={testUser} />
    </UIProvider>,
  );

  return { user, view };
};

describe('UserAvatar', () => {
  it('should display initials of name', async () => {
    const testUser = testUsers[0];
    await waitFor(() => setup(testUser));
    expect(await screen.findByText('BB')).toBeInTheDocument();
  });

  it('should display name on hover', async () => {
    const testUser = testUsers[0];
    const { user } = await waitFor(() => setup(testUser));
    await act(async () => await user.hover(await screen.findByText('BB')));
    expect(await screen.getByText(testUser.displayName || '')).toBeInTheDocument();
  });
});
