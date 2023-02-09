import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import Button from 'components/kit/Button';
import { PostUserParams } from 'services/types';
import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import history from 'shared/routes/history';
import { UsersProvider } from 'stores/users';

import useModalCreateUser, {
  ADMIN_LABEL,
  API_SUCCESS_MESSAGE_CREATE,
  BUTTON_NAME,
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
  const { contextHolder, modalOpen } = useModalCreateUser({});

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
      <UsersProvider>
        <HistoryRouter history={history}>
          <Container />
        </HistoryRouter>
      </UsersProvider>
    </UIProvider>,
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

  it('should submit a valid create user request', async () => {
    await setup();

    await user.type(screen.getByLabelText(USER_NAME_LABEL), USERNAME);
    await user.click(screen.getByRole('button', { name: BUTTON_NAME }));

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
