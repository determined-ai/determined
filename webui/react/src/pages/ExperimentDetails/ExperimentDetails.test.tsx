import { render, screen, waitFor } from '@testing-library/react';
import { HelmetProvider } from 'react-helmet-async';

import {
  getExperimentDetails,
  getExpTrials,
  getExpValidationHistory,
  getProject,
  getTrialDetails,
  getWorkspace,
} from 'services/api';
import { StoreProvider as UIProvider } from 'stores/contexts/UI';
import {} from 'stores/cluster';

import ExperimentDetails, { ERROR_MESSAGE, INVALID_ID_MESSAGE } from './ExperimentDetails';
import RESPONSES from './ExperimentDetails.test.mock';

vi.useFakeTimers();
/**
 * Setup mock functions in a way that the responses can
 * be overridden dynamically between test sections.
 * The idea is to import the function from the module,
 * mock the module and replace the function(s) with vi.fn(),
 * then override the implementation or return value
 */
const { BrowserRouter, useParams } = await import('react-router-dom');
vi.mock('react-router-dom', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router-dom')>()),
  useParams: vi.fn(),
}));

vi.mock('services/api', () => ({
  getExperimentDetails: vi.fn(),
  getExpTrials: vi.fn(),
  getExpValidationHistory: vi.fn(),
  getProject: vi.fn(),
  getResourcePools: vi.fn().mockReturnValue(Promise.resolve([])),
  getTrialDetails: vi.fn(),
  getWorkspace: vi.fn(),
  getWorkspaceProjects: vi.fn().mockReturnValue({ projects: [] }),
  getWorkspaces: vi.fn().mockResolvedValue({ workspaces: [] }),
}));

/**
 * TODO: Temporarily mock ExperimentVisualization module.
 * This is a challenging module to test as it has `readStream` calls.
 * For now, simply return a simple placeholder.
 */
vi.mock('./ExperimentVisualization', () => ({
  __esModule: true,
  default: () => <div>Experiment Visualization</div>,
}));

const setup = () => {
  const view = render(
    <UIProvider>
      <HelmetProvider>
        <BrowserRouter>
          <ExperimentDetails />
        </BrowserRouter>
      </HelmetProvider>
    </UIProvider>,
  );
  return { view };
};

describe('Experiment Details Page', () => {
  describe('Invalid Experiment ID', () => {
    const INVALID_ID = 'beadbead';

    beforeAll(() => {
      vi.mocked(useParams).mockReturnValue({ experimentId: INVALID_ID });
    });

    it('should show invalid experiment page without id', async () => {
      setup();
      const invalidMessage = await screen.findByText(`${INVALID_ID_MESSAGE} ${INVALID_ID}`);
      expect(invalidMessage).toBeInTheDocument();
    });
  });

  describe('Missing Experiment', () => {
    const NON_EXISTING_ID = 9999;

    beforeAll(() => {
      vi.mocked(useParams).mockReturnValue({ experimentId: `${NON_EXISTING_ID}` });
      vi.mocked(getExperimentDetails).mockRejectedValue(new Error('Fetch Error'));
    });

    it('should show experiment is unfetchable', async () => {
      setup();
      const errorMessage = await screen.findByText(`${ERROR_MESSAGE} ${NON_EXISTING_ID}`);
      expect(errorMessage).toBeInTheDocument();
    });
  });

  describe('Single Trial Experiment', () => {
    beforeAll(() => {
      vi.mocked(useParams).mockReturnValue({ experimentId: '1241' });
      vi.mocked(getExperimentDetails).mockResolvedValue(
        RESPONSES.singleTrial.getExperimentsDetails,
      );
      vi.mocked(getExpValidationHistory).mockResolvedValue(
        RESPONSES.singleTrial.getExpValidationHistory,
      );
      vi.mocked(getExpTrials).mockResolvedValue(RESPONSES.singleTrial.getExpTrials);
      vi.mocked(getProject).mockResolvedValue(RESPONSES.singleTrial.getProject);
      vi.mocked(getTrialDetails).mockResolvedValue(RESPONSES.singleTrial.getTrialDetails);
      vi.mocked(getWorkspace).mockResolvedValue(RESPONSES.multiTrial.getWorkspace);
    });

    it('should show single trial experiment page with id', async () => {
      setup();

      const experimentId = RESPONSES.singleTrial.getExperimentsDetails.id;
      const experimentName = RESPONSES.singleTrial.getExperimentsDetails.name;
      await waitFor(() => {
        expect(screen.getByText(`Experiment ${experimentId}`)).toBeInTheDocument();
        expect(screen.getByRole('experimentName')).toHaveTextContent(experimentName);
      });

      expect(screen.getByText('Overview')).toBeInTheDocument();
      expect(screen.getByText('Hyperparameters')).toBeInTheDocument();
      expect(screen.getByText('Logs')).toBeInTheDocument();
    });
  });

  describe('Multi-Trial Experiment', () => {
    beforeAll(() => {
      vi.mocked(useParams).mockReturnValue({ experimentId: '1249' });
      vi.mocked(getExperimentDetails).mockResolvedValue(RESPONSES.multiTrial.getExperimentsDetails);
      vi.mocked(getExpValidationHistory).mockResolvedValue(
        RESPONSES.multiTrial.getExpValidationHistory,
      );
      vi.mocked(getExpTrials).mockResolvedValue(RESPONSES.multiTrial.getExpTrials);
      vi.mocked(getProject).mockResolvedValue(RESPONSES.multiTrial.getProject);
      vi.mocked(getWorkspace).mockResolvedValue(RESPONSES.multiTrial.getWorkspace);
    });

    it('should show multi-trial experiment page with id', async () => {
      setup();

      const experimentId = RESPONSES.multiTrial.getExperimentsDetails.id;
      const experimentName = RESPONSES.multiTrial.getExperimentsDetails.name;
      await waitFor(() => {
        expect(screen.getByText(`Experiment ${experimentId}`)).toBeInTheDocument();
        expect(screen.getByRole('experimentName')).toHaveTextContent(experimentName);
      });

      expect(screen.getByText('Visualization')).toBeInTheDocument();
      expect(screen.getAllByText('Trials').length).toBeGreaterThan(0);
    });
  });
});
