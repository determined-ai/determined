import React from 'react';

import { ExperimentBase, ExperimentOld } from 'types';
import { generateOldExperiment } from 'utils/task';

import ExperimentInfoBox from './ExperimentInfoBox';

export default {
  component: ExperimentInfoBox,
  title: 'ExperimentInfoBox',
};

const sampleExperiment: ExperimentOld = generateOldExperiment(3);

const experimentDetails: ExperimentBase = {
  ...sampleExperiment,
  username: 'hamid',
};

export const state = (): React.ReactNode => (
  <ExperimentInfoBox experiment={experimentDetails} />
);
