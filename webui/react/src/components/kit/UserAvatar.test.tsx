import { waitFor } from '@testing-library/dom';
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect, useState } from 'react';

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

vi.mock('services/api', () => ({
  getUsers: () => Promise.resolve({ users: testUsers }),
}));

vi.mock('components/kit/Tooltip');

const Component = ({ user }: Partial<Props> = {}) => {
  const [canceler] = useState(new AbortController());

  useEffect(() => {
    usersStore.ensureUsersFetched(canceler);
    usersStore.updateCurrentUser(44);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canceler]);

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
