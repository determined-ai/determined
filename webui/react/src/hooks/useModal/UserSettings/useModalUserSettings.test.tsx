import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button, Modal } from 'antd';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { DetailedUser } from 'types';

import useModalUserSettings from './useModalUserSettings';

const OPEN_MODAL_TEXT = 'Open Modal';
const USERNAME = 'test_username1';
const DISPLAY_NAME = 'Test Name';
const CHANGE_NAME_TEXT = 'Change name';
const CHANGE_PASSWORD_TEXT = 'Change password';
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
};

describe('useModalUserSettings', () => {
  it('opens modal with correct values', async () => {
    await setup();

    await screen.findByRole('heading', { name: USER_SETTINGS_HEADER });
    expect(screen.getByText(DISPLAY_NAME)).toBeInTheDocument();
    expect(screen.getByText(USERNAME)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_NAME_TEXT)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_PASSWORD_TEXT)).toBeInTheDocument();
  });

  it('closes the modal', async () => {
    await setup();

    await waitFor(() => {
      userEvent.click(screen.getByRole('button', { name: /close/i }));
    });

    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: USER_SETTINGS_HEADER })).not.toBeInTheDocument();
    });
  });
});
