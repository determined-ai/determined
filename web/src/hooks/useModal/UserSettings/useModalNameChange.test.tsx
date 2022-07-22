import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { PatchUserParams } from 'services/types';
import { DetailedUser } from 'types';

import useModalNameChange, {
  API_SUCCESS_MESSAGE,
  CANCEL_BUTTON_LABEL,
  DISPLAY_NAME_LABEL,
  MODAL_HEADER_LABEL,
  NAME_TOO_LONG_MESSAGE,
  OK_BUTTON_LABEL,
} from './useModalNameChange';

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
const NEW_DISPLAY_NAME = 'New Display Name';

const CURRENT_USER: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: USER_ID,
  isActive: true,
  isAdmin: false,
  username: USERNAME,
};

const USERS: Array<DetailedUser> = [ CURRENT_USER ];

const user = userEvent.setup();

const Container: React.FC = () => {
  const { contextHolder, modalOpen } = useModalNameChange();
  const storeDispatch = useStoreDispatch();

  const loadUsers = useCallback(() => {
    storeDispatch({ type: StoreAction.SetUsers, value: USERS });
    storeDispatch({ type: StoreAction.SetCurrentUser, value: CURRENT_USER });
  }, [ storeDispatch ]);

  useEffect(() => loadUsers(), [ loadUsers ]);

  return (
    <div>
      <Button onClick={() => modalOpen()}>{OPEN_MODAL_TEXT}</Button>
      {contextHolder}
    </div>
  );
};

const setup = async () => {
  const view = render(
    <StoreProvider>
      <Container />
    </StoreProvider>,
  );

  await user.click(await view.findByText(OPEN_MODAL_TEXT));
  await view.findByRole('heading', { name: MODAL_HEADER_LABEL });

  return view;
};

describe('useModalNameChange', () => {
  it('should open modal with correct values', async () => {
    await setup();

    expect(screen.getByRole('textbox', { name: DISPLAY_NAME_LABEL })).toHaveValue(DISPLAY_NAME);
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

  it('should validate the display name update request', async () => {
    await setup();

    const input = await screen.findByRole('textbox', { name: DISPLAY_NAME_LABEL });
    await user.type(input, 'a'.repeat(81));
    await user.click(screen.getByRole('button', { name: OK_BUTTON_LABEL }));

    // Check for error alert message.
    expect(await screen.findByText(NAME_TOO_LONG_MESSAGE)).toBeInTheDocument();
  });

  it('should submit a valid display name update request', async () => {
    await setup();

    await user.clear(screen.getByRole('textbox', { name: DISPLAY_NAME_LABEL }));
    await user.click(screen.getByRole('textbox', { name: DISPLAY_NAME_LABEL }));
    await user.keyboard(NEW_DISPLAY_NAME);

    mockPatchUser.mockResolvedValue({
      ...CURRENT_USER,
      displayName: NEW_DISPLAY_NAME,
    });

    await user.click(await screen.findByRole('button', { name: OK_BUTTON_LABEL }));

    // Check for successful toast message.
    await waitFor(() => {
      expect(screen.getByText(API_SUCCESS_MESSAGE)).toBeInTheDocument();
    });

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER_LABEL })).not.toBeInTheDocument();
    });

    // Check that the API method was called with the correct parameters.
    expect(mockPatchUser).toHaveBeenCalledWith({
      userId: USER_ID,
      userParams: { displayName: NEW_DISPLAY_NAME },
    });
  });
});
