import { render } from '@testing-library/react';
import React from 'react';

import StoreProvider from 'contexts/Store';
import { ExperimentsProvider } from 'stores/experiments';
import { TasksProvider } from 'stores/tasks';

import { ClusterOverallStats } from './ClusterOverallStats';

const setup = () => {
  const view = render(
    <StoreProvider>
      <ExperimentsProvider>
        <TasksProvider>
          <ClusterOverallStats />
        </TasksProvider>
      </ExperimentsProvider>
    </StoreProvider>,
  );
  return { view };
};

describe('ClusterOverallStats', () => {
  it('displays cluster overall stats ', () => {
    const { view } = setup();
    expect(view.getByText('Connected Agents')).toBeInTheDocument();
  });
});
