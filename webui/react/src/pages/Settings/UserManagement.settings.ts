import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { SettingsConfig } from 'hooks/useSettings';
import { V1GetUsersRequestSortBy } from 'services/api-ts-sdk';

export type UserColumnName = 'action' | 'displayName' | 'isActive' | 'isAdmin' | 'modifiedAt';

export const DEFAULT_COLUMNS: UserColumnName[] = [
  'displayName',
  'isActive',
  'isAdmin',
  'modifiedAt',
];

export const DEFAULT_COLUMN_WIDTHS: Record<UserColumnName, number> = {
  action: 20,
  displayName: 60,
  isActive: 40,
  isAdmin: 40,
  modifiedAt: 80,
};

export interface UserManagementSettings extends InteractiveTableSettings {
  name?: string;
  sortDesc: boolean;
  sortKey: V1GetUsersRequestSortBy;
}

const config: SettingsConfig<UserManagementSettings> = {
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(string),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: UserColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    name: {
      defaultValue: undefined,
      storageKey: 'name',
      type: union([string, undefinedType]),
    },
    row: {
      defaultValue: undefined,
      skipUrlEncoding: true,
      storageKey: 'row',
      type: union([array(number), array(string), undefinedType]),
    },
    sortDesc: {
      defaultValue: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: V1GetUsersRequestSortBy.MODIFIEDTIME,
      storageKey: 'sortKey',
      type: union([
        literal(V1GetUsersRequestSortBy.ACTIVE),
        literal(V1GetUsersRequestSortBy.ADMIN),
        literal(V1GetUsersRequestSortBy.DISPLAYNAME),
        literal(V1GetUsersRequestSortBy.MODIFIEDTIME),
        literal(V1GetUsersRequestSortBy.UNSPECIFIED),
        literal(V1GetUsersRequestSortBy.USERNAME),
        literal(V1GetUsersRequestSortBy.NAME),
      ]),
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
  storagePath: 'user-management',
};

export default config;
