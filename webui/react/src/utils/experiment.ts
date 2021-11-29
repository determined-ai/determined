import * as Type from 'types';

import { deletePathList, getPathList, isNumber, setPathList, unflattenObject } from './data';

export const isSingleTrialExperiment = (experiment: Type.ExperimentBase): boolean => {
  return experiment?.config.searcher.name === Type.ExperimentSearcherName.Single
      || experiment?.config.searcher.max_trials === 1;
};

export const trialHParamsToExperimentHParams = (
  trialHParams: Type.TrialHyperparameters,
): Type.Hyperparameters => {
  const hParams = Object.keys(trialHParams).reduce((acc, key) => {
    return {
      ...acc,
      [key]: {
        type: Type.HyperparameterType.Constant,
        val: trialHParams[key] as number,
      },
    };
  }, {} as Record<keyof Type.TrialHyperparameters, unknown>);
  return unflattenObject(hParams) as Type.Hyperparameters;
};

/* Experiment Config */

const stepRemovalTranslations = [
  { newName: 'searcher.max_length', oldName: 'searcher.max_steps' },
  { oldName: 'min_validation_period' },
  { oldName: 'min_checkpoint_period' },
  { newName: 'searcher.max_length', oldName: 'searcher.target_trial_steps' },
  { newName: 'searcher.length_per_round', oldName: 'searcher.steps_per_round' },
  { newName: 'searcher.budget', oldName: 'searcher.step_budget' },
];

const getLengthFromStepCount = (
  config: Type.RawJson,
  stepCount: number,
): [string, number] => {
  const DEFAULT_BATCHES_PER_STEP = 100;
  // provide backward compat for step count
  const batchesPerStep = config.batches_per_step || DEFAULT_BATCHES_PER_STEP;
  return [ 'batches', stepCount * batchesPerStep ];
};

// Add opportunistic backward compatibility to old configs.
export const upgradeConfig = (config: Type.RawJson): void => {
  stepRemovalTranslations.forEach(translation => {
    const oldPath = translation.oldName.split('.');
    const curValue = getPathList<undefined | null | number | unknown>(config, oldPath);
    if (curValue === undefined) return;
    if (curValue === null) {
      deletePathList(config, oldPath);
    }
    if (isNumber(curValue)) {
      const [ key, count ] = getLengthFromStepCount(config, curValue);
      const newPath = (translation.newName || translation.oldName).split('.');
      setPathList(config, newPath, { [key]: count });
      if (translation.newName) deletePathList(config, oldPath);
    }
  });

  delete config.batches_per_step;
};
