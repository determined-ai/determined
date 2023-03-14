import { waitFor } from '@testing-library/dom';
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useCallback, useEffect, useState } from 'react';

import { NEW_PASSWORD_LABEL } from 'hooks/useModal/UserSettings/useModalPasswordChange';
import { PatchUserParams } from 'services/types';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { setAuth } from 'stores/auth';
import usersStore from 'stores/users';
import { DetailedUser } from 'types';

import SettingsAccount, { CHANGE_PASSWORD_TEXT } from './SettingsAccount';

const mockPatchUser = jest.fn();

jest.mock('services/api', () => ({
  ...jest.requireActual('services/api'),
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
  patchUser: (params: PatchUserParams) => {
    mockPatchUser(params);
    return Promise.resolve({
      displayName: params.userParams.displayName,
      id: 1,
      isActive: true,
      isAdmin: false,
      username: params.userParams.username,
    });
  },
}));

const user = userEvent.setup();

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const currentUser: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: 1,
  isActive: true,
  isAdmin: false,
  username: USERNAME,
};

const Container: React.FC = () => {
  const [canceler] = useState(new AbortController());

  const loadUsers = useCallback(() => {
    usersStore.updateCurrentUser(currentUser.id);
  }, []);

  useEffect(() => {
    usersStore.ensureUsersFetched(canceler);
    setAuth({ isAuthenticated: true });
  }, [canceler]);

  useEffect(() => {
    loadUsers();
  }, [loadUsers]);

  return <SettingsAccount />;
};

const setup = () =>
  render(
    <UIProvider>
      <Container />
    </UIProvider>,
  );

describe('SettingsAccount', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.clearAllTimers();
  });

  it('should render with correct values', async () => {
    setup();
    expect(await screen.findByText(USERNAME)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_PASSWORD_TEXT)).toBeInTheDocument();
  });
  it('should be able to change display name', async () => {
    act(() => {
      setup();
    });
    await user.click(screen.getByTestId('edit-displayname'));
    await user.type(screen.getByPlaceholderText('Add display name'), 'a');
    await user.keyboard('{enter}');
    expect(mockPatchUser).toHaveBeenCalledWith({
      userId: 1,
      userParams: { displayName: `${DISPLAY_NAME}a` },
    });
  });
  it('should be able to view change password modal when click', async () => {
    await waitFor(() => setup());
    await user.click(screen.getByText(CHANGE_PASSWORD_TEXT));
    expect(screen.getByText(NEW_PASSWORD_LABEL)).toBeInTheDocument();
  });
});
