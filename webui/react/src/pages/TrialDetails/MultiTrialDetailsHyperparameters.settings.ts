import { array, literal, number, string, type, undefined as undefinedType, union } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';
import {
  ViewType,
  VisualizationFilters,
} from 'pages/ExperimentDetails/ExperimentVisualization/ExperimentVisualizationFilters';
import { Scale } from 'types';

const DEFAULT_BATCH = 0;
const DEFAULT_BATCH_MARGIN = 10;
const DEFAULT_MAX_TRIALS = 100;

export interface ExperimentHyperparametersSettings {
  filters: VisualizationFilters;
}

const defaultFilters: VisualizationFilters = {
  batch: DEFAULT_BATCH,
  batchMargin: DEFAULT_BATCH_MARGIN,
  hParams: [],
  maxTrial: DEFAULT_MAX_TRIALS,
  metric: undefined,
  scale: Scale.Linear,
  view: ViewType.Grid,
};

export const settingsConfigForExperimentHyperparameters = (
  id: number,
): SettingsConfig<ExperimentHyperparametersSettings> => ({
  settings: {
    filters: {
      defaultValue: defaultFilters,
      skipUrlEncoding: true,
      storageKey: 'filters',
      type: type({
        batch: number,
        batchMargin: number,
        hParams: array(string),
        maxTrial: number,
        metric: union([
          undefinedType,
          type({ name: string, type: union([literal('training'), literal('validation')]) }),
        ]),
        scale: union([literal('linear'), literal('log')]),
        view: union([literal('grid'), literal('list')]),
      }),
    },
  },
  storagePath: `experiment-hyperparameters-${id}`,
});
