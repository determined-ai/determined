import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React from 'react';

import StoreProvider from 'contexts/Store';
import { PostUserParams } from 'services/types';
import { UsersProvider } from 'stores/users';

import useModalCreateUser, {
  ADMIN_LABEL,
  API_SUCCESS_MESSAGE_CREATE,
  DISPLAY_NAME_LABEL,
  MODAL_HEADER_LABEL_CREATE,
  USER_NAME_LABEL,
} from './useModalCreateUser';

const mockCreateUser = jest.fn();

jest.mock('services/api', () => ({
  getUserRoles: () => Promise.resolve([]),
  postUser: (params: PostUserParams) => {
    mockCreateUser(params);
    return Promise.resolve({ user: { id: 1 } });
  },
}));

const OPEN_MODAL_TEXT = 'Open Modal';
const USERNAME = 'test_username1';

const user = userEvent.setup();

const Container: React.FC = () => {
  const { contextHolder, modalOpen } = useModalCreateUser({ groups: [] });

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
      <UsersProvider>
        <Container />
      </UsersProvider>
    </StoreProvider>,
  );

  await user.click(await view.findByText(OPEN_MODAL_TEXT));
  await view.findByRole('heading', { name: MODAL_HEADER_LABEL_CREATE });

  // Check for the modal to finish loading.
  await waitFor(() => {
    expect(screen.queryByText('Loading', { exact: false })).not.toBeInTheDocument();
  });

  return view;
};

describe('useModalCreateUser', () => {
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
      expect(
        screen.queryByRole('heading', { name: MODAL_HEADER_LABEL_CREATE }),
      ).not.toBeInTheDocument();
    });
  });

  it('should close the modal via cancel button', async () => {
    await setup();

    await user.click(await screen.findByText('Cancel'));

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(
        screen.queryByRole('heading', { name: MODAL_HEADER_LABEL_CREATE }),
      ).not.toBeInTheDocument();
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
      expect(
        screen.getByText(API_SUCCESS_MESSAGE_CREATE, { collapseWhitespace: false }),
      ).toBeInTheDocument();
    });

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(
        screen.queryByRole('heading', { name: MODAL_HEADER_LABEL_CREATE }),
      ).not.toBeInTheDocument();
    });

    // Check that the API method was called with the correct parameters.
    expect(mockCreateUser).toHaveBeenCalledWith({
      user: { active: true, username: USERNAME },
    });
  });
});
