import * as t from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';

import { ioRowHeight, ioTableViewMode, RowHeight, TableViewMode } from './OptionsMenu';

export const rowHeightMap: Record<RowHeight, number> = {
  [RowHeight.EXTRA_TALL]: 44,
  [RowHeight.TALL]: 40,
  [RowHeight.MEDIUM]: 36,
  [RowHeight.SHORT]: 32,
};

export interface DataGridGlobalSettings {
  rowHeight: RowHeight;
  tableViewMode: TableViewMode;
}

export const dataGridGlobalSettingsConfig = t.intersection([
  t.type({}),
  t.partial({
    rowHeight: ioRowHeight,
    tableViewMode: ioTableViewMode,
  }),
]);

export const dataGridGlobalSettingsDefaults = {
  rowHeight: RowHeight.MEDIUM,
  tableViewMode: 'scroll',
} as const;

export const dataGridGlobalSettingsPath = 'globalTableSettings';

export const settingsConfigGlobal: SettingsConfig<DataGridGlobalSettings> = {
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
  storagePath: dataGridGlobalSettingsPath,
};
