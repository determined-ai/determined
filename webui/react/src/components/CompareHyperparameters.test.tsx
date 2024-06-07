//

import { render, screen } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { BrowserRouter } from 'react-router-dom';

import { SettingsProvider } from 'hooks/useSettingsProvider';

import { COMPARE_HEAT_MAPS } from './CompareHeatMaps';
import { NO_DATA_MESSAGE } from './CompareHyperparameters';
import { CompareHyperparametersWithMocks } from './CompareHyperparameters.test.mock';
import { COMPARE_PARALLEL_COORDINATES } from './CompareParallelCoordinates';
import { COMPARE_SCATTER_PLOTS } from './CompareScatterPlots';
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

const setup = (empty?: boolean) => {
  render(
    <BrowserRouter>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <SettingsProvider>
            <CompareHyperparametersWithMocks empty={empty} />
          </SettingsProvider>
        </ThemeProvider>
      </UIProvider>
    </BrowserRouter>,
  );
};

describe('CompareHyperparameters component', () => {
  it('renders Parallel Coordinates', () => {
    setup();
    expect(screen.getByTestId(COMPARE_PARALLEL_COORDINATES)).toBeInTheDocument();
  });
  it('renders Scatter Plots', () => {
    setup();
    expect(screen.getByTestId(COMPARE_SCATTER_PLOTS)).toBeInTheDocument();
  });
  it('renders Heat Maps', () => {
    setup();
    expect(screen.getByTestId(COMPARE_HEAT_MAPS)).toBeInTheDocument();
  });
  it('renders no data state', () => {
    setup(true);
    expect(screen.queryByTestId(COMPARE_PARALLEL_COORDINATES)).not.toBeInTheDocument();
    expect(screen.queryByTestId(COMPARE_SCATTER_PLOTS)).not.toBeInTheDocument();
    expect(screen.queryByTestId(COMPARE_HEAT_MAPS)).not.toBeInTheDocument();
    expect(screen.queryByText(NO_DATA_MESSAGE)).toBeInTheDocument();
  });
});
