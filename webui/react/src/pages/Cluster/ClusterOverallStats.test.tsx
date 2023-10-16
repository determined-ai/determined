import { render } from '@testing-library/react';
import { theme, isDarkMode } from 'utils/tests/getTheme';
import { UIProvider, ThemeProvider } from 'components/kit/Theme';

import { ClusterOverallStats } from './ClusterOverallStats';

vi.mock('services/api', () => ({
  getActiveTasks: () => Promise.resolve({ commands: 0, notebooks: 0, shells: 0, tensorboards: 0 }),
  getAgents: () => Promise.resolve([]),
  getExperiments: () => Promise.resolve({ experiments: [], pagination: { total: 0 } }),
  getResourcePools: () => Promise.resolve({}),
}));

const setup = () => {
  const view = render(
    <ThemeProvider>
      <UIProvider theme={theme} darkMode={isDarkMode}>
        <ClusterOverallStats />
      </UIProvider>
    </ThemeProvider>
  );
  return { view };
};

describe('ClusterOverallStats', () => {
  it('displays cluster overall stats ', () => {
    const { view } = setup();
    expect(view.getByText('Connected Agents')).toBeInTheDocument();
  });
});
