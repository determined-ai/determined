import { render, screen } from '@testing-library/react';
import React, { useCallback, useEffect, useMemo } from 'react';
import { HelmetProvider } from 'react-helmet-async';
import { Router } from 'react-router-dom';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';
import history from 'shared/routes/history';
import { DetailedUser } from 'types';

import Settings, { TabType } from './Settings';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const Container: React.FC = () => {
  const storeDispatch = useStoreDispatch();

  const currentUser: DetailedUser = useMemo(
    () => ({
      displayName: DISPLAY_NAME,
      id: 1,
      isActive: true,
      isAdmin: true,
      username: USERNAME,
    }),
    [],
  );

  const loadUser = useCallback(() => {
    storeDispatch({ type: StoreAction.SetCurrentUser, value: currentUser });
  }, [storeDispatch, currentUser]);

  useEffect(() => loadUser(), [loadUser]);

  return <Settings />;
};

const setup = () => {
  return render(
    <StoreProvider>
      <HelmetProvider>
        <Router history={history}>
          <Container />
        </Router>
      </HelmetProvider>
    </StoreProvider>,
  );
};

describe('Settings Page', () => {
  it('should render the page properly', () => {
    setup();
    expect(screen.getByText('Username')).toBeInTheDocument();
    expect(screen.getByText('Display Name')).toBeInTheDocument();
    expect(screen.getByText('Password')).toBeInTheDocument();
  });

  it('should enable user management for admins only', () => {
    // TODO: Change out to set user to be admin and check for user management tab.
    history.push('/?f_rbac=on');
    setup();

    expect(screen.queryByRole('tab', { name: TabType.UserManagement })).toBeInTheDocument();
  });
});
