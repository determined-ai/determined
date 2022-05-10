import * as Type from 'types';

import { RawJson } from '../shared/types';
import { clone, deletePathList, getPathList, isNumber, setPathList,
  unflattenObject } from '../shared/utils/data';

// Differentiate Experiment from Task.
export const isExperiment = (
  obj: Type.AnyTask | Type.ExperimentItem,
): obj is Type.ExperimentItem => {
  return 'config' in obj && 'archived' in obj;
};

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
  { oldName: 'min_checkpoint_period' },
  { oldName: 'min_validation_period' },
  { newName: 'searcher.budget', oldName: 'searcher.step_budget' },
  { newName: 'searcher.length_per_round', oldName: 'searcher.steps_per_round' },
  { newName: 'searcher.max_length', oldName: 'searcher.max_steps' },
  { newName: 'searcher.max_length', oldName: 'searcher.target_trial_steps' },
];

const getLengthFromStepCount = (
  config: RawJson,
  stepCount: number,
): [string, number] => {
  const DEFAULT_BATCHES_PER_STEP = 100;
  // provide backward compat for step count
  const batchesPerStep = config.batches_per_step || DEFAULT_BATCHES_PER_STEP;
  return [ 'batches', stepCount * batchesPerStep ];
};

// Add opportunistic backward compatibility to old configs.
export const upgradeConfig = (config: RawJson): RawJson => {
  const newConfig = clone(config);

  stepRemovalTranslations.forEach(translation => {
    const oldPath = translation.oldName.split('.');
    const curValue = getPathList<undefined | null | number | unknown>(newConfig, oldPath);
    if (curValue === undefined) return;
    if (curValue === null) deletePathList(newConfig, oldPath);
    if (isNumber(curValue)) {
      const [ key, count ] = getLengthFromStepCount(newConfig, curValue);
      const newPath = (translation.newName || translation.oldName).split('.');
      setPathList(newConfig, newPath, { [key]: count });

      if (translation.newName) deletePathList(newConfig, oldPath);
    }
  });

  delete newConfig.batches_per_step;

  return newConfig;
};
