import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from 'antd';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { V1LoginRequest } from 'services/api-ts-sdk';
import { SetUserPasswordParams } from 'services/types';
import { DetailedUser } from 'types';

import useModalPasswordChange, {
  API_SUCCESS_MESSAGE,
  CANCEL_BUTTON_LABEL,
  CONFIRM_PASSWORD_LABEL,
  MODAL_HEADER_LABEL,
  NEW_PASSWORD_LABEL,
  OK_BUTTON_LABEL,
  OLD_PASSWORD_LABEL,
} from './useModalPasswordChange';

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
const FIRST_PASSWORD_VALUE = 'Password1';
const SECOND_PASSWORD_VALUE = 'Password2';

const CURRENT_USER: DetailedUser = {
  displayName: 'Test Name',
  id: USER_ID,
  isActive: true,
  isAdmin: false,
  username: USERNAME,
};

const USERS: Array<DetailedUser> = [CURRENT_USER];

const user = userEvent.setup();

const Container: React.FC = () => {
  const { contextHolder, modalOpen } = useModalPasswordChange();
  const storeDispatch = useStoreDispatch();

  const loadUsers = useCallback(() => {
    storeDispatch({ type: StoreAction.SetUsers, value: USERS });
    storeDispatch({ type: StoreAction.SetCurrentUser, value: CURRENT_USER });
  }, [storeDispatch]);

  useEffect(() => loadUsers(), [loadUsers]);

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

    expect(screen.getByLabelText(OLD_PASSWORD_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(NEW_PASSWORD_LABEL)).toBeInTheDocument();
    expect(screen.getByLabelText(CONFIRM_PASSWORD_LABEL)).toBeInTheDocument();
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

  it('should validate the password update request', async () => {
    await setup();

    await user.type(screen.getByLabelText(OLD_PASSWORD_LABEL), ',');
    await user.type(screen.getByLabelText(NEW_PASSWORD_LABEL), '.');
    await user.type(screen.getByLabelText(CONFIRM_PASSWORD_LABEL), '/');
    await user.click(screen.getByRole('button', { name: OK_BUTTON_LABEL }));

    await waitFor(() => {
      expect(screen.getAllByRole('alert')).toHaveLength(6);
    });
  });

  it('should submit a valid password update request', async () => {
    await setup();

    await user.type(screen.getByLabelText(OLD_PASSWORD_LABEL), FIRST_PASSWORD_VALUE);
    await user.type(screen.getByLabelText(NEW_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
    await user.type(screen.getByLabelText(CONFIRM_PASSWORD_LABEL), SECOND_PASSWORD_VALUE);
    await user.click(screen.getByRole('button', { name: OK_BUTTON_LABEL }));

    // Check for successful toast message.
    await waitFor(() => {
      expect(screen.getByText(API_SUCCESS_MESSAGE)).toBeInTheDocument();
    });

    // Check for the modal to be dismissed.
    await waitFor(() => {
      expect(screen.queryByRole('heading', { name: MODAL_HEADER_LABEL })).not.toBeInTheDocument();
    });

    // Check that the API method was called with the correct parameters.
    expect(mockSetUserPassword).toHaveBeenCalledWith({
      password: SECOND_PASSWORD_VALUE,
      userId: USER_ID,
    });
  });
});
