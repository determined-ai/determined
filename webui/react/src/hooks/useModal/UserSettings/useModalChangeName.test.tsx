import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button, Modal } from 'antd';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { PatchUserParams } from 'services/types';
import { DetailedUser } from 'types';

import useModalUserSettings from './useModalUserSettings';

const mockPatchUser = jest.fn();

jest.mock('services/api', () => ({
  patchUser: (params: PatchUserParams) => {
    return mockPatchUser(params);
  },
}));

const OPEN_MODAL_TEXT = 'Open Modal';
const USERNAME = 'test_username1';
const USER_ID = 1;
const DISPLAY_NAME = 'Test Name';
const CHANGE_NAME_TEXT = 'Change name';
const USER_SETTINGS_HEADER = 'Account';
const UPDATED_DISPLAY_NAME = 'New Displayname';

const currentUser: DetailedUser = {
  displayName: DISPLAY_NAME,
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
  const view = render(
    <StoreProvider>
      <TestApp />
    </StoreProvider>,
  );
  const user = userEvent.setup();
  await user.click(await screen.findByText(OPEN_MODAL_TEXT));
  await user.click(await screen.findByText(CHANGE_NAME_TEXT));

  return { user, view };
};

describe('useModalChangeName', () => {
  it('opens modal with correct values', async () => {
    await setup();

    await screen.findByRole('heading', { name: CHANGE_NAME_TEXT });
    expect(screen.getByRole('textbox', { name: 'Display name' })).toHaveValue(DISPLAY_NAME);
  });

  it('validates the display name update request', async () => {
    const { user } = await setup();

    const input = await screen.findByRole('textbox', { name: 'Display name' });
    await user.type(input, 'a'.repeat(81));
    await user.click(screen.getAllByRole('button', { name: CHANGE_NAME_TEXT })[1]);

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument();
    });
  });

  it('submits a valid display name update request', async () => {
    const { user } = await setup();

    await screen.findByRole('heading', { name: CHANGE_NAME_TEXT });
    await user.clear(screen.getByRole('textbox', { name: 'Display name' }));
    await user.click(screen.getByRole('textbox', { name: 'Display name' }));
    await user.keyboard(UPDATED_DISPLAY_NAME);

    mockPatchUser.mockResolvedValue({
      ...currentUser,
      displayName: UPDATED_DISPLAY_NAME,
    });

    await user.click(screen.getAllByRole('button', { name: CHANGE_NAME_TEXT })[1]);

    // TODO: test for toast message appearance?

    // modal closes:
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: CHANGE_NAME_TEXT })).not.toBeInTheDocument();
    });
    expect(screen.getByRole('heading', { name: USER_SETTINGS_HEADER })).toBeInTheDocument();

    // api method was called:
    expect(mockPatchUser).toHaveBeenCalledWith(
      {
        userId: USER_ID,
        userParams: { displayName: UPDATED_DISPLAY_NAME },
      },
    );

    // store was updated:
    expect(screen.queryByText(DISPLAY_NAME)).not.toBeInTheDocument();
    expect(screen.getByText(UPDATED_DISPLAY_NAME)).toBeInTheDocument();
  });

  it('closes the modal and returns to User Settings modal', async () => {
    const { user } = await setup();

    await waitFor(async () => {
      await user.click(screen.getAllByRole('button', { name: /close/i })[1]);
    });

    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: CHANGE_NAME_TEXT })).not.toBeInTheDocument();
    });
    expect(screen.getByRole('heading', { name: USER_SETTINGS_HEADER })).toBeInTheDocument();
  });
});
