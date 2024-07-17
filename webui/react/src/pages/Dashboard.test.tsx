import { render, screen, waitFor } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { useEffect } from 'react';

import { ThemeProvider } from 'components/ThemeProvider';
import authStore from 'stores/auth';
import userStore from 'stores/users';
import { DetailedUser } from 'types';

import Dashboard from './Dashboard';
vi.mock('services/api', () => ({
  getCommands: () => Promise.resolve([]),
  getExperiments: () => Promise.resolve({
    experiments: [],
  }),
  getJupyterLabs: () => Promise.resolve([]),
  getProjectsByUserActivity: () => Promise.resolve([]),
  getShells: () => Promise.resolve([]),
  getTensorBoards: () => Promise.resolve([]),
}));

const CURRENT_USER: DetailedUser = { id: 1, isActive: true, isAdmin: false, username: 'bunny' };

const Container: React.FC = () => {
  useEffect(() => {
    authStore.setAuth({ isAuthenticated: true });
    authStore.setAuthChecked();
    userStore.updateCurrentUser(CURRENT_USER);
  }, []);

  return (
    <Dashboard testWithoutPage />
  );
};

const setup = () => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <Container />
      </ThemeProvider>
    </UIProvider>,
  );
};

describe('Dashboard', () => {
  it('renders', async () => {
    setup();
    await waitFor(() => {
      expect(screen.getByText('Your Recent Submissions')).toBeInTheDocument();
    });
  });
});
