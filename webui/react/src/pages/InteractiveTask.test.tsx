import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { StoreProvider as UIProvider } from 'shared/contexts/stores/UI';
import authStore from 'stores/auth';

import { ConfirmationProvider } from '../components/kit/useConfirm';

import InteractiveTask from './InteractiveTask';

const TASK_NAME = 'JupyterLab (test-task-name)';
const TASK_RESOURCE_POOL = 'aux-pool';

vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useParams: () => ({
    taskId: 'task-id',
    taskName: TASK_NAME,
    taskResourcePool: TASK_RESOURCE_POOL,
    taskType: 'JupyterLab',
    taskUrl: 'http://taskUrl.com',
  }),
  useRouteMatch: () => ({ url: '/company/company-id1/team/team-id1' }),
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
      <UIProvider>
        <HelmetProvider>
          <ConfirmationProvider>
            <InteractiveTaskPageContainer />
          </ConfirmationProvider>
        </HelmetProvider>
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
