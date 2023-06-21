import { render, screen } from '@testing-library/react';
import React, { useCallback, useEffect } from 'react';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { StoreProvider as UIProvider } from 'stores/contexts/UI';
import userStore from 'stores/users';
import { DetailedUser } from 'types';

import Settings from './Settings';

const DISPLAY_NAME = 'Test Name';
const USERNAME = 'test_username1';
const CURRENT_USER: DetailedUser = {
  displayName: DISPLAY_NAME,
  id: 1,
  isActive: true,
  isAdmin: true,
  username: USERNAME,
};

const Container: React.FC = () => {
  const loadUser = useCallback(() => {
    userStore.updateCurrentUser(CURRENT_USER);
  }, []);

  useEffect(() => loadUser(), [loadUser]);

  return <Settings />;
};

const setup = () => {
  return render(
    <UIProvider>
      <HelmetProvider>
        <BrowserRouter>
          <Container />
        </BrowserRouter>
      </HelmetProvider>
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
