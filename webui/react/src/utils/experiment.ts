import {
  ExperimentBase, ExperimentSearcherName, Hyperparameter,
  HyperparameterBase,
  Hyperparameters, HyperparametersFlattened, HyperparameterType, TrialHyperparameters,
} from 'types';

export const flattenHyperparameters = (
  hyperparams: Hyperparameters,
  keys: string[] = [],
): HyperparametersFlattened => {
  return Object.keys(hyperparams).reduce((acc, key) => {
    const hp = hyperparams[key];
    const keyPath = [ ...keys, key ].join('.');
    if (hp.type) {
      acc[keyPath] = hp as Hyperparameter;
    } else {
      acc = { ...acc, ...flattenHyperparameters(hp as Hyperparameters, [ ...keys, key ]) };
    }
    return acc;
  }, {} as HyperparametersFlattened);
};

export const isSingleTrialExperiment = (experiment: ExperimentBase): boolean => {
  return experiment?.config.searcher.name === ExperimentSearcherName.Single
        || experiment?.config.searcher.max_trials === 1;
};

export const trialHParamsToExperimentHParams = (
  trialHParams: TrialHyperparameters,
): Hyperparameters => {
  const experimentHParams: Hyperparameters = {};
  Object.entries(trialHParams).forEach(([ paramPath, value ]) => {
    let key = paramPath;
    let matches = key.match(/^([^.]+)\.(.+)$/);
    let pathRef: Hyperparameters | HyperparameterBase = experimentHParams;
    while (matches?.length === 3) {
      const prefix = matches[1];
      key = matches[2];
      (pathRef as Hyperparameters)[prefix] = (pathRef as Hyperparameters)[prefix] ?? {};
      pathRef = (pathRef as Hyperparameters)[prefix];
      matches = key.match(/^([^.]+)\.(.+)$/);
    }
    (pathRef as Hyperparameters)[key] = {
      type: HyperparameterType.Constant,
      val: value as number,
    };
  });
  return experimentHParams;
};
