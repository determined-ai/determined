import {
  ExperimentBase, ExperimentSearcherName, Hyperparameters, HyperparameterType, TrialHyperparameters,
} from 'types';

import { unflattenObject } from './data';

export const isSingleTrialExperiment = (experiment: ExperimentBase): boolean => {
  return experiment?.config.searcher.name === ExperimentSearcherName.Single
      || experiment?.config.searcher.max_trials === 1;
};

export const trialHParamsToExperimentHParams = (
  trialHParams: TrialHyperparameters,
): Hyperparameters => {
  const hParams = Object.keys(trialHParams).reduce((acc, key) => {
    return {
      ...acc,
      [key]: {
        type: HyperparameterType.Constant,
        val: trialHParams[key] as number,
      },
    };
  }, {} as Record<keyof TrialHyperparameters, unknown>);
  return unflattenObject(hParams) as Hyperparameters;
};
