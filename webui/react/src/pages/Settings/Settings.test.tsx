import { render, screen } from '@testing-library/react';
import React from 'react';
import { HelmetProvider } from 'react-helmet-async';
import { Router } from 'react-router-dom';

import StoreProvider from 'contexts/Store';
import history from 'shared/routes/history';

import Settings, { TabType } from './Settings';

const setup = () => {
  return render(
    <StoreProvider>
      <HelmetProvider>
        <Router history={history}>
          <Settings />
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
