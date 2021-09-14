import {
  ExperimentBase, ExperimentSearcherName, TrialHyperparameters, UnknownRecord,
} from 'types';

import {
  isSingleTrialExperiment, trialHParamsToExperimentHParams,
} from './experiment';

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
