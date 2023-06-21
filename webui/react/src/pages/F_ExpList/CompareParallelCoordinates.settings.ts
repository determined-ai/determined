import { array, keyof, string, type, undefined as undefinedType, union } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';
import { Metric, Scale } from 'types';

export interface ExperimentHyperparametersSettings {
  hParams: string[];
  metric?: Metric;
  scale: Scale;
}

export const settingsConfigForExperimentHyperparameters = (
  hParams: string[],
  projectId: number,
): SettingsConfig<ExperimentHyperparametersSettings> => ({
  settings: {
    hParams: {
      defaultValue: hParams,
      skipUrlEncoding: true,
      storageKey: 'hParams',
      type: array(string),
    },
    metric: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'metric',
      type: union([
        undefinedType,
        type({ name: string, type: keyof({ training: null, validation: null }) }),
      ]),
    },
    scale: {
      defaultValue: Scale.Linear,
      storageKey: 'scale',
      type: keyof({ linear: null, log: null }),
    },
  },
  storagePath: `experiment-compare-hyperparameters-${projectId}`,
});
