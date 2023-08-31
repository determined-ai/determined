import { waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useCallback, useEffect } from 'react';

import { patchUser as mockPatchUser } from 'services/api';
import { PatchUserParams } from 'services/types';
import authStore from 'stores/auth';
import { StoreProvider as UIProvider } from 'stores/contexts/UI';
import userStore from 'stores/users';
import userSettings from 'stores/userSettings';
import { DetailedUser } from 'types';

import { ConfirmationProvider } from './kit/useConfirm';
import UserSettings from './UserSettings';

vi.mock('services/api', () => ({
  getUsers: () =>
    Promise.resolve({
      users: [
        {
          displayName: 'Test Name',
          id: 1,
          isActive: true,
          isAdmin: false,
          username: 'test_username1',
        },
      ],
    }),
  patchUser: vi.fn((params: PatchUserParams) =>
    Promise.resolve({
      displayName: params.userParams.displayName,
      id: 1,
      isActive: true,
    }),
  ),
}));

const user = userEvent.setup();

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const CURRENT_USER: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: 1,
  isActive: true,
  isAdmin: false,
  username: USERNAME,
};

const Container: React.FC = () => {
  const loadUsers = useCallback(() => {
    userStore.updateCurrentUser(CURRENT_USER);
  }, []);

  useEffect(() => {
    authStore.setAuth({ isAuthenticated: true });
    userSettings.startPolling();
    return userStore.fetchUsers();
  }, []);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  return (
    <UserSettings
      show={true}
      onClose={() => {
        return null;
      }}
    />
  );
};

const setup = () =>
  render(
    <UIProvider>
      <ConfirmationProvider>
        <Container />
      </ConfirmationProvider>
    </UIProvider>,
  );

describe('UserSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.clearAllTimers();
  });

  it('should render with correct values', async () => {
    setup();
    expect(screen.getByText('Username')).toBeInTheDocument();
    expect(screen.getByText('Display Name')).toBeInTheDocument();
    expect(screen.getByText('Password')).toBeInTheDocument();
    expect(await screen.findByText(USERNAME)).toBeInTheDocument();
  });
  it('should be able to change display name', async () => {
    setup();
    await user.click(screen.getByTestId('edit-displayname'));
    await user.type(screen.getByPlaceholderText('Add display name'), 'a');
    await user.click(screen.getByTestId('submit-displayname'));
    expect(mockPatchUser).toHaveBeenCalledWith({
      userId: 1,
      userParams: { displayName: `${DISPLAY_NAME}a` },
    });
    await waitFor(() =>
      expect(screen.getByTestId('value-displayname')).toHaveTextContent(`${DISPLAY_NAME}a`),
    );
  });
});
