import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
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

const Container: React.FC = () => {
  const { contextHolder, modalOpen: openUserSettingsModal } = useModalUserSettings();
  const storeDispatch = useStoreDispatch();

  const loadUsers = useCallback(() => {
    storeDispatch({ type: StoreAction.SetUsers, value: users });
    storeDispatch({ type: StoreAction.SetCurrentUser, value: currentUser });
  }, [ storeDispatch ]);

  useEffect(() => loadUsers(), [ loadUsers ]);

  return (
    <>
      <Button onClick={() => openUserSettingsModal()}>
        {OPEN_MODAL_TEXT}
      </Button>
      {contextHolder}
    </>
  );
};

const setup = async () => {
  const user = userEvent.setup();

  render(
    <StoreProvider>
      <Container />
    </StoreProvider>,
  );

  await user.click(screen.getByText(OPEN_MODAL_TEXT));

  return user;
};

describe('useModalUserSettings', () => {
  it('should open modal with correct values', async () => {
    await setup();

    await screen.findByRole('heading', { name: USER_SETTINGS_HEADER });
    expect(screen.getByText(DISPLAY_NAME)).toBeInTheDocument();
    expect(screen.getByText(USERNAME)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_NAME_TEXT)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_PASSWORD_TEXT)).toBeInTheDocument();
  });

  it('should close the modal', async () => {
    const user = await setup();

    await waitFor(async () => {
      await user.click(screen.getByRole('button', { name: /close/i }));
    });

    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: USER_SETTINGS_HEADER })).not.toBeInTheDocument();
    });
  });
});
