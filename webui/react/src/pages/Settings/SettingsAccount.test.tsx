import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { NEW_PASSWORD_LABEL } from 'hooks/useModal/UserSettings/useModalPasswordChange';
import { PatchUserParams } from 'services/types';
import { DetailedUser } from 'types';

import SettingsAccount, { CHANGE_PASSWORD_TEXT } from './SettingsAccount';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const currentUser: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: 1,
  isActive: true,
  isAdmin: false,
  username: USERNAME,
};

const mockPatchUser = jest.fn();

jest.mock('services/api', () => ({
  patchUser: (params: PatchUserParams) => {
    mockPatchUser(params);
    return Promise.resolve({
      displayName: params.userParams.displayName,
      id: 1,
      isActive: true,
      isAdmin: false,
      username: params.userParams.username,
    });
  },
}));

const users: Array<DetailedUser> = [currentUser];
const user = userEvent.setup();

const Container: React.FC = () => {
  const storeDispatch = useStoreDispatch();

  const loadUsers = useCallback(() => {
    storeDispatch({ type: StoreAction.SetUsers, value: users });
    storeDispatch({ type: StoreAction.SetCurrentUser, value: currentUser });
  }, [storeDispatch]);

  useEffect(() => loadUsers(), [loadUsers]);

  return <SettingsAccount />;
};

const setup = () =>
  render(
    <StoreProvider>
      <Container />
    </StoreProvider>,
  );

describe('SettingsAccount', () => {
  it('should render with correct values', () => {
    const { container } = setup();

    expect(screen.getByDisplayValue(USERNAME)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_PASSWORD_TEXT)).toBeInTheDocument();

    // Fetching element by specific attribute is not natively supported.
    const editor = container.querySelector(`[data-value="${DISPLAY_NAME}"]`);
    expect(editor).toBeInTheDocument();
  });
  it('should be able to change display name', async () => {
    setup();
    await user.type(screen.getByPlaceholderText('Add display name'), 'a');
    await user.keyboard('{enter}');
    expect(mockPatchUser).toHaveBeenCalledWith({
      userId: 1,
      userParams: { displayName: `${DISPLAY_NAME}a` },
    });
  });
  it('should be able to view change password modal when click', async () => {
    setup();
    await user.click(screen.getByText(CHANGE_PASSWORD_TEXT));
    expect(screen.getByText(NEW_PASSWORD_LABEL)).toBeInTheDocument();
  });
});
