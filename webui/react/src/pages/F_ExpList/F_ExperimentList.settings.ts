import * as t from 'io-ts';
import { literal, union } from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { SettingsConfig } from 'hooks/useSettings';

import { defaultExperimentColumns } from './glide-table/columns';

export type ExpListView = 'scroll' | 'paged';
export interface F_ExperimentListSettings {
  columns: string[];
  filterset: string; // save FilterFormSet as string
  pageLimit: number;
}
export const settingsConfigForProject = (id: number): SettingsConfig<F_ExperimentListSettings> => ({
  settings: {
    columns: {
      defaultValue: defaultExperimentColumns,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: t.array(t.string),
    },
    filterset: {
      defaultValue: JSON.stringify(INIT_FORMSET),
      skipUrlEncoding: true,
      storageKey: 'filterset',
      type: t.string,
    },
    pageLimit: {
      defaultValue: 20,
      skipUrlEncoding: true,
      storageKey: 'pageLimit',
      type: t.number,
    },
  },
  storagePath: `f_project-details-${id}`,
});

export interface F_ExperimentListGlobalSettings {
  expListView: ExpListView;
}

export const settingsConfigGlobal: SettingsConfig<F_ExperimentListGlobalSettings> = {
  settings: {
    expListView: {
      defaultValue: 'scroll',
      skipUrlEncoding: true,
      storageKey: 'expListView',
      type: union([literal('scroll'), literal('paged')]),
    },
  },
  storagePath: 'f_project-details-global',
};
