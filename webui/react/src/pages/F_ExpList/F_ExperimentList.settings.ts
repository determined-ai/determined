import { array, string } from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';

import { defaultExperimentColumns } from './glide-table/columns';

export interface F_ExperimentListSettings {
  columns: string[];
}
export const settingsConfigForProject = (id: number): SettingsConfig<F_ExperimentListSettings> => ({
  settings: {
    columns: {
      defaultValue: defaultExperimentColumns,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(string),
    },
  },
  storagePath: `f_project-details-${id}`,
});
