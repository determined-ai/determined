import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { Router } from 'react-router-dom';

import StoreProvider from 'contexts/Store';
import history from 'shared/routes/history';
import { Mode } from 'types';

import ThemeToggle, { ThemeOptions } from './ThemeToggle';

const ThemeToggleContainer: React.FC = () => (
  <StoreProvider>
    <Router history={history}>
      <ThemeToggle />
    </Router>
  </StoreProvider>
);

const setup = () => render(<ThemeToggleContainer />);

describe('ThemeToggle', () => {
  it('should have system mode as the default setting', async () => {
    await setup();
    const defaultOption = ThemeOptions[Mode.System];
    expect(await screen.findByText(defaultOption.displayName)).toBeInTheDocument();
  });

  it('should cycle through all the modes in the correct order', async () => {
    const optionCount = Object.keys(ThemeOptions).length;
    let option = ThemeOptions[Mode.System];

    await setup();

    for (let i = 0; i < optionCount; i++) {
      expect(await screen.findByText(option.displayName)).toBeInTheDocument();
      userEvent.click(screen.getByText(option.displayName));
      option = ThemeOptions[option.next];
    }
  });
});
