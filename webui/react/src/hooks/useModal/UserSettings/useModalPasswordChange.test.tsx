import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useCallback, useEffect, useState } from 'react';

import Button from 'components/kit/Button';
import { V1LoginRequest } from 'services/api-ts-sdk';
import { SetUserPasswordParams } from 'services/types';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import { setAuth } from 'stores/auth';
import usersStore from 'stores/users';
import { DetailedUser } from 'types';

import useModalPasswordChange, {
  API_SUCCESS_MESSAGE,
  CANCEL_BUTTON_LABEL,
  CONFIRM_PASSWORD_LABEL,
  MODAL_HEADER_LABEL,
  NEW_PASSWORD_LABEL,
  OK_BUTTON_LABEL,
  OLD_PASSWORD_LABEL,
} from './useModalPasswordChange';

const mockSetUserPassword = jest.fn();

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

jest.setTimeout(10000);

jest.mock('services/api', () => ({
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
  setUserPassword: (params: SetUserPasswordParams) => {
    return mockSetUserPassword(params);
  },
}));

const user = userEvent.setup();

const Container: React.FC = () => {
  const { contextHolder, modalOpen } = useModalPasswordChange();
  const [canceler] = useState(new AbortController());

  const loadUsers = useCallback(async () => {
    await usersStore.ensureUsersFetched(canceler);
    setAuth({ isAuthenticated: true });
    usersStore.updateCurrentUser(CURRENT_USER.id);
  }, [canceler]);

  useEffect(() => {
    loadUsers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div>
      <Button onClick={() => modalOpen()}>{OPEN_MODAL_TEXT}</Button>
      {contextHolder}
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
  await view.findByRole('heading', { name: MODAL_HEADER_LABEL });

  return view;
};

describe('useModalPasswordChange', () => {
  it('should open modal with correct values', async () => {
    await setup();

    expect(screen.getByLabelText(OLD_PASSWORD_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(NEW_PASSWORD_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(CONFIRM_PASSWORD_LABEL)).toBeInTheDocument();
  });

  it('should close the modal via upper right close button', async () => {
    await setup();

    await user.click(await screen.findByLabelText('Close'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER_LABEL })).not.toBeInTheDocument();
    });
  });

  it('should close the modal via cancel button', async () => {
    await setup();

    await user.click(await screen.findByText(CANCEL_BUTTON_LABEL));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER_LABEL })).not.toBeInTheDocument();
    });
  });

  it('should submit a valid password update request', async () => {
    await setup();

    await user.type(screen.getByLabelText(OLD_PASSWORD_LABEL), FIRST_PASSWORD_VALUE);
    await user.type(screen.getByLabelText(NEW_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
    await user.type(screen.getByLabelText(CONFIRM_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
    await user.click(screen.getByRole('button', { name: OK_BUTTON_LABEL }));

    jest.advanceTimersToNextTimer();

    // Check for successful toast message.
    await waitFor(() => {
      expect(screen.getByText(API_SUCCESS_MESSAGE)).toBeInTheDocument();
    });

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER_LABEL })).not.toBeInTheDocument();
    });

    // Check that the API method was called with the correct parameters.
    expect(mockSetUserPassword).toHaveBeenCalledWith({
      password: SECOND_PASSWORD_VALUE,
      userId: USER_ID,
    });
  });
});
