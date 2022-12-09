import { waitFor } from '@testing-library/dom';
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TooltipProps } from 'antd/es/tooltip';
import React, { useCallback, useEffect, useState } from 'react';

import StoreProvider from 'contexts/Store';
import { useFetchUsers, UsersProvider } from 'stores/users';

import UserAvatar, { Props } from './UserAvatar';

const testUsers = [
  {
    displayName: 'Bugs Bunny',
    id: 44,
    initials: 'BB',
    isActive: true,
    isAdmin: true,
    userId: 44,
    username: 'elmerFudd01',
  },
];

jest.mock('services/api', () => ({
  getUsers: () => Promise.resolve({ users: testUsers }),
}));

jest.mock('antd', () => {
  const antd = jest.requireActual('antd');

  /** We need to mock Tooltip in order to override getPopupContainer to null. getPopupContainer
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

const Component = ({ hideTooltip = false, userId, ...props }: Partial<Props> = {}) => {
  const [canceler] = useState(new AbortController());
  const fetchUsers = useFetchUsers(canceler);
  const asyncFetch = useCallback(async () => {
    await fetchUsers();
  }, [fetchUsers]);

  useEffect(() => {
    asyncFetch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return <UserAvatar hideTooltip={hideTooltip} userId={userId} {...props} />;
};

const setup = ({ hideTooltip = false, userId, ...props }: Partial<Props> = {}) => {
  const user = userEvent.setup();

  const view = render(
    <StoreProvider>
      <UsersProvider>
        <Component hideTooltip={hideTooltip} userId={userId} {...props} />
      </UsersProvider>
    </StoreProvider>,
  );

  return { user, view };
};

describe('UserAvatar', () => {
  it('should display initials of name', async () => {
    const testUser = testUsers[0];
    await waitFor(() => setup(testUser));
    expect(await screen.findByText(testUser.initials)).toBeInTheDocument();
  });

  it('should display name on hover', async () => {
    const testUser = testUsers[0];
    const { user } = await waitFor(() => setup(testUser));
    await act(async () => await user.hover(await screen.findByText(testUser.initials)));
    expect(await screen.getByText(testUser.displayName)).toBeInTheDocument();
  });
});
