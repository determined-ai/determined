import React from 'react';

import {
  CheckpointState, ExperimentBase, ExperimentOld, RunState, TrialDetails, TrialItem,
} from 'types';
import { generateOldExperiment } from 'utils/task';

import TrialInfoBox from './TrialInfoBox';

export default {
  component: TrialInfoBox,
  title: 'TrialInfoBox',
};

const sampleExperiment: ExperimentOld = generateOldExperiment(3);

const sampleTrialItem: TrialItem = {
  autoRestarts: 0,
  bestAvailableCheckpoint: {
    resources: { noOpCheckpoint: 1542 },
    state: CheckpointState.Completed,
    totalBatches: 10000,
  },
  experimentId: 1,
  hyperparameters: {
    boolVal: false,
    floatVale: 3.5,
    intVal: 3,
    objectVal: { paramA: 3, paramB: 'str' },
    stringVal: 'loss',
  },
  id: 1,
  startTime: Date.now.toString(),
  state: RunState.Completed,
  totalBatchesProcessed: 10000,
};

const trialDetails: TrialDetails = {
  ...sampleTrialItem,
  totalCheckpointSize: 0,
};

const experimentDetails: ExperimentBase = {
  parentArchived: false,
  projectName: 'Uncategorized',
  projectOwnerId: 1,
  workspaceId: 1,
  workspaceName: 'Uncategorized',
  ...sampleExperiment,
  originalConfig: `
  entrypoint: model_def:MNistTrial
  hyperparameters:
    dropout1: {maxval: 0.8, minval: 0.2, type: double}
    dropout2: {maxval: 0.8, minval: 0.2, type: double}
    global_batch_size: 64
    learning_rate: {maxval: 1.0, minval: 0.0001, type: double}
    n_filters1: {maxval: 64, minval: 8, type: int}
    n_filters2: {maxval: 72, minval: 8, type: int}
  name: mnist_pytorch_adaptive_search
  records_per_epoch: 10
  searcher:
    max_length: {batches: 937}
    max_trials: 16
    metric: validation_loss
    name: adaptive_asha
    smaller_is_better: true`,
  userId: 345,
};

export const state = (): React.ReactNode => (
  <TrialInfoBox experiment={experimentDetails} trial={trialDetails} />
);
