import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DefaultTheme, UIProvider } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';
import React, { useEffect } from 'react';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';
import authStore from 'stores/auth';

import InteractiveTask from './InteractiveTask';

const TASK_NAME = 'JupyterLab (test-task-name)';
const TASK_RESOURCE_POOL = 'aux-pool';

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useParams: () => ({
    taskId: 'task-id',
    taskName: TASK_NAME,
    taskResourcePool: TASK_RESOURCE_POOL,
    taskType: 'jupyter-lab',
    taskUrl: 'http://taskUrl.com',
  }),
  useRouteMatch: () => ({ url: '/company/company-id1/team/team-id1' }),
}));

vi.mock('services/api', () => ({
  getCommand: () => {
    return Promise.resolve({});
  },
  getJupyterLab: () => {
    return Promise.resolve({});
  },
  getShell: () => {
    return Promise.resolve({});
  },
  getTensorBoard: () => {
    return Promise.resolve({});
  },
}));

const InteractiveTaskPageContainer: React.FC = () => {
  useEffect(() => {
    authStore.setAuth({ isAuthenticated: true });
  }, []);

  return <InteractiveTask />;
};

const InteractiveTaskContainer: React.FC = () => {
  return (
    <BrowserRouter>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <HelmetProvider>
            <ConfirmationProvider>
              <InteractiveTaskPageContainer />
            </ConfirmationProvider>
          </HelmetProvider>
        </ThemeProvider>
      </UIProvider>
    </BrowserRouter>
  );
};

const setup = () => render(<InteractiveTaskContainer />);

describe('InteractiveTask', () => {
  it('should render page with task name and resource pool', async () => {
    setup();
    expect(await screen.findByText(TASK_NAME)).toBeInTheDocument();
    expect(await screen.findByText(TASK_RESOURCE_POOL)).toBeInTheDocument();
  });

  it('should render page with context menu', async () => {
    setup();
    userEvent.click(screen.getByTestId('task-action-dropdown-trigger'));
    expect(await screen.findByText('Kill')).toBeInTheDocument();
    expect(await screen.findByText('View Logs')).toBeInTheDocument();
  });

  it('should render page with correct title', () => {
    setup();
    expect(document.title).toEqual(TASK_NAME);
  });
});
