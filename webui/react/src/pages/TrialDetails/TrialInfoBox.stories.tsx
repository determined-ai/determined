import React from 'react';

import {
  CheckpointState, ExperimentDetails, ExperimentOld, RunState, TrialDetails, TrialItem,
} from 'types';
import { generateOldExperiment } from 'utils/task';

import TrialInfoBox from './TrialInfoBox';

export default {
  component: TrialInfoBox,
  title: 'TrialInfoBox',
};

const sampleExperiment: ExperimentOld = generateOldExperiment(3);

const sampleTrialItem: TrialItem = {
  bestAvailableCheckpoint: {
    id: 3,
    resources: { noOpCheckpoint: 1542 },
    startTime: Date.now.toString(),
    state: CheckpointState.Completed,
    stepId: 34,
    trialId: 3,
    validationMetric: 0.023,
  },
  experimentId: 1,
  hparams: {
    boolVal: false,
    floatVale: 3.5,
    intVal: 3,
    objectVal: { paramA: 3, paramB: 'str' },
    stringVal: 'loss',
  },
  id: 1,
  numCompletedCheckpoints: 1,
  numSteps: 100,
  seed: 142,
  startTime: Date.now.toString(),
  state: RunState.Completed,
  totalBatchesProcessed: 10000,
  url: '/det/trials/1',
};

const trialDetails: TrialDetails = {
  ...sampleTrialItem,
  steps: [],
};

const experimentDetails: ExperimentDetails = {
  ...sampleExperiment,
  trials: [
    sampleTrialItem,
  ],
  username: 'hamid',
  validationHistory: [ {
    endTime: Date.now().toString(),
    trialId: 0,
    validationError: 0.023,
  } ],
};

export const state = (): React.ReactNode => (
  <TrialInfoBox experiment={experimentDetails} trial={trialDetails} />
);
