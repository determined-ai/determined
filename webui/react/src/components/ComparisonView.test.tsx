import { render, screen } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { BrowserRouter } from 'react-router-dom';

import { SettingsProvider } from 'hooks/useSettingsProvider';

import { EMPTY_MESSAGE } from './ComparisonView';
import {
  ExperimentComparisonViewWithMocks,
  RunComparisonViewWithMocks,
} from './ComparisonView.test.mock';
import { ThemeProvider } from './ThemeProvider';

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
                open={true}
                onWidthChange={handleWidthChange}>
                <p>Children</p>
              </ExperimentComparisonViewWithMocks>
            ) : (
              <RunComparisonViewWithMocks
                empty={empty}
                open={true}
                onWidthChange={handleWidthChange}>
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
  describe.each(['experiments', 'runs'] as const)('%s', (type) => {
    it('renders children', () => {
      setup(type);
      expect(screen.getByText('Children')).toBeInTheDocument();
    });
    it('shows empty message', () => {
      setup(type, true);
      expect(screen.getByText(EMPTY_MESSAGE)).toBeInTheDocument();
    });
  });
});
