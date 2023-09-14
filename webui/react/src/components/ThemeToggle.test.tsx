import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React from 'react';
import { BrowserRouter } from 'react-router-dom';

import { Mode, UIProvider } from 'components/kit/Theme';

import ThemeToggle, { ThemeOptions } from './ThemeToggle';

const ThemeToggleContainer: React.FC = () => (
  <UIProvider>
    <BrowserRouter>
      <ThemeToggle />
    </BrowserRouter>
  </UIProvider>
);

const user = userEvent.setup();

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
      await user.click(screen.getByText(option.displayName));
      option = ThemeOptions[option.next];
    }
  });
});
