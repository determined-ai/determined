import {
  ExperimentBase, ExperimentSearcherName, Hyperparameters, TrialHyperparameters, UnknownRecord,
} from 'types';

import {
  flattenHyperparameters, isSingleTrialExperiment, trialHParamsToExperimentHParams,
} from './experiment';

describe('flattenHyperparameters', () => {
  const tests: UnknownRecord[] = [
    {
      input: {
        arch: {
          n_filters1: { maxval: 64, minval: 8, type: 'int' },
          n_filters2: { maxval: 72, minval: 8, type: 'int' },
        },
        dropout1: { maxval: 0.8, minval: 0.2, type: 'double' },
        dropout2: { maxval: 0.8, minval: 0.2, type: 'double' },
        global_batch_size: { type: 'const', val: 64 },
        learning_rate: { maxval: 1, minval: 0.0001, type: 'double' },
      },
      output: {
        'arch.n_filters1': { maxval: 64, minval: 8, type: 'int' },
        'arch.n_filters2': { maxval: 72, minval: 8, type: 'int' },
        'dropout1': { maxval: 0.8, minval: 0.2, type: 'double' },
        'dropout2': { maxval: 0.8, minval: 0.2, type: 'double' },
        'global_batch_size': { type: 'const', val: 64 },
        'learning_rate': { maxval: 1, minval: 0.0001, type: 'double' },
      },
    },
    {
      input: { a: { b: { c: { type: 'const', val: 5 } } } },
      output: { 'a.b.c': { type: 'const', val: 5 } },
    },
  ];
  it('should flatten hyperparameter config', () => {
    tests.forEach(test => {
      expect(flattenHyperparameters(test.input as Hyperparameters)).toStrictEqual(test.output);
    });
  });
});

describe('isSingleTrialExperiment', () => {
  const tests = [
    {
      input: { config: { searcher: { name: ExperimentSearcherName.Single } } },
      output: true,
    },
    {
      input: { config: { searcher: { name: ExperimentSearcherName.Random } } },
      output: false,
    },
    {
      input: { config: { searcher: { max_trials: 1 } } },
      output: true,
    },
    {
      input: { config: { searcher: { max_trials: 10 } } },
      output: false,
    },
  ];
  it('should detect single trial experiment from config', () => {
    tests.forEach(test => {
      expect(isSingleTrialExperiment(test.input as ExperimentBase)).toStrictEqual(test.output);
    });
  });
});

describe('trialHParamsToExperimentHParams', () => {
  const tests: UnknownRecord[] = [
    {
      input: {
        'arch.n_filters1': 62,
        'arch.n_filters2': 35,
        'dropout1': 0.5532344404035913,
        'dropout2': 0.762201718697237,
        'global_batch_size': 64,
        'learning_rate': 0.016050683206198273,
      },
      output: {
        arch: {
          n_filters1: { type: 'const', val: 62 },
          n_filters2: { type: 'const', val: 35 },
        },
        dropout1: { type: 'const', val: 0.5532344404035913 },
        dropout2: { type: 'const', val: 0.762201718697237 },
        global_batch_size: { type: 'const', val: 64 },
        learning_rate: { type: 'const', val: 0.016050683206198273 },
      },
    },
    {
      input: { 'a.b.c': 5 },
      output: { a: { b: { c: { type: 'const', val: 5 } } } },
    },
  ];
  it('should convert trial hyperparameters to experiment config hyperparameters', () => {
    tests.forEach(test => {
      expect(trialHParamsToExperimentHParams(test.input as TrialHyperparameters))
        .toStrictEqual(test.output);
    });
  });
});
