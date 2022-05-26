import { render, screen } from '@testing-library/react';
import React from 'react';
import { Router } from 'react-router-dom';

import StoreProvider from 'contexts/Store';
import history from 'shared/routes/history';

import ThemeToggle from './ThemeToggle';

const ThemeToggleContainer: React.FC = () => {

  return (
    <StoreProvider>
      <Router history={history}>
        <ThemeToggle />
      </Router>
    </StoreProvider>
  );
};

const SYSTEM_MODE = 'System Mode';
const setup = () => render(<ThemeToggleContainer />);

describe('ThemeToggle', () => {
  it('System Mode is the default', async () => {
    await setup();
    expect(await screen.findByText(SYSTEM_MODE)).toBeInTheDocument();
  });

});
