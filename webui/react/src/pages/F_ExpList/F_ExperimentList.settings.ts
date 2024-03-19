import * as t from 'io-ts';

import { INIT_FORMSET } from 'components/FilterForm/components/FilterFormStore';
import { SettingsConfig } from 'hooks/useSettings';

import { defaultColumnWidths, defaultExperimentColumns } from './glide-table/columns';
import { ioTableViewMode, TableViewMode } from './glide-table/GlideTable';
import { ioRowHeight, RowHeight } from './glide-table/OptionsMenu';

const SelectAllType = t.type({
  exclusions: t.array(t.number),
  type: t.literal('ALL_EXCEPT'),
});

const RegularSelectionType = t.type({
  selections: t.array(t.number),
  type: t.literal('ONLY_IN'),
});

export const SelectionType = t.union([RegularSelectionType, SelectAllType]);
export type SelectionType = t.TypeOf<typeof SelectionType>;
export const DEFAULT_SELECTION: t.TypeOf<typeof RegularSelectionType> = {
  selections: [],
  type: 'ONLY_IN',
};

// have to intersect with an empty object bc of settings store type issue
export const ProjectSettings = t.intersection([
  t.type({}),
  t.partial({
    columns: t.array(t.string),
    columnWidths: t.record(t.string, t.number),
    compare: t.boolean,
    filterset: t.string, // save FilterFormSet as string
    heatmapOn: t.boolean,
    heatmapSkipped: t.array(t.string),
    pageLimit: t.number,
    pinnedColumnsCount: t.number,
    selection: SelectionType,
    sortString: t.string,
  }),
]);
export type ProjectSettings = t.TypeOf<typeof ProjectSettings>;

export const ProjectUrlSettings = t.partial({
  compare: t.boolean,
  page: t.number,
});

export const settingsPathForProject = (id: number): string => `experimentListingForProject${id}`;
export const defaultProjectSettings: Required<ProjectSettings> = {
  columns: defaultExperimentColumns,
  columnWidths: defaultColumnWidths,
  compare: false,
  filterset: JSON.stringify(INIT_FORMSET),
  heatmapOn: false,
  heatmapSkipped: [],
  pageLimit: 20,
  pinnedColumnsCount: 3,
  selection: DEFAULT_SELECTION,
  sortString: 'id=desc',
};

export interface F_ExperimentListGlobalSettings {
  rowHeight: RowHeight;
  tableViewMode: TableViewMode;
}

export const experimentListGlobalSettingsConfig = t.partial({
  rowHeight: ioRowHeight,
  tableViewMode: ioTableViewMode,
});

export const experimentListGlobalSettingsDefaults = {
  rowHeight: RowHeight.MEDIUM,
  tableViewMode: 'scroll',
} as const;

export const experimentListGlobalSettingsPath = 'globalTableSettings';

export const settingsConfigGlobal: SettingsConfig<F_ExperimentListGlobalSettings> = {
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
  storagePath: experimentListGlobalSettingsPath,
};
