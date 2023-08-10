import { array, boolean, literal, number, partial, record, string, TypeOf, union } from 'io-ts';

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
  excludedExperiments: number[];
  filterset: string; // save FilterFormSet as string
  sortString: string;
  pageLimit: number;
  pinnedColumnsCount: number;
  heatmapSkipped: string[];
  heatmapOn: boolean;
  selectAll: boolean;
  selectedExperiments: Array<number>;
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
    excludedExperiments: {
      defaultValue: [],
      skipUrlEncoding: true,
      storageKey: 'excludedExperiments',
      type: array(number),
    },
    filterset: {
      defaultValue: JSON.stringify(INIT_FORMSET),
      skipUrlEncoding: true,
      storageKey: 'filterset',
      type: string,
    },
    heatmapOn: {
      defaultValue: false,
      skipUrlEncoding: true,
      storageKey: 'heatmapOn',
      type: boolean,
    },
    heatmapSkipped: {
      defaultValue: [],
      skipUrlEncoding: true,
      storageKey: 'heatmapSkipped',
      type: array(string),
    },
    pageLimit: {
      defaultValue: 20,
      skipUrlEncoding: true,
      storageKey: 'pageLimit',
      type: number,
    },
    pinnedColumnsCount: {
      defaultValue: 3,
      skipUrlEncoding: true,
      storageKey: 'pinnedColumnsCount',
      type: number,
    },
    selectAll: {
      defaultValue: false,
      skipUrlEncoding: true,
      storageKey: 'selectAll',
      type: boolean,
    },
    selectedExperiments: {
      defaultValue: [],
      skipUrlEncoding: true,
      storageKey: 'selectedExperiments',
      type: array(number),
    },
    sortString: {
      defaultValue: 'id=desc',
      skipUrlEncoding: true,
      storageKey: 'sortString',
      type: string,
    },
  },
  storagePath: `experimentListingForProject${id}`,
});

export interface F_ExperimentListGlobalSettings {
  expListView: ExpListView;
  rowHeight: RowHeight;
}

const ioExpListView = union([literal('scroll'), literal('paged')]);

export const experimentListGlobalSettingsConfig = partial({
  expListView: ioExpListView,
  rowHeight: ioRowHeight,
});

export const experimentListGlobalSettingsDefaults = {
  expListView: 'scroll',
  rowHeight: RowHeight.MEDIUM,
} as const;

export const experimentListGlobalSettingsPath = 'globalTableSettings';

export const settingsConfigGlobal: SettingsConfig<F_ExperimentListGlobalSettings> = {
  settings: {
    expListView: {
      defaultValue: 'scroll',
      skipUrlEncoding: true,
      storageKey: 'expListView',
      type: ioExpListView,
    },
    rowHeight: {
      defaultValue: RowHeight.MEDIUM,
      skipUrlEncoding: true,
      storageKey: 'rowHeight',
      type: ioRowHeight,
    },
  },
  storagePath: experimentListGlobalSettingsPath,
};
