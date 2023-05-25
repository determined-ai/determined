import { array, boolean, number, string } from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { SettingsConfig } from 'hooks/useSettings';

import { defaultExperimentColumns } from './glide-table/columns';

export interface F_ExperimentListSettings {
  columns: string[];
  compare: boolean;
  compareWidth: number;
  filterset: string; // save FilterFormSet as string
}
export const settingsConfigForProject = (id: number): SettingsConfig<F_ExperimentListSettings> => ({
  settings: {
    columns: {
      defaultValue: defaultExperimentColumns,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(string),
    },
    compare: {
      defaultValue: false,
      storageKey: 'compare',
      type: boolean,
    },
    compareWidth: {
      defaultValue: 340,
      skipUrlEncoding: true,
      storageKey: 'compareWidth',
      type: number,
    },
    filterset: {
      defaultValue: JSON.stringify(INIT_FORMSET),
      skipUrlEncoding: true,
      storageKey: 'filterset',
      type: string,
    },
  },
  storagePath: `f_project-details-${id}`,
});
