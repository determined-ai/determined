import { render, screen } from '@testing-library/react';
import userEvent, { UserEvent } from '@testing-library/user-event';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';
import SearchDetails from 'pages/SearchDetails';

vi.mock('services/api', () => ({
  getExperimentDetails: vi.fn(),
  patchExperiment: vi.fn(),
}));

const setup = (): { user: UserEvent } => {
  const user = userEvent.setup();

  render(
    <BrowserRouter>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <HelmetProvider>
            <SearchDetails />
          </HelmetProvider>
        </ThemeProvider>
      </UIProvider>
    </BrowserRouter>,
  );

  return { user };
};

describe('SearchDetails', () => {
  it('should have tabs', () => {
    setup();

    expect(screen.getByRole('tab', { name: 'Runs' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Notes' })).toBeInTheDocument();
  });
});
