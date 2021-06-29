import React from 'react';

import {
  ExperimentBase, ExperimentOld, RunState, TrialDetails, TrialItem,
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
    resources: { noOpCheckpoint: 1542 },
    startTime: Date.now.toString(),
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
  workloads: [],
};

const experimentDetails: ExperimentBase = {
  ...sampleExperiment,
  username: 'hamid',
};

export const state = (): React.ReactNode => (
  <TrialInfoBox experiment={experimentDetails} trial={trialDetails} />
);
