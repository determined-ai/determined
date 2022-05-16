import { render, screen} from '@testing-library/react';
import React, { useEffect } from 'react';

import StoreProvider, { StoreAction, useStoreDispatch } from 'contexts/Store';

import InteractiveTask from './InteractiveTask';

const TASK_NAME = "JupyterLab (test-task-name)"
const TASK_RESOURCE_POOL = 'aux-pool'

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'), // use actual for all non-hook parts
  useParams: () => ({
    taskName: TASK_NAME,
    taskResourcePool: TASK_RESOURCE_POOL,
    taskUrl:'http://taskUrl.com',
    taskType:'JupyterLab',
    taskId:'task-id'
  }),
  useRouteMatch: () => ({ url: '/company/company-id1/team/team-id1' }),
}));

const InteractiveTaskPageContainer: React.FC = () => {

  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return (
    <InteractiveTask/>
  );
};

const InteractiveTaskContainer: React.FC = () => {
  return (
    <StoreProvider>
      <InteractiveTaskPageContainer />
    </StoreProvider>
  );
};

const setup = async () => {
  render(
    <InteractiveTaskContainer />,
  );
};

describe('InteractiveTask', () => {
  it('task name and resource pool are shown', async () => {
    await setup();
    expect(await screen.findByText(TASK_NAME)).toBeInTheDocument();
    expect(await screen.findByText(TASK_RESOURCE_POOL)).toBeInTheDocument();
  });

});
