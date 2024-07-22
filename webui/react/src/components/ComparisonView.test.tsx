import { render, screen } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { BrowserRouter } from 'react-router-dom';

import { SettingsProvider } from 'hooks/useSettingsProvider';

import { EMPTY_MESSAGE } from './ComparisonView';
import {
  ExperimentComparisonViewWithMocks,
  METRIC_DATA,
  RunComparisonViewWithMocks,
  SELECTED_EXPERIMENTS,
  SELECTED_RUNS,
} from './ComparisonView.test.mock';
import { ThemeProvider } from './ThemeProvider';

vi.mock('services/api', () => ({
  searchExperiments: () => {
    return {
      experiments: SELECTED_EXPERIMENTS,
    };
  },
  searchRuns: () => {
    return {
      runs: SELECTED_RUNS,
    };
  },
}));

vi.mock('hooks/useSettings', async (importOriginal) => {
  const useSettings = vi.fn(() => {
    const settings = {
      hParams: ['learning_rate'],
      metric: {
        group: 'training',
        name: 'loss',
      },
      scale: 'linear',
    };
    const updateSettings = vi.fn();

    return { isLoading: false, settings, updateSettings };
  });

  return {
    __esModule: true,
    ...(await importOriginal<typeof import('hooks/useSettings')>()),
    useSettings,
  };
});

vi.mock('hooks/useMetrics', async (importOriginal) => {
  const useMetrics = vi.fn(() => {
    return METRIC_DATA;
  });

  return {
    __esModule: true,
    ...(await importOriginal<typeof import('hooks/useMetrics')>()),
    useMetrics,
  };
});

const setup = (type: 'experiments' | 'runs', empty?: boolean) => {
  const handleWidthChange = vi.fn();
  render(
    <BrowserRouter>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <SettingsProvider>
            {type === 'experiments' ? (
              <ExperimentComparisonViewWithMocks
                empty={empty}
                open
                onWidthChange={handleWidthChange}>
                <p>Children</p>
              </ExperimentComparisonViewWithMocks>
            ) : (
              <RunComparisonViewWithMocks empty={empty} open onWidthChange={handleWidthChange}>
                <p>Children</p>
              </RunComparisonViewWithMocks>
            )}
          </SettingsProvider>
        </ThemeProvider>
      </UIProvider>
    </BrowserRouter>,
  );

  return { handleWidthChange };
};

describe('ComparisonView', () => {
  describe('Experiments', () => {
    it('renders children', () => {
      setup('experiments');
      expect(screen.getByText('Children')).toBeInTheDocument();
    });
    it('shows empty message', () => {
      setup('experiments', true);
      expect(screen.getByText(EMPTY_MESSAGE)).toBeInTheDocument();
    });
  });
  describe('Runs', () => {
    it('renders children', () => {
      setup('runs');
      expect(screen.getByText('Children')).toBeInTheDocument();
    });
    it('shows empty message', () => {
      setup('runs', true);
      expect(screen.getByText(EMPTY_MESSAGE)).toBeInTheDocument();
    });
  });
});
