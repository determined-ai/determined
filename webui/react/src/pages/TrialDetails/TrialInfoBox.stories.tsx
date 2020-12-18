import React from 'react';

import {
  CheckpointState, ExperimentBase, ExperimentOld, RunState, TrialDetails2, TrialItem2,
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

const experimentDetails: ExperimentBase = {
  ...sampleExperiment,
  username: 'hamid',
};

export const state = (): React.ReactNode => (
  <TrialInfoBox experiment={experimentDetails} trial={trialDetails} />
);
