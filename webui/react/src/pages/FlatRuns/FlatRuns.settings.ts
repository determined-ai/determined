import { array, boolean, number, partial, record, string } from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { SettingsConfig } from 'hooks/useSettings';

//import { defaultColumnWidths, defaultExperimentColumns } from './expListColumns';
//import { ioRowHeight, ioTableViewMode, RowHeight, TableViewMode } from './OptionsMenu';

export interface FlatRunsSettings {
  columns: string[];
  columnWidths: Record<string, number>;
  compare: boolean;
  excludedRuns: number[];
  filterset: string; // save FilterFormSet as string
  sortString: string;
  pageLimit: number;
  pinnedColumnsCount: number;
  heatmapSkipped: string[];
  heatmapOn: boolean;
  selectAll: boolean;
  selectedRuns: Array<number>;
}
export const settingsConfigForProject = (id: number): SettingsConfig<FlatRunsSettings> => ({
  settings: {
    columns: {
      defaultValue: defaultRunColumns,
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
    excludedRuns: {
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
    selectedRuns: {
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
  storagePath: `flatRunsForProject${id}`,
});

export interface FlatRunsGlobalSettings {
  rowHeight: RowHeight;
  tableViewMode: TableViewMode;
}

export const flatRunsGlobalSettingsConfig = partial({
  rowHeight: ioRowHeight,
  tableViewMode: ioTableViewMode,
});

export const flatRunsGlobalSettingsDefaults = {
  rowHeight: RowHeight.MEDIUM,
  tableViewMode: 'scroll',
} as const;

export const flatRunsGlobalSettingsPath = 'globalTableSettings';

export const settingsConfigGlobal: SettingsConfig<FlatRunsGlobalSettings> = {
  settings: {
    rowHeight: {
      defaultValue: RowHeight.MEDIUM,
      skipUrlEncoding: true,
      storageKey: 'rowHeight',
      type: ioRowHeight,
    },
    tableViewMode: {
      defaultValue: 'scroll',
      skipUrlEncoding: true,
      storageKey: 'tableViewMode',
      type: ioTableViewMode,
    },
  },
  storagePath: flatRunsGlobalSettingsPath,
};
