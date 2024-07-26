import { render } from '@testing-library/react';
import HewUIProvider, { DefaultTheme } from 'hew/Theme';
import { useInitApi } from 'hew/Toast';
import { ConfirmationProvider } from 'hew/useConfirm';
import { HelmetProvider } from 'react-helmet-async';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';
import { generateTestExperimentData } from 'utils/tests/generateTestData';

import TrialDetails from './TrialDetails';

const { trial, experiment } = generateTestExperimentData();

const mockLogRetentionResponse = {
  remainingLogRetentionDays: -1,
};

const mockUseFeature = vi.hoisted(() => vi.fn());
vi.mock('hooks/useFeature', () => ({
  default: () => ({
    isOn: mockUseFeature,
  }),
}));

const mockUseParams = vi.hoisted(() => vi.fn(() => ({ experimentId: 7823, trialId: 54176 })));
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useParams: mockUseParams,
}));

const mockGetExperimentDetails = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
const mockGetTrialDetails = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
const mockGetTrialRemainingLogRetentionDays = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
const mockGetTrialWorkloads = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
vi.mock('services/api', () => {
  return {
    getExperimentDetails: mockGetExperimentDetails,
    getTrialDetails: mockGetTrialDetails,
    getTrialRemainingLogRetentionDays: mockGetTrialRemainingLogRetentionDays,
    getTrialWorkloads: mockGetTrialWorkloads,
  };
});

const Wrapper = () => {
  useInitApi();
  return <TrialDetails />;
};

const setup = () => {
  return render(
    <BrowserRouter>
      <HelmetProvider>
        <ThemeProvider>
          <HewUIProvider theme={DefaultTheme.Light}>
            <ConfirmationProvider>
              <Wrapper />
            </ConfirmationProvider>
          </HewUIProvider>
        </ThemeProvider>
      </HelmetProvider>
    </BrowserRouter>,
  );
};

describe('TrialDetails', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });
  it('renders loading state', () => {
    const container = setup();

    expect(container.getByText(/Fetching/)).toBeInTheDocument();
  });
  it('shows page when requests load', async () => {
    mockGetTrialDetails.mockReturnValue(Promise.resolve(trial));
    mockGetExperimentDetails.mockReturnValue(Promise.resolve(experiment));
    mockGetTrialRemainingLogRetentionDays.mockReturnValue(
      Promise.resolve(mockLogRetentionResponse),
    );

    const container = setup();

    expect(await container.findByText(/Uncategorized/)).toBeInTheDocument();
  });

  describe.each([true, false])('when f_flat_runs is %s', (f_flat_runs) => {
    it('shows proper copy', async () => {
      mockGetTrialDetails.mockReturnValue(Promise.resolve(trial));
      mockGetExperimentDetails.mockReturnValue(Promise.resolve(experiment));
      mockGetTrialRemainingLogRetentionDays.mockReturnValue(
        Promise.resolve(mockLogRetentionResponse),
      );
      mockUseFeature.mockReturnValue(f_flat_runs);

      const container = setup();
      expect(
        container.getByText(new RegExp(`Fetching ${f_flat_runs ? 'run' : 'trial'}`)),
      ).toBeInTheDocument();

      expect(
        await container.findByText(
          new RegExp(`Uncategorized ${f_flat_runs ? 'Runs' : 'Experiments'}`),
        ),
      ).toBeInTheDocument();
    });
  });
});
