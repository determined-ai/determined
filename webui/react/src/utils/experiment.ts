import { ExperimentBase, ExperimentSearcherName } from 'types';

export const isSingleTrialExperiment = (experiment: ExperimentBase): boolean => {
  return experiment?.config.searcher.name === ExperimentSearcherName.Single
        || experiment?.config.searcher.max_trials === 1;
};
