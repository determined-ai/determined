import { render, screen } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { BrowserRouter } from 'react-router-dom';

import { SettingsProvider } from 'hooks/useSettingsProvider';

import { COMPARE_HEAT_MAPS } from './CompareHeatMaps';
import { NO_DATA_MESSAGE } from './CompareHyperparameters';
import {
  CompareRunHyperparametersWithMocks,
  CompareTrialHyperparametersWithMocks,
} from './CompareHyperparameters.test.mock';
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
const mockFlatRunsFeatureCheck = vi.hoisted(() => vi.fn());
vi.mock('hooks/useFeature', () => ({
  default: () => ({
    isOn: mockFlatRunsFeatureCheck,
  }),
}));

const setup = (
  type: 'trials' | 'runs',
  state: 'empty' | 'uncomparable_metrics' | 'no_metrics' | 'normal' = 'normal',
) => {
  render(
    <BrowserRouter>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <SettingsProvider>
            {type === 'trials' ? (
              <CompareTrialHyperparametersWithMocks state={state} />
            ) : (
              <CompareRunHyperparametersWithMocks state={state} />
            )}
          </SettingsProvider>
        </ThemeProvider>
      </UIProvider>
    </BrowserRouter>,
  );
};

describe('CompareHyperparameters component', () => {
  describe.each(['trials', 'runs'] as const)('%s', (type) => {
    it('renders Parallel Coordinates', () => {
      setup(type);
      expect(screen.getByTestId(COMPARE_PARALLEL_COORDINATES)).toBeInTheDocument();
    });
    it('renders Parallel Coordinates error when metrics are incompatable', () => {
      setup(type, 'uncomparable_metrics');
      expect(
        screen.getByText('Records are not comparable using current parameters.'),
      ).toBeInTheDocument();
    });
    it('renders Scatter Plots', () => {
      setup(type);
      expect(screen.getByTestId(COMPARE_SCATTER_PLOTS)).toBeInTheDocument();
    });
    it('renders Heat Maps', () => {
      setup(type);
      expect(screen.getByTestId(COMPARE_HEAT_MAPS)).toBeInTheDocument();
    });
    it('renders no data state', () => {
      setup(type, 'empty');
      expect(screen.queryByTestId(COMPARE_PARALLEL_COORDINATES)).not.toBeInTheDocument();
      expect(screen.queryByTestId(COMPARE_SCATTER_PLOTS)).not.toBeInTheDocument();
      expect(screen.queryByTestId(COMPARE_HEAT_MAPS)).not.toBeInTheDocument();
      expect(screen.queryByText(NO_DATA_MESSAGE)).toBeInTheDocument();
    });

    describe.each([true, false])('when f_flat_runs is %s', (featureIsOn) => {
      afterEach(() => {
        mockFlatRunsFeatureCheck.mockReset();
      });
      it('renders no metrics state', () => {
        mockFlatRunsFeatureCheck.mockReturnValue(featureIsOn);
        setup(type, 'no_metrics');
        expect(screen.getByText('Not enough data points to plot.')).toBeInTheDocument();
        if (featureIsOn) {
          expect(
            screen.getByText('Please wait until the searches are further along.'),
          ).toBeInTheDocument();
        } else {
          expect(
            screen.getByText('Please wait until the experiments are further along.'),
          ).toBeInTheDocument();
        }
      });
    });
  });
});
