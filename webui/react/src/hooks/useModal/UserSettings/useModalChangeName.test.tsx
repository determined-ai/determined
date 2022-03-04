import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button, Modal } from 'antd';
import React, { useCallback } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { DetailedUser } from 'types';

import useModalUserSettings from './useModalUserSettings';

const OPEN_MODAL_TEXT = 'Open Modal';
const LOAD_USERS_TEXT = 'Load Users';
const USERNAME = 'test_username1';
const DISPLAY_NAME = 'Test Name';
const CHANGE_NAME_TEXT = 'Change name';
const USER_SETTINGS_HEADER = 'Account';

const currentUser: DetailedUser = {
  displayName: DISPLAY_NAME,
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
  userEvent.click(await screen.findByText(LOAD_USERS_TEXT));
};

describe('useModalChangeName', () => {
  it('opens modal with correct values', async () => {
    await setup();
    userEvent.click(screen.getByText(CHANGE_NAME_TEXT));

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: CHANGE_NAME_TEXT })).toBeInTheDocument();
      expect(screen.getByRole('textbox', { name: 'Display name' })).toHaveValue(DISPLAY_NAME);
    });
  });

  it('closes the modal and returns to User Settings modal', async () => {
    await setup();
    userEvent.click(screen.getByText(CHANGE_NAME_TEXT));

    await waitFor(() => {
      userEvent.click(screen.getAllByRole('button', { name: /cancel/i })[1]);
    });

    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: CHANGE_NAME_TEXT })).not.toBeInTheDocument();
      expect(screen.queryByRole('heading', { name: USER_SETTINGS_HEADER })).toBeInTheDocument();
    });
  });
});
