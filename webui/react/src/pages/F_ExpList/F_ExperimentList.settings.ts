import { array, boolean, literal, number, record, string, TypeOf, union } from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { SettingsConfig } from 'hooks/useSettings';
import { valueof } from 'ioTypes';

import { defaultColumnWidths, defaultExperimentColumns } from './glide-table/columns';

export type ExpListView = 'scroll' | 'paged';
export const RowHeight = {
  EXTRA_TALL: 'EXTRA_TALL',
  MEDIUM: 'MEDIUM',
  SHORT: 'SHORT',
  TALL: 'TALL',
} as const;
const ioRowHeight = valueof(RowHeight);
export type RowHeight = TypeOf<typeof ioRowHeight>;

export interface F_ExperimentListSettings {
  columns: string[];
  columnWidths: Record<string, number>;
  compare: boolean;
  filterset: string; // save FilterFormSet as string
  sortString: string;
  pageLimit: number;
  rowHeight: RowHeight;
  pinnedColumnsCount: number;
}
export const settingsConfigForProject = (id: number): SettingsConfig<F_ExperimentListSettings> => ({
  settings: {
    columns: {
      defaultValue: defaultExperimentColumns,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(string),
    },
    columnWidths: {
      defaultValue: defaultColumnWidths,
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: record(string, number),
    },
    compare: {
      defaultValue: false,
      storageKey: 'compare',
      type: boolean,
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
    pinnedColumnsCount: {
      defaultValue: 0,
      skipUrlEncoding: true,
      storageKey: 'pinnedColumnsCount',
      type: number,
    },
    rowHeight: {
      defaultValue: RowHeight.MEDIUM,
      skipUrlEncoding: true,
      storageKey: 'rowHeight',
      type: ioRowHeight,
    },
    sortString: {
      defaultValue: '',
      skipUrlEncoding: true,
      storageKey: 'sortString',
      type: string,
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
