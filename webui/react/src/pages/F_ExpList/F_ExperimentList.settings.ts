import { array, boolean, keyof, literal, number, string, TypeOf, union } from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { SettingsConfig } from 'hooks/useSettings';

import { defaultExperimentColumns } from './glide-table/columns';

export type ExpListView = 'scroll' | 'paged';
export const RowHeight = {
  COMFY: 'COMFY',
  COMPACT: 'COMPACT',
  DEFAULT: 'DEFAULT',
} as const;
const ioRowHeight = keyof({
  [RowHeight.COMFY]: null,
  [RowHeight.COMPACT]: null,
  [RowHeight.DEFAULT]: null,
});
export type RowHeight = TypeOf<typeof ioRowHeight>;

export interface F_ExperimentListSettings {
  columns: string[];
  compare: boolean;
  compareWidth: number;
  filterset: string; // save FilterFormSet as string
  pageLimit: number;
  rowHeight: RowHeight;
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
    pageLimit: {
      defaultValue: 20,
      skipUrlEncoding: true,
      storageKey: 'pageLimit',
      type: number,
    },
    rowHeight: {
      defaultValue: RowHeight.DEFAULT,
      skipUrlEncoding: true,
      storageKey: 'rowHeight',
      type: ioRowHeight,
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
