import { render, screen } from '@testing-library/react';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { ConfirmationProvider } from 'hew/useConfirm';

import {} from 'stores/cluster';

import { ThemeProvider } from 'components/ThemeProvider';
import { ExperimentBase, TrialDetails } from 'types';
import { mockIntegrationData } from 'utils/integrations.test';

import TrialInfoBox from './TrialInfoBox';

vi.useFakeTimers();
const setup = (trial: TrialDetails, experiment: ExperimentBase) => {
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <ConfirmationProvider>
          <TrialInfoBox experiment={experiment} trial={trial} />
        </ConfirmationProvider>
      </ThemeProvider>
    </UIProvider>,
  );
};

const mockExperiment: ExperimentBase = {
  archived: false,
  config: {
    checkpointPolicy: 'best',
    checkpointStorage: {
      bucket: 'det-determined-master-us-west-2-573932760021',
      saveExperimentBest: 0,
      saveTrialBest: 1,
      saveTrialLatest: 1,
      type: 's3' as const,
    },
    hyperparameters: {
      global_batch_size: { type: 'const' as const, val: 64 },
      hidden_size: { type: 'const' as const, val: 64 },
      learning_rate: { maxval: 0.1, minval: 0.0001, type: 'double' as const },
    },
    labels: [],
    maxRestarts: 5,
    name: 'mnist_pytorch_lightning_adaptive',
    profiling: { enabled: false },
    resources: {},
    searcher: {
      max_length: { batches: 937, epochs: 1, records: 1 },
      max_trials: 16,
      metric: 'val_loss',
      name: 'adaptive_asha' as const,
      smallerIsBetter: true,
    },
  },
  configRaw: {},
  hyperparameters: {},
  id: 1,
  jobId: '',
  labels: [],
  name: '',
  numTrials: 3,
  originalConfig: '',
  parentArchived: false,
  projectId: 0,
  projectName: '',
  projectOwnerId: 0,
  resourcePool: '',
  searcherType: '',
  startTime: '',
  state: 'ACTIVE',
  trialIds: [1, 2, 3],
  userId: 0,
  workspaceId: 0,
  workspaceName: '',
};

const mockTrial1: TrialDetails = {
  autoRestarts: 0,
  bestAvailableCheckpoint: {
    endTime: '2022-08-05T05:56:47.998811Z',
    resources: {
      'code/': 0,
      'code/adaptive.yaml': 418,
      'code/const.yaml': 349,
      'code/data.py': 3254,
      'code/mnist.py': 2996,
      'code/model_def.py': 1445,
      'code/README.md': 1492,
      'code/startup-hook.sh': 59,
      'load_data.json': 2665,
      'state_dict.pth': 680041,
      'workload_sequencer.pkl': 89,
    },
    state: 'COMPLETED' as const,
    totalBatches: 58,
    uuid: 'b01dfffc-ad3e-4398-b205-b66d920d7b0b',
  },
  bestValidationMetric: {
    endTime: '2022-08-05T05:56:49.692523Z',
    metrics: { accuracy: 0.17780853807926178, val_loss: 2.141766309738159 },
    totalBatches: 58,
  },
  endTime: '2022-08-05T05:56:51.432745Z',
  experimentId: 1,
  hyperparameters: {
    global_batch_size: 64,
    hidden_size: 64,
    learning_rate: 0.08311236560675993,
  },
  id: 3,
  latestValidationMetric: {
    endTime: '2022-08-05T05:56:49.692523Z',
    metrics: { accuracy: 0.17780853807926178, val_loss: 2.141766309738159 },
    totalBatches: 58,
  },
  startTime: '2022-08-05T05:56:23.400221Z',
  state: 'COMPLETED' as const,
  totalBatchesProcessed: 58,
  totalCheckpointSize: 58,
};

const mockTrial2: TrialDetails = {
  autoRestarts: 0,
  bestAvailableCheckpoint: {
    endTime: '2022-08-05T05:56:19.781968Z',
    resources: {
      'code/': 0,
      'code/adaptive.yaml': 418,
      'code/const.yaml': 349,
      'code/data.py': 3254,
      'code/mnist.py': 2996,
      'code/model_def.py': 1445,
      'code/README.md': 1492,
      'code/startup-hook.sh': 59,
      'load_data.json': 2664,
      'state_dict.pth': 680105,
      'workload_sequencer.pkl': 89,
    },
    state: 'COMPLETED' as const,
    totalBatches: 58,
    uuid: '3b780651-0250-4307-8ec0-fac09a688970',
  },
  bestValidationMetric: {
    endTime: '2022-08-05T05:56:21.150552Z',
    metrics: { accuracy: 0.1809730976819992, val_loss: 2.1605682373046875 },
    totalBatches: 58,
  },
  endTime: '2022-08-05T05:56:49.524883Z',
  experimentId: 1,
  hyperparameters: {
    global_batch_size: 64,
    hidden_size: 64,
    learning_rate: 0.0951845051299588,
  },
  id: 2,
  latestValidationMetric: {
    endTime: '2022-08-05T05:56:21.150552Z',
    metrics: { accuracy: 0.1809730976819992, val_loss: 2.1605682373046875 },
    totalBatches: 58,
  },
  logRetentionDays: 12,
  startTime: '2022-08-05T05:55:55.152888Z',
  state: 'COMPLETED' as const,
  totalBatchesProcessed: 58,
  totalCheckpointSize: 58,
};

const mockTrial3: TrialDetails = {
  autoRestarts: 0,
  bestAvailableCheckpoint: {
    endTime: '2022-08-05T05:56:41.515152Z',
    resources: {
      'code/': 0,
      'code/adaptive.yaml': 418,
      'code/const.yaml': 349,
      'code/data.py': 3254,
      'code/mnist.py': 2996,
      'code/model_def.py': 1445,
      'code/README.md': 1492,
      'code/startup-hook.sh': 59,
      'load_data.json': 2665,
      'state_dict.pth': 680105,
      'workload_sequencer.pkl': 89,
    },
    state: 'COMPLETED' as const,
    totalBatches: 58,
    uuid: 'b3d4ed7e-625c-4083-9e6c-ffba4ac03729',
  },
  bestValidationMetric: {
    endTime: '2022-08-05T05:56:42.958782Z',
    metrics: { accuracy: 0.3433544337749481, val_loss: 1.7270305156707764 },
    totalBatches: 58,
  },
  endTime: '2022-08-05T05:56:49.345991Z',
  experimentId: 1249,
  hyperparameters: {
    global_batch_size: 64,
    hidden_size: 64,
    learning_rate: 0.06607116139078467,
  },
  id: 3567,
  latestValidationMetric: {
    endTime: '2022-08-05T05:56:42.958782Z',
    metrics: { accuracy: 0.3433544337749481, val_loss: 1.7270305156707764 },
    totalBatches: 58,
  },
  logRetentionDays: -1,
  startTime: '2022-08-05T05:56:16.181755Z',
  state: 'COMPLETED' as const,
  totalBatchesProcessed: 58,
  totalCheckpointSize: 58,
};

describe('Trial Info Box', () => {
  describe('should show log retention days box', () => {
    it('should show Log Retention box with "-" for days', async () => {
      setup(mockTrial1, mockExperiment);
      expect(await screen.findByText('Log Retention Days')).toBeVisible();
      expect(await screen.findByText('-')).toBeVisible();
    });

    it('should show Log Retention box with given number of days', async () => {
      setup(mockTrial2, mockExperiment);
      expect(await screen.findByText('Log Retention Days')).toBeVisible();
      expect(await screen.findByText(`${mockTrial2.logRetentionDays} days`)).toBeVisible();
    });

    it('should show Log Retention box with "Forever" for days', async () => {
      setup(mockTrial3, mockExperiment);
      expect(await screen.findByText('Log Retention Days')).toBeVisible();
      expect(await screen.findByText('Forever')).toBeVisible();
    });
  });

  describe('Lineage card', () => {
    it('should show Data input card with tge lineage link when pachyderm integration data is present', async () => {
      const mockExperimentWith = Object.assign(
        { ...mockExperiment },
        {
          config: { ...mockExperiment.config, integrations: { pachyderm: mockIntegrationData } },
        },
      );

      setup(mockTrial1, mockExperimentWith);
      expect(await screen.findByText('Data input')).toBeVisible();
      expect(await screen.findByText(mockIntegrationData.dataset.repo)).toBeVisible();
    });

    it('should not show Data input card when pachyderm integration is missing', async () => {
      setup(mockTrial1, mockExperiment);
      expect(await screen.queryByText('Data input')).not.toBeInTheDocument();
    });
  });
});
