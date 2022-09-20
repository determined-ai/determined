import { render, screen, waitFor } from '@testing-library/react';
import React from 'react';
import { HelmetProvider } from 'react-helmet-async';
import { Router, useParams } from 'react-router-dom';

import StoreProvider from 'contexts/Store';
import {
  getExperimentDetails,
  getExpTrials,
  getExpValidationHistory,
  getProject,
  getTrialDetails,
  getWorkspace,
} from 'services/api';
import history from 'shared/routes/history';

import ExperimentDetails, { ERROR_MESSAGE, INVALID_ID_MESSAGE } from './ExperimentDetails';
import RESPONSES from './ExperimentDetails.test.mock';

/**
 * Setup mock functions in a way that the responses can
 * be overridden dynamically between test sections.
 * The idea is to import the function from the module,
 * mock the module and replace the function(s) with jest.fn(),
 * then override the implementation or return value
 */

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: jest.fn().mockReturnValue({ experimentId: undefined }),
}));

jest.mock('services/api', () => ({
  ...jest.requireActual('services/api'),
  getExperimentDetails: jest.fn(),
  getExpTrials: jest.fn(),
  getExpValidationHistory: jest.fn(),
  getProject: jest.fn(),
  getTrialDetails: jest.fn(),
  getWorkspace: jest.fn(),
}));

/**
 * TODO: Temporarily mock ExperimentVisualization module.
 * This is a challenging module to test as it has `readStream` calls.
 * For now, simply return a simple placeholder.
 */
jest.mock('./ExperimentVisualization', () => ({
  __esModule: true,
  default: () => <div>Experiment Visualization</div>,
}));

const setup = () => {
  const view = render(
    <StoreProvider>
      <HelmetProvider>
        <Router history={history}>
          <ExperimentDetails />
        </Router>
      </HelmetProvider>
    </StoreProvider>,
  );
  return { view };
};

describe('Experment Details Page', () => {
  describe('Invalid Experiment ID', () => {
    const INVALID_ID = 'beadbead';

    beforeAll(() => {
      (useParams as jest.Mock).mockReturnValue({ experimentId: INVALID_ID });
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
      (useParams as jest.Mock).mockReturnValue({ experimentId: NON_EXISTING_ID });
      (getExperimentDetails as jest.Mock).mockRejectedValue(new Error('Fetch Error'));
    });

    it('should show experiment is unfetchable', async () => {
      setup();
      const errorMessage = await screen.findByText(`${ERROR_MESSAGE} ${NON_EXISTING_ID}`);
      expect(errorMessage).toBeInTheDocument();
    });
  });

  describe('Single Trial Experiment', () => {
    beforeAll(() => {
      (useParams as jest.Mock).mockReturnValue({ experimentId: 1241 });
      (getExperimentDetails as jest.Mock).mockResolvedValue(
        RESPONSES.singleTrial.getExperimentsDetails,
      );
      (getExpValidationHistory as jest.Mock).mockResolvedValue(
        RESPONSES.singleTrial.getExpValidationHistory,
      );
      (getExpTrials as jest.Mock).mockResolvedValue(RESPONSES.singleTrial.getExpTrials);
      (getProject as jest.Mock).mockResolvedValue(RESPONSES.singleTrial.getProject);
      (getTrialDetails as jest.Mock).mockResolvedValue(RESPONSES.singleTrial.getTrialDetails);
      (getWorkspace as jest.Mock).mockResolvedValue(RESPONSES.multiTrial.getWorkspace);
    });

    it('should show single trial experiment page with id', async () => {
      const { container } = setup().view;

      const experimentId = RESPONSES.singleTrial.getExperimentsDetails.id;
      const experimentName = RESPONSES.singleTrial.getExperimentsDetails.name;
      await waitFor(() => {
        expect(screen.getByText(`Experiment ${experimentId}`)).toBeInTheDocument();
        expect(container.querySelector(`[data-value="${experimentName}"]`)).toBeInTheDocument();
      });

      expect(screen.getByText('Overview')).toBeInTheDocument();
      expect(screen.getByText('Hyperparameters')).toBeInTheDocument();
      expect(screen.getByText('Logs')).toBeInTheDocument();
    });
  });

  describe('Multi-Trial Experiment', () => {
    beforeAll(() => {
      (useParams as jest.Mock).mockReturnValue({ experimentId: 1249 });
      (getExperimentDetails as jest.Mock).mockResolvedValue(
        RESPONSES.multiTrial.getExperimentsDetails,
      );
      (getExpValidationHistory as jest.Mock).mockResolvedValue(
        RESPONSES.multiTrial.getExpValidationHistory,
      );
      (getExpTrials as jest.Mock).mockResolvedValue(RESPONSES.multiTrial.getExpTrials);
      (getProject as jest.Mock).mockResolvedValue(RESPONSES.multiTrial.getProject);
      (getWorkspace as jest.Mock).mockResolvedValue(RESPONSES.multiTrial.getWorkspace);
    });

    it('should show multi-trial experiment page with id', async () => {
      const { container } = setup().view;

      const experimentId = RESPONSES.multiTrial.getExperimentsDetails.id;
      const experimentName = RESPONSES.multiTrial.getExperimentsDetails.name;
      await waitFor(() => {
        expect(screen.getByText(`Experiment ${experimentId}`)).toBeInTheDocument();
        expect(container.querySelector(`[data-value="${experimentName}"]`)).toBeInTheDocument();
      });

      expect(screen.getByText('Visualization')).toBeInTheDocument();
      expect(screen.getByText('Trials')).toBeInTheDocument();
    });
  });
});
