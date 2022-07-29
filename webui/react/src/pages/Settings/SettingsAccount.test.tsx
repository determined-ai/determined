import { render, screen } from '@testing-library/react';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
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

const users: Array<DetailedUser> = [ currentUser ];

const Container: React.FC = () => {
  const storeDispatch = useStoreDispatch();

  const loadUsers = useCallback(() => {
    storeDispatch({ type: StoreAction.SetUsers, value: users });
    storeDispatch({ type: StoreAction.SetCurrentUser, value: currentUser });
  }, [ storeDispatch ]);

  useEffect(() => loadUsers(), [ loadUsers ]);

  return <SettingsAccount />;
};

const setup = () => render(
  <StoreProvider>
    <Container />
  </StoreProvider>,
);

describe('SettingsAccount', () => {
  it('should render with correct values', () => {
    const { container } = setup();

    expect(screen.getByText(USERNAME)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_PASSWORD_TEXT)).toBeInTheDocument();

    // Fetching element by specific attribute is not natively supported.
    const editor = container.querySelector(`[data-value="${DISPLAY_NAME}"]`);
    expect(editor).toBeInTheDocument();
  });
});
