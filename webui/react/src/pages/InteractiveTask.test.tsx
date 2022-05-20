import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';

import InteractiveTask from './InteractiveTask';

const TASK_NAME = 'JupyterLab (test-task-name)';
const TASK_RESOURCE_POOL = 'aux-pool';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'), // use actual for all non-hook parts
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

  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return (
    <InteractiveTask />
  );
};

const InteractiveTaskContainer: React.FC = () => {
  return (
    <StoreProvider>
      <InteractiveTaskPageContainer />
    </StoreProvider>
  );
};

const setup = () => render(<InteractiveTaskContainer />);

describe('InteractiveTask', () => {
  it('task name and resource pool are shown', async () => {
    await setup();
    expect(await screen.findByText(TASK_NAME)).toBeInTheDocument();
    expect(await screen.findByText(TASK_RESOURCE_POOL)).toBeInTheDocument();
  });

  it('Context menu is shown', async () => {
    await setup();
    userEvent.click(screen.getByTestId('task-action-dropdown-trigger'));
    expect(await screen.findByText('Kill')).toBeInTheDocument();
    expect(await screen.findByText('View Logs')).toBeInTheDocument();
  });

});
