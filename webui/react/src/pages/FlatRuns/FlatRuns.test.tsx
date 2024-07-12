import { render, screen } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';
import { SettingsProvider } from 'hooks/useSettingsProvider';
import { getProjectColumns, getProjectNumericMetricsRange, searchRuns } from 'services/api';
import { ProjectColumn } from 'types';

import FlatRuns from './FlatRuns';
import RESPONSE from './FlatRuns.test.mock';

vi.mock('services/api', () => ({
  getProjectColumns: vi.fn(),
  getProjectNumericMetricsRange: vi.fn(),
  searchRuns: vi.fn(),
}));

vi.mock('hooks/useSettings', async (importOriginal) => {
  const useSettings = vi.fn(() => {
    const settings = {
      selections: [],
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

const mockProps = {
  projectId: 123,
  searchId: 432,
  workspaceId: 321,
};
const mockColumns: ProjectColumn[] = [
  {
    column: 'id',
    displayName: 'Global Run ID',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_NUMBER',
  },
  {
    column: 'experimentDescription',
    displayName: 'Search Description',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_TEXT',
  },
  { column: 'tags', displayName: 'Tags', location: 'LOCATION_TYPE_RUN', type: 'COLUMN_TYPE_TEXT' },
  {
    column: 'forkedFrom',
    displayName: 'Forked',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_NUMBER',
  },
  {
    column: 'startTime',
    displayName: 'Start Time',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_DATE',
  },
  {
    column: 'endTime',
    displayName: 'End Time',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_DATE',
  },
  {
    column: 'duration',
    displayName: 'Duration',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_NUMBER',
  },
  {
    column: 'state',
    displayName: 'State',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_TEXT',
  },
  {
    column: 'searcherType',
    displayName: 'Searcher',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_TEXT',
  },
  {
    column: 'resourcePool',
    displayName: 'Resource Pool',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_TEXT',
  },
  {
    column: 'checkpointSize',
    displayName: 'Checkpoint Size',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_NUMBER',
  },
  {
    column: 'checkpointCount',
    displayName: 'Checkpoints',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_NUMBER',
  },
  { column: 'user', displayName: 'User', location: 'LOCATION_TYPE_RUN', type: 'COLUMN_TYPE_TEXT' },
  {
    column: 'searcherMetric',
    displayName: 'Searcher Metric',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_TEXT',
  },
  {
    column: 'searcherMetricsVal',
    displayName: 'Searcher Metric Value',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_NUMBER',
  },
  {
    column: 'externalExperimentId',
    displayName: 'External Experiment ID',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_TEXT',
  },
  {
    column: 'projectId',
    displayName: 'Project ID',
    location: 'LOCATION_TYPE_RUN',
    type: 'COLUMN_TYPE_NUMBER',
  },
];
const mockHeatMap = [
  { max: 0.6561000000000001, metricsName: 'training.lr.min', min: 0.000010000000000000003 },
  {
    max: 0.07578124850988388,
    metricsName: 'training.train_top3_acc.last',
    min: 0.07578124850988388,
  },
  {
    max: 0.49296873807907104,
    metricsName: 'training.train_top36_acc.last',
    min: 0.49296873807907104,
  },
  { max: 0.609286835575863, metricsName: 'validation.m57.mean', min: 0.495158441024297 },
  { max: 1, metricsName: 'validation.some_val_metric.mean', min: 1 },
  { max: 0.9999386703315561, metricsName: 'validation.m197.max', min: 0.9878777677030942 },
  { max: 0.9999641983661229, metricsName: 'validation.m84.max', min: 0.9652832081416906 },
  { max: 0.801562488079071, metricsName: 'training.train_top78_acc.min', min: 0.801562488079071 },
  { max: 0.135937497019768, metricsName: 'training.train_top5_acc.max', min: 0.135937497019768 },
  { max: 0.05350976950983155, metricsName: 'validation.m16.min', min: 0.000020457237697169006 },
  { max: 0, metricsName: 'training.some_train_metric.min', min: 0 },
  { max: 4.55310354232788, metricsName: 'training.train_loss.mean', min: 0 },
  { max: 0.914843738079071, metricsName: 'training.train_top85_acc.max', min: 0.914843738079071 },
  { max: 0.621415996837992, metricsName: 'validation.m40.mean', min: 0.496130762236218 },
  { max: 0.6568516845108008, metricsName: 'validation.m239.last', min: 0.09100698454816025 },
];

const setup = () => {
  const view = render(
    <BrowserRouter>
      <UIProvider theme={DefaultTheme.Light}>
        <ThemeProvider>
          <SettingsProvider>
            <FlatRuns {...mockProps} />
          </SettingsProvider>
        </ThemeProvider>
      </UIProvider>
    </BrowserRouter>,
  );
  return { view };
};

describe('Flat Runs', () => {
  beforeAll(() => {
    vi.mocked(searchRuns).mockResolvedValue(RESPONSE);
    vi.mocked(getProjectColumns).mockReturnValue(Promise.resolve(mockColumns));
    vi.mocked(getProjectNumericMetricsRange).mockReturnValue(Promise.resolve(mockHeatMap));
  });
  describe('Runs count', () => {
    it('should show the runs count', async () => {
      setup();
      const runsCount = await screen.findByText(`${RESPONSE.runs.length} runs`);
      expect(runsCount).toBeVisible();
      expect(runsCount.innerText).toBe(`${RESPONSE.runs.length} runs`);
    });
    // it('should change the runs count when selecting runs', async () => {
    //   await setup();
    //   const runsCount = await screen.findByTestId('runSelection');
    //   expect(runsCount).toBeVisible();
    //   expect(runsCount.innerText).toBe(`${RESPONSE.runs.length} runs`);
    // });
  });
});
