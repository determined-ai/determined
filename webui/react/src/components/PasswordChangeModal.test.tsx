import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useCallback, useEffect } from 'react';

import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import { StoreProvider as UIProvider } from 'components/kit/Theme';
import { setUserPassword as mockSetUserPassword } from 'services/api';
import { V1LoginRequest } from 'services/api-ts-sdk';
import authStore from 'stores/auth';
import userStore from 'stores/users';
import { DetailedUser } from 'types';

vi.useFakeTimers();

import PasswordChangeModalComponent, {
  API_SUCCESS_MESSAGE,
  CONFIRM_PASSWORD_LABEL,
  OK_BUTTON_LABEL,
  OLD_PASSWORD_LABEL,
} from './PasswordChangeModal';

const OPEN_MODAL_TEXT = 'Open Modal';
const USERNAME = 'test_username1';
const USER_ID = 1;
const FIRST_PASSWORD_VALUE = 'Password1';
const SECOND_PASSWORD_VALUE = 'Password2';
const CURRENT_USER: DetailedUser = {
  displayName: 'Test Name',
  id: USER_ID,
  isActive: true,
  isAdmin: false,
  username: USERNAME,
};

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
  login: ({ password, username }: V1LoginRequest) => {
    if (password === FIRST_PASSWORD_VALUE && username === USERNAME) {
      return Promise.resolve();
    } else {
      return Promise.reject();
    }
  },
  setUserPassword: vi.fn(),
}));

const user = userEvent.setup({ delay: null });

const Container: React.FC = () => {
  const PasswordChangeModal = useModal(PasswordChangeModalComponent);

  const loadUsers = useCallback(async () => {
    await userStore.fetchUsers();
    authStore.setAuth({ isAuthenticated: true });
    userStore.updateCurrentUser(CURRENT_USER);
  }, []);

  useEffect(() => {
    loadUsers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div>
      <Button onClick={PasswordChangeModal.open}>{OPEN_MODAL_TEXT}</Button>
      <PasswordChangeModal.Component newPassword={SECOND_PASSWORD_VALUE} />
    </div>
  );
};

const setup = async () => {
  const view = render(
    <UIProvider>
      <Container />
    </UIProvider>,
  );

  await user.click(await view.findByText(OPEN_MODAL_TEXT));

  return view;
};

describe('Password Change Modal', () => {
  it('should submit a valid password update request', async () => {
    await setup();

    await user.type(screen.getByLabelText(OLD_PASSWORD_LABEL), FIRST_PASSWORD_VALUE);
    await user.type(screen.getByLabelText(CONFIRM_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
    await user.click(screen.getByRole('button', { name: OK_BUTTON_LABEL }));

    vi.advanceTimersToNextTimer();

    // Check for successful toast message.
    await waitFor(() => {
      expect(screen.getByText(API_SUCCESS_MESSAGE)).toBeInTheDocument();
    });

    // Check that the API method was called with the correct parameters.
    expect(mockSetUserPassword).toHaveBeenCalledWith({
      password: SECOND_PASSWORD_VALUE,
      userId: USER_ID,
    });
  });
} /* { timeout: 10000 } */);
