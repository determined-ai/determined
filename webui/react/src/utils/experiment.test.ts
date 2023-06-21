import { RawJson } from 'types';
import * as Type from 'types';

import * as utils from './experiment';

describe('Experiment Utilities', () => {
  describe('isExperiment', () => {
    it('should validate experiment tasks', () => {
      const experimentTask: Type.ExperimentItem = {
        archived: false,
        config: {} as Type.ExperimentConfig,
        configRaw: {},
        hyperparameters: {},
        id: 123,
        jobId: '',
        labels: [],
        name: 'ResNet-50',
        numTrials: 1,
        projectId: 0,
        resourcePool: 'gpu-pool',
        searcherType: 'single',
        startTime: '2021-11-29T00:00:00Z',
        state: Type.RunState.Active,
        userId: 345,
      };
      expect(utils.isExperiment(experimentTask)).toBe(true);
    });

    it('should invalidate non-experiment tasks', () => {
      const commandTask = {
        id: 'kenzo',
        name: 'Count Active Processed',
        resourcePool: 'cpu-pool',
        startTime: '2021-11-29T00:00:00Z',
        state: Type.CommandState.Queued,
        type: Type.CommandType.Command,
        userId: 345,
        workspaceId: 0,
      };
      expect(utils.isExperiment(commandTask)).toBe(false);
    });
  });

  describe('isSingleTrialExperiment', () => {
    const tests = [
      {
        input: { config: { searcher: { name: Type.ExperimentSearcherName.Single } } },
        output: true,
      },
      {
        input: { config: { searcher: { name: Type.ExperimentSearcherName.Random } } },
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
      tests.forEach((test) => {
        const result = utils.isSingleTrialExperiment(test.input as Type.ExperimentBase);
        expect(result).toStrictEqual(test.output);
      });
    });
  });

  describe('trialHParamsToExperimentHParams', () => {
    const tests: { input: Type.TrialHyperparameters; output: RawJson }[] = [
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
      tests.forEach((test) => {
        const result = utils.trialHParamsToExperimentHParams(test.input);
        expect(result).toStrictEqual(test.output);
      });
    });
  });

  describe('upgradeConfig', () => {
    it('should upgrade old config properties to new config properties', () => {
      const tests = [
        {
          input: { min_checkpoint_period: 32 },
          output: { min_checkpoint_period: { batches: 3200 } },
        },
        {
          input: { min_validation_period: 32 },
          output: { min_validation_period: { batches: 3200 } },
        },
        {
          input: { searcher: { max_steps: 10 } },
          output: { searcher: { max_length: { batches: 1000 } } },
        },
        {
          input: { searcher: { step_budget: 100 } },
          output: { searcher: { budget: { batches: 10000 } } },
        },
        {
          input: { searcher: { steps_per_round: 2 } },
          output: { searcher: { length_per_round: { batches: 200 } } },
        },
        {
          input: { searcher: { target_trial_steps: 10 } },
          output: { searcher: { max_length: { batches: 1000 } } },
        },
      ];
      tests.forEach((test) => {
        expect(utils.upgradeConfig(test.input)).toStrictEqual(test.output);
      });
    });

    it('should remove old config properties with null values', () => {
      const oldConfig = { min_checkpoint_period: null };
      const newConfig = {};
      expect(utils.upgradeConfig(oldConfig)).toStrictEqual(newConfig);
    });
  });
});
