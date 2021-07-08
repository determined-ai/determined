import {
  ExperimentBase, ExperimentSearcherName, Hyperparameter,
  Hyperparameters, HyperparametersFlattened,
} from 'types';

export const flattenHyperParams = (
  hyperparams: Hyperparameters,
  keys: string[] = [],
): HyperparametersFlattened => {
  return Object.keys(hyperparams).reduce((acc, key) => {
    const hp = hyperparams[key];
    const keyPath = [ ...keys, key ].join('.');
    if (hp.type) {
      acc[keyPath] = hp as Hyperparameter;
    } else {
      acc = { ...acc, ...flattenHyperParams(hp as Hyperparameters, [ ...keys, key ]) };
    }
    return acc;
  }, {} as HyperparametersFlattened);
};

export const isSingleTrialExperiment = (experiment: ExperimentBase): boolean => {
  return experiment?.config.searcher.name === ExperimentSearcherName.Single
        || experiment?.config.searcher.max_trials === 1;
};
