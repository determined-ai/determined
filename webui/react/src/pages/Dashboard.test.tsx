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
  getProjectsByUserActivity: () => Promise.resolve([
    {
      archived: false,
      description: '',
      errorMessage: '',
      id: 1,
      immutable: true,
      key: '',
      lastExperimentStartedAt: '2024-07-17T16:18:56.813686Z',
      name: 'Uncategorized',
      notes: [
        {
          contents: '',
          name: 'Untitled',
        },
      ],
      numActiveExperiments: 0,
      numExperiments: 1297,
      numRuns: 41995,
      state: 'UNSPECIFIED',
      userId: 1,
      username: 'admin',
      workspaceId: 1,
      workspaceName: 'Uncategorized',
    },
  ]),
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
  beforeEach(() => {
    setup();
  });

  it('renders with correct titles', async () => {
    await waitFor(() => {
      expect(screen.getByText('Recently Viewed Projects')).toBeInTheDocument();
      expect(screen.getByText('Your Recent Submissions')).toBeInTheDocument();
    });
  });

  it('renders ProjectCards with project data', async () => {
    await waitFor(() => {
      expect(screen.getByText('Uncategorized')).toBeInTheDocument();
    });
  });

  it('renders empty state for submissions', async () => {
    await waitFor(() => {
      expect(screen.getByText('No submissions')).toBeInTheDocument();
      expect(screen.getByText('Your recent experiments and tasks will show up here.')).toBeInTheDocument();
      expect(screen.getByText('Get started')).toBeInTheDocument();
    });
  });
});
