import { render, screen } from '@testing-library/react';
import React, { useCallback, useEffect, useMemo } from 'react';
import { HelmetProvider } from 'react-helmet-async';
import { unstable_HistoryRouter as HistoryRouter } from 'react-router-dom';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import history from 'shared/routes/history';
import { AuthProvider } from 'stores/auth';
import { UsersProvider, useUpdateCurrentUser } from 'stores/users';
import { DetailedUser } from 'types';

import Settings from './Settings';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';

const Container: React.FC = () => {
  const updateCurrentUser = useUpdateCurrentUser();

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
    updateCurrentUser(currentUser.id);
  }, [updateCurrentUser, currentUser]);

  useEffect(() => loadUser(), [loadUser]);

  return <Settings />;
};

const setup = () => {
  return render(
    <UIProvider>
      <UsersProvider>
        <AuthProvider>
          <HelmetProvider>
            <HistoryRouter history={history}>
              <Container />
            </HistoryRouter>
          </HelmetProvider>
        </AuthProvider>
      </UsersProvider>
    </UIProvider>,
  );
};

describe('Settings Page', () => {
  it('should render the page properly', () => {
    setup();
    expect(screen.getByText('Username')).toBeInTheDocument();
    expect(screen.getByText('Display Name')).toBeInTheDocument();
    expect(screen.getByText('Password')).toBeInTheDocument();
  });
});
