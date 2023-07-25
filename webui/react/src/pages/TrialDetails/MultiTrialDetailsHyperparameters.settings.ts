import { array, keyof, number, string, type, undefined as undefinedType, union } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';
import { Metric, Scale } from 'types';

const DEFAULT_BATCH = 0;
const DEFAULT_BATCH_MARGIN = 10;
const DEFAULT_MAX_TRIALS = 100;

export interface ExperimentHyperparametersSettings {
  batch: number;
  batchMargin: number;
  hParams: string[];
  maxTrial: number;
  metric?: Metric;
  scale: Scale;
}

export const settingsConfigForExperimentHyperparameters = (
  experimentId: number,
  trialId: number,
  hParams: string[],
): SettingsConfig<ExperimentHyperparametersSettings> => ({
  settings: {
    batch: {
      defaultValue: DEFAULT_BATCH,
      storageKey: 'batch',
      type: number,
    },
    batchMargin: {
      defaultValue: DEFAULT_BATCH_MARGIN,
      storageKey: 'batchMargin',
      type: number,
    },
    hParams: {
      defaultValue: hParams,
      skipUrlEncoding: true,
      storageKey: 'hParams',
      type: array(string),
    },
    maxTrial: {
      defaultValue: DEFAULT_MAX_TRIALS,
      storageKey: 'maxTrial',
      type: number,
    },
    metric: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'metric',
      type: union([undefinedType, type({ name: string, type: string })]),
    },
    scale: {
      defaultValue: Scale.Linear,
      storageKey: 'scale',
      type: keyof({ linear: null, log: null }),
      // See https://github.com/gcanti/io-ts/blob/master/index.md#union-of-string-literals
    },
  },
  storagePath: `experiment-hyperparameters-${experimentId}-${trialId}`,
});
