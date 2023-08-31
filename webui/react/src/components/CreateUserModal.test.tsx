import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { BrowserRouter } from 'react-router-dom';

import CreateUserModalComponent, {
  API_SUCCESS_MESSAGE_CREATE,
  BUTTON_NAME,
  MODAL_HEADER_LABEL_CREATE,
  USER_NAME_LABEL,
} from 'components/CreateUserModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import { postUser as mockCreateUser } from 'services/api';

vi.mock('services/api', () => ({
  getUserRoles: () => Promise.resolve([]),
  postUser: vi.fn().mockReturnValue({ user: { id: 1 } }),
}));

const OPEN_MODAL_TEXT = 'Open Modal';
const USERNAME = 'test_username1';

const user = userEvent.setup();

const Container: React.FC = () => {
  const CreateUserModal = useModal(CreateUserModalComponent);

  return (
    <div>
      <Button onClick={CreateUserModal.open}>{OPEN_MODAL_TEXT}</Button>
      <CreateUserModal.Component />
    </div>
  );
};

const setup = async () => {
  const view = render(
    <BrowserRouter>
      <Container />
    </BrowserRouter>,
  );

  await user.click(await view.findByText(OPEN_MODAL_TEXT));
  await view.findByText(MODAL_HEADER_LABEL_CREATE);

  // Check for the modal to finish loading.
  await waitFor(() => {
    expect(screen.queryByText('Loading', { exact: false })).not.toBeInTheDocument();
  });

  return view;
};

describe('Create User Modal', () => {
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

    // Check that the API method was called with the correct parameters.
    expect(mockCreateUser).toHaveBeenCalledWith({
      user: { active: true, username: USERNAME },
    });
  });
});
