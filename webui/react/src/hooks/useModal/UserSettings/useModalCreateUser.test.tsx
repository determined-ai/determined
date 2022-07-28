import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { PostUserParams } from 'services/types';
import { DetailedUser } from 'types';

import useModalCreateUser, { ADMIN_LABEL, API_SUCCESS_MESSAGE, DISPLAY_NAME_LABEL,
  MODAL_HEADER_LABEL, USER_NAME_LABEL } from './useModalCreateUser';

const mockCreateUser = jest.fn();

jest.mock('services/api', () => ({
  postUser: (params: PostUserParams) => {
    return mockCreateUser(params);
  },
}));

const OPEN_MODAL_TEXT = 'Open Modal';
const USERNAME = 'test_username1';
const USER_ID = 1;

const CURRENT_USER: DetailedUser = {
  displayName: 'Test Name',
  id: USER_ID,
  isActive: true,
  isAdmin: false,
  username: USERNAME,
};

const USERS: Array<DetailedUser> = [ CURRENT_USER ];

const user = userEvent.setup();

const Container: React.FC = () => {
  const { contextHolder, modalOpen } = useModalCreateUser({});
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

describe('useModalPasswordChange', () => {
  it('should open modal with correct values', async () => {
    await setup();

    expect(screen.getByLabelText(USER_NAME_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(DISPLAY_NAME_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(ADMIN_LABEL)).toBeInTheDocument();
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

    await user.click(await screen.findByText('Cancel'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER_LABEL })).not.toBeInTheDocument();
    });
  });

  it('should validate the create user request', async () => {
    await setup();

    await user.click(screen.getByRole('button', { name: 'Create User' }));

    await waitFor(() => {
      expect(screen.getAllByRole('alert')).toHaveLength(1);
    });
  });

  it('should submit a valid create user request', async () => {
    await setup();

    await user.type(screen.getByLabelText(USER_NAME_LABEL), USERNAME);
    await user.click(screen.getByRole('button', { name: 'Create User' }));

    // Check for successful toast message.
    await waitFor(() => {
      expect(screen.getByText(API_SUCCESS_MESSAGE)).toBeInTheDocument();
    });

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER_LABEL })).not.toBeInTheDocument();
    });

    // Check that the API method was called with the correct parameters.
    expect(mockCreateUser).toHaveBeenCalledWith({
      admin: false,
      displayName: undefined,
      username: USERNAME,
    });
  });
});
