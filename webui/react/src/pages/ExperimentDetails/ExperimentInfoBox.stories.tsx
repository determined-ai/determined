import React from 'react';

import { CheckpointState, ExperimentDetails, ExperimentOld, RunState } from 'types';
import { generateOldExperiment } from 'utils/task';

import ExperimentInfoBox from './ExperimentInfoBox';

export default {
  component: ExperimentInfoBox,
  title: 'ExperimentInfoBox',
};

const sampleExperiment: ExperimentOld = generateOldExperiment(3);

const experimentDetails: ExperimentDetails = {
  ...sampleExperiment,
  trials: [
    {
      bestAvailableCheckpoint: {
        resources: { noOpCheckpoint: 1542 },
        startTime: Date.now.toString(),
        state: CheckpointState.Completed,
        trialId: 3,
        validationMetric: 0.023,
      },
      experimentId: 1,
      hparams: {},
      id: 1,
      numCompletedCheckpoints: 1,
      numSteps: 100,
      seed: 142,
      startTime: Date.now.toString(),
      state: RunState.Completed,
      totalBatchesProcessed: 10000,
      url: '/experiments/1/trials/1',
    },
  ],
  username: 'hamid',
  validationHistory: [ {
    endTime: Date.now().toString(),
    trialId: 0,
    validationError: 0.023,
  } ],
};

// export const state = (): React.ReactNode => (
//   // <ExperimentInfoBox experiment={experimentDetails} />
// );
