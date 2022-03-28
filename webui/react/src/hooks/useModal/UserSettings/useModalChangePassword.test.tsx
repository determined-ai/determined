import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button, Modal } from 'antd';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { V1LoginRequest } from 'services/api-ts-sdk';
import { SetUserPasswordParams } from 'services/types';
import { DetailedUser } from 'types';

import useModalUserSettings from './useModalUserSettings';

const mockSetUserPassword = jest.fn();

jest.mock('services/api', () => ({
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

const OPEN_MODAL_TEXT = 'Open Modal';
const USERNAME = 'test_username1';
const USER_ID = 1;
const CHANGE_PASSWORD_TEXT = 'Change password';
const USER_SETTINGS_HEADER = 'Account';
const FIRST_PASSWORD_VALUE = 'Password';
const SECOND_PASSWORD_VALUE = 'Password2';
const OLD_PASSWORD_LABEL = 'Old Password';
const NEW_PASSWORD_LABEL = 'New Password';
const CONFIRM_PASSWORD_LABEL = 'Confirm Password';

const currentUser: DetailedUser = {
  displayName: 'Test name',
  id: USER_ID,
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

  useEffect(() => {
    loadUsers();
  });

  return (
    <div>
      {contextHolder}
      <Button onClick={() => openUserSettingsModal()}>
        {OPEN_MODAL_TEXT}
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
  userEvent.click(await screen.findByText(CHANGE_PASSWORD_TEXT));
};

describe('useModalChangePassword', () => {
  it('opens modal with correct values', async () => {
    await setup();

    await screen.findByRole('heading', { name: CHANGE_PASSWORD_TEXT });
    expect(screen.getByLabelText(OLD_PASSWORD_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(NEW_PASSWORD_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(CONFIRM_PASSWORD_LABEL)).toBeInTheDocument();
  });

  it('validates the password update request', async () => {
    await setup();

    await screen.findByRole('heading', { name: CHANGE_PASSWORD_TEXT });
    userEvent.type(screen.getByLabelText(OLD_PASSWORD_LABEL), ',');
    userEvent.type(screen.getByLabelText(NEW_PASSWORD_LABEL), '.');
    userEvent.type(screen.getByLabelText(CONFIRM_PASSWORD_LABEL), '/');
    userEvent.click(screen.getAllByRole('button', { name: CHANGE_PASSWORD_TEXT })[1]);

    await waitFor(() => {
      expect(screen.getAllByRole('alert')).toHaveLength(6);
    });
  });

  it('submits a valid password update request', async () => {
    await setup();

    await screen.findByRole('heading', { name: CHANGE_PASSWORD_TEXT });
    userEvent.type(screen.getByLabelText(OLD_PASSWORD_LABEL), FIRST_PASSWORD_VALUE);
    userEvent.type(screen.getByLabelText(NEW_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
    userEvent.type(screen.getByLabelText(CONFIRM_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
    userEvent.click(screen.getAllByRole('button', { name: CHANGE_PASSWORD_TEXT })[1]);

    // TODO: test for toast message appearance?

    // modal closes:
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: CHANGE_PASSWORD_TEXT })).not.toBeInTheDocument();
    });
    expect(screen.getByRole('heading', { name: USER_SETTINGS_HEADER })).toBeInTheDocument();

    // api method was called:
    expect(mockSetUserPassword).toHaveBeenCalledWith({
      password: SECOND_PASSWORD_VALUE,
      userId: USER_ID,
    });
  });

  it('closes the modal and returns to User Settings modal', async () => {
    await setup();

    await waitFor(() => {
      userEvent.click(screen.getAllByRole('button', { name: /close/i })[1]);
    });

    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: CHANGE_PASSWORD_TEXT })).not.toBeInTheDocument();
    });
    expect(screen.getByRole('heading', { name: USER_SETTINGS_HEADER })).toBeInTheDocument();
  });
});
