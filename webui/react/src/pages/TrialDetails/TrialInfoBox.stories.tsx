import React from 'react';

import {
  CheckpointState, ExperimentDetails, ExperimentOld, RunState, TrialDetails2, TrialItem, TrialItem2,
} from 'types';
import { generateOldExperiment } from 'utils/task';

import TrialInfoBox from './TrialInfoBox';

export default {
  component: TrialInfoBox,
  title: 'TrialInfoBox',
};

const sampleExperiment: ExperimentOld = generateOldExperiment(3);

const sampleTrialItem: TrialItem2 = {
  bestAvailableCheckpoint: {
    numBatches: 100,
    priorBatchesProcessed: 9900,
    resources: { noOpCheckpoint: 1542 },
    startTime: Date.now.toString(),
    state: CheckpointState.Completed,

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
  startTime: Date.now.toString(),
  state: RunState.Completed,
  totalBatchesProcessed: 10000,
};

const trialDetails: TrialDetails2 = {
  ...sampleTrialItem,
  workloads: [],
};

const experimentDetails: ExperimentDetails = {
  ...sampleExperiment,
  trials: [
    // sampleTrialItem, // TODO experiment migration
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
