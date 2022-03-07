import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button, Modal } from 'antd';
import React, { useCallback } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { V1LoginRequest } from 'services/api-ts-sdk';
import { SetUserPasswordParams } from 'services/types';
import { DetailedUser } from 'types';

import useModalUserSettings from './useModalUserSettings';

const OPEN_MODAL_TEXT = 'Open Modal';
const LOAD_USERS_TEXT = 'Load Users';
const USERNAME = 'test_username1';
const CHANGE_PASSWORD_TEXT = 'Change password';
const USER_SETTINGS_HEADER = 'Account';
const FIRST_PASSWORD_VALUE = 'Password';
const SECOND_PASSWORD_VALUE = 'Password2';
const OLD_PASSWORD_LABEL = 'Old Password';
const NEW_PASSWORD_LABEL = 'New Password';
const CONFIRM_PASSWORD_LABEL = 'Confirm Password';

const currentUser: DetailedUser = {
  displayName: 'Test name',
  id: 1,
  isActive: true,
  isAdmin: false,
  username: USERNAME,
};

const users: Array<DetailedUser> = [ currentUser ];

const TestApp: React.FC = () => {
  const [ modal, contextHolder ] = Modal.useModal();
  const { modalOpen: openUserSettingsModal } = useModalUserSettings(modal);
  const storeDispatch = useStoreDispatch();

  const loadUsers = useCallback(() => {
    storeDispatch({
      type: StoreAction.SetUsers,
      value: users,
    });
    storeDispatch({
      type: StoreAction.SetCurrentUser,
      value: currentUser,
    });
  }, [ storeDispatch ]);

  return (
    <div>
      {contextHolder}
      <Button onClick={() => openUserSettingsModal()}>
        {OPEN_MODAL_TEXT}
      </Button>
      <Button onClick={() => loadUsers()}>
        {LOAD_USERS_TEXT}
      </Button>
    </div>
  );
};

const setup = async () => {
  render(
    <StoreProvider>
      <TestApp />
    </StoreProvider>,
  );
  userEvent.click(await screen.findByText(OPEN_MODAL_TEXT));
  userEvent.click(screen.getByText(LOAD_USERS_TEXT));
  userEvent.click(screen.getByText(CHANGE_PASSWORD_TEXT));
};

const mockSetUserPassword = jest.fn((params) => {
  return Promise.resolve(params);
});

jest.mock('services/api', () => {
  return {
    login: ({ password, username }: V1LoginRequest) => {
      if (password === FIRST_PASSWORD_VALUE && username === USERNAME) {
        return Promise.resolve();
      } else {
        return Promise.reject();
      }
    },
    setUserPassword: (params: SetUserPasswordParams) => {
      mockSetUserPassword(params);
    },
  };
});

describe('useModalChangePassword', () => {
  it('opens modal with correct values', async () => {
    await setup();

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: CHANGE_PASSWORD_TEXT })).toBeInTheDocument();
      expect(screen.getByLabelText(OLD_PASSWORD_LABEL)).toBeInTheDocument();
      expect(screen.getByLabelText(NEW_PASSWORD_LABEL)).toBeInTheDocument();
      expect(screen.getByLabelText(CONFIRM_PASSWORD_LABEL)).toBeInTheDocument();
    });
  });

  it('validates the password update request', async () => {
    await setup();

    await waitFor(() => {
      userEvent.type(screen.getByLabelText(OLD_PASSWORD_LABEL), ',');
      userEvent.type(screen.getByLabelText(NEW_PASSWORD_LABEL), '.');
      userEvent.type(screen.getByLabelText(CONFIRM_PASSWORD_LABEL), '/');
      userEvent.click(screen.getAllByRole('button', { name: CHANGE_PASSWORD_TEXT })[1]);
    });

    await waitFor(() => {
      expect(screen.getAllByRole('alert')).toHaveLength(6);
    });
  });

  it('submits a valid password update request', async () => {
    await setup();

    await waitFor(() => {
      userEvent.type(screen.getByLabelText(OLD_PASSWORD_LABEL), FIRST_PASSWORD_VALUE);
      userEvent.type(screen.getByLabelText(NEW_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
      userEvent.type(screen.getByLabelText(CONFIRM_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
      userEvent.click(screen.getAllByRole('button', { name: CHANGE_PASSWORD_TEXT })[1]);
    });

    // TODO: test for toast message appearance?

    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: CHANGE_PASSWORD_TEXT })).not.toBeInTheDocument();
      expect(screen.getByRole('heading', { name: USER_SETTINGS_HEADER })).toBeInTheDocument();
      expect(mockSetUserPassword).toHaveBeenCalledWith({
        password: SECOND_PASSWORD_VALUE,
        username: USERNAME,
      });
    });
  });

  it('closes the modal and returns to User Settings modal', async () => {
    await setup();

    await waitFor(() => {
      userEvent.click(screen.getAllByRole('button', { name: /cancel/i })[1]);
    });

    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: CHANGE_PASSWORD_TEXT })).not.toBeInTheDocument();
      expect(screen.getByRole('heading', { name: USER_SETTINGS_HEADER })).toBeInTheDocument();
    });
  });
});
