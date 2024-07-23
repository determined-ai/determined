import { render } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';

import TrialDetails from './TrialDetails';

const mockUseFeature = vi.hoisted(() => vi.fn());
vi.mock('hooks/useFeature', () => ({
  default: () => ({
    isOn: mockUseFeature,
  }),
}));

const mockUseParams = vi.hoisted(() => vi.fn(() => ({ experimentId: 1, trialId: 1 })));
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useParams: mockUseParams,
}));

const mockGetExperimentDetails = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
const mockGetTrialDetails = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
const mockGetTrialRemainingLogRetentionDays = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
vi.mock('services/api', () => {
  return {
    getExperimentDetails: mockGetExperimentDetails,
    getTrialDetails: mockGetTrialDetails,
    getTrialRemainingLogRetentionDays: mockGetTrialRemainingLogRetentionDays,
  };
});

const setup = () => {
  return render(
    <BrowserRouter>
      <ThemeProvider>
        <UIProvider theme={DefaultTheme.Light}>
          <TrialDetails />
        </UIProvider>
      </ThemeProvider>
    </BrowserRouter>,
  );
};

describe('TrialDetails', () => {
  it('renders loading state', () => {
    const container = setup();

    expect(container.getByText(/Fetching .* information/)).toBeInTheDocument();
  });
});
