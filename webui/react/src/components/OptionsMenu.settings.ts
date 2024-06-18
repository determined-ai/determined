import * as t from 'io-ts';

import { SettingsConfig } from 'hooks/useSettings';

import { ioRowHeight, RowHeight } from './OptionsMenu';

export const rowHeightMap: Record<RowHeight, number> = {
  [RowHeight.EXTRA_TALL]: 44,
  [RowHeight.TALL]: 40,
  [RowHeight.MEDIUM]: 36,
  [RowHeight.SHORT]: 32,
};

export interface DataGridGlobalSettings {
  rowHeight: RowHeight;
}

export const dataGridGlobalSettingsConfig = t.intersection([
  t.type({}),
  t.partial({
    rowHeight: ioRowHeight,
  }),
]);

export const dataGridGlobalSettingsDefaults = {
  rowHeight: RowHeight.MEDIUM,
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
  },
  storagePath: dataGridGlobalSettingsPath,
};
