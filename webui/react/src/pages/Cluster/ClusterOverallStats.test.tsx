import { render } from '@testing-library/react';
import React from 'react';

import StoreProvider from 'contexts/Store';
import { ResourcePoolsProvider } from 'stores/resourcePools';

import { ClusterOverallStats } from './ClusterOverallStats';

const setup = () => {
  const view = render(
    <StoreProvider>
      <ResourcePoolsProvider>
        <ClusterOverallStats />
      </ResourcePoolsProvider>
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
