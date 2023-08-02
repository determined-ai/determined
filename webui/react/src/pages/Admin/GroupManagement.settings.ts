import { array, boolean, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';

export type UserColumnName = 'id' | 'name' | 'users' | 'action';

export const DEFAULT_COLUMNS: UserColumnName[] = ['id', 'name', 'users'];

export const DEFAULT_COLUMN_WIDTHS: Record<UserColumnName, number> = {
  action: 20,
  id: 20,
  name: 40,
  users: 40,
};

export type GroupManagementSettings = InteractiveTableSettings;

const config: SettingsConfig<InteractiveTableSettings> = {
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      storageKey: 'columns',
      type: array(string),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: UserColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    row: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'row',
      type: union([undefinedType, union([array(string), array(number)])]),
    },
    sortDesc: {
      defaultValue: false,
      storageKey: 'sortDesc',
      type: boolean,
    },
    tableLimit: {
      defaultValue: MINIMUM_PAGE_SIZE,
      storageKey: 'tableLimit',
      type: number,
    },
    tableOffset: {
      defaultValue: 0,
      storageKey: 'tableOffset',
      type: number,
    },
  },
  storagePath: 'group-management',
};

export default config;
