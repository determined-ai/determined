import { render } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { BrowserRouter } from 'react-router-dom';

import { ThemeProvider } from 'components/ThemeProvider';

import TrialDetails from './TrialDetails';

const mockUseFeature = vi.hoisted(() => vi.fn());
const mockGetExperimentDetails = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
const mockGetTrialDetails = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
const mockGetTrialRemainingLogRetentionDays = vi.hoisted(() => vi.fn(() => new Promise(() => {})));
vi.mock('hooks/useFeature', () => ({
  default: () => ({
    isOn: mockUseFeature,
  }),
}));

vi.mock('servicses/api', () => {
  const baseObject = {
    getExperimentDetails: mockGetExperimentDetails,
    getTrialDetails: mockGetTrialDetails,
    getTrialRemainingLogRetentionDays: mockGetTrialRemainingLogRetentionDays,
  };
  return new Proxy(baseObject, {
    get(target, prop) {
      if (prop in target) {
        return target[prop];
      }
      return () => {
        console.error(`unhandled request: ${prop.toString()}`);
        return new Promise(() => {});
      };
    },
  });
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
