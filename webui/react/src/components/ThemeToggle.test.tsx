import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

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
const LIGHT_MODE = 'Light Mode'
const DARK_MODE = 'Dark Mode'
const setup = () => render(<ThemeToggleContainer />);

describe('ThemeToggle', () => {
  it('System Mode is the default setting', async () => {
    await setup();
    expect(await screen.findByText(SYSTEM_MODE)).toBeInTheDocument();
  });

  it('Light Mode is activated after System Mode', async () => {
    await setup();
    userEvent.click(screen.getByText(SYSTEM_MODE))
    expect(await screen.findByText(LIGHT_MODE)).toBeInTheDocument();
  });

  it('Dark Mode is activated after Light Mode', async () => {
    await setup();
    userEvent.click(screen.getByText(LIGHT_MODE))
    expect(await screen.findByText(DARK_MODE)).toBeInTheDocument();
  });

  it('System Mode is activated after Dark Mode', async () => {
    await setup();
    userEvent.click(screen.getByText(DARK_MODE))
    expect(await screen.findByText(SYSTEM_MODE)).toBeInTheDocument();
  });

});
