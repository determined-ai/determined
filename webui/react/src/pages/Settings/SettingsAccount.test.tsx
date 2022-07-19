import { render, screen } from '@testing-library/react';
import React, { useCallback, useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import { DetailedUser } from 'types';

import SettingsAccount from './SettingsAccount';

const USERNAME = 'test_username1';
const DISPLAY_NAME = 'Test Name';
const CHANGE_NAME_TEXT = 'Change name';
const CHANGE_PASSWORD_TEXT = 'Change password';

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
    setup();

    expect(screen.getByText(DISPLAY_NAME)).toBeInTheDocument();
    expect(screen.getByText(USERNAME)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_NAME_TEXT)).toBeInTheDocument();
    expect(screen.getByText(CHANGE_PASSWORD_TEXT)).toBeInTheDocument();
  });
});
