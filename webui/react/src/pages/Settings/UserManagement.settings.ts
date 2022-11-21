import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetUsersRequestSortBy } from 'services/api-ts-sdk/models';

export type UserColumnName =
  | 'action'
  | 'displayName'
  | 'username'
  | 'isActive'
  | 'isAdmin'
  | 'modifiedAt';

export const DEFAULT_COLUMNS: UserColumnName[] = [
  'displayName',
  'username',
  'isActive',
  'isAdmin',
  'modifiedAt',
];

export const DEFAULT_COLUMN_WIDTHS: Record<UserColumnName, number> = {
  action: 20,
  displayName: 80,
  isActive: 40,
  isAdmin: 40,
  modifiedAt: 80,
  username: 120,
};

export interface UserManagementSettings extends InteractiveTableSettings {
  sortDesc: boolean;
  sortKey: V1GetUsersRequestSortBy;
}

const config: SettingsConfig = {
  settings: [
    {
      defaultValue: DEFAULT_COLUMNS,
      key: 'columns',
      storageKey: 'columns',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      defaultValue: DEFAULT_COLUMNS.map((col: UserColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
        isArray: true,
      },
    },
    {
      key: 'row',
      type: { baseType: BaseType.Integer, isArray: true },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetUsersRequestSortBy.MODIFIED_TIME,
      key: 'sortKey',
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: MINIMUM_PAGE_SIZE,
      key: 'tableLimit',
      storageKey: 'tableLimit',
      type: { baseType: BaseType.Integer },
    },
    {
      defaultValue: 0,
      key: 'tableOffset',
      type: { baseType: BaseType.Integer },
    },
  ],
  storagePath: 'user-management',
};

export default config;
