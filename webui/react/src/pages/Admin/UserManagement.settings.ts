import * as t from 'io-ts';

import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { V1GetUsersRequestSortBy } from 'services/api-ts-sdk';
import { ValueOf } from 'types';

export type UserColumnName =
  | 'action'
  | 'displayName'
  | 'isActive'
  | 'isAdmin'
  | 'modifiedAt'
  | 'lastAuthAt';

export const DEFAULT_COLUMNS: UserColumnName[] = [
  'displayName',
  'isActive',
  'isAdmin',
  'modifiedAt',
  'lastAuthAt',
];

export const DEFAULT_COLUMN_WIDTHS: Record<UserColumnName, number> = {
  action: 20,
  displayName: 60,
  isActive: 40,
  isAdmin: 40,
  lastAuthAt: 80,
  modifiedAt: 80,
};

export const UserStatus = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
} as const;
export type UserStatus = ValueOf<typeof UserStatus>;

export const UserRole = {
  ADMIN: 'admin',
  MEMBER: 'member',
} as const;
export type UserRole = ValueOf<typeof UserRole>;

export const UserManagementSettings = t.intersection([
  t.type({
    columns: t.array(t.string),
    columnWidths: t.array(t.number),
    sortDesc: t.boolean,
    tableLimit: t.number,
    tableOffset: t.number,
  }),
  t.partial({
    name: t.string,
    roleFilter: t.keyof({
      [UserRole.ADMIN]: null,
      [UserRole.MEMBER]: null,
    }),
    row: t.union([t.array(t.number), t.array(t.string)]),
    sortKey: t.keyof({
      [V1GetUsersRequestSortBy.ACTIVE]: null,
      [V1GetUsersRequestSortBy.ADMIN]: null,
      [V1GetUsersRequestSortBy.DISPLAYNAME]: null,
      [V1GetUsersRequestSortBy.MODIFIEDTIME]: null,
      [V1GetUsersRequestSortBy.UNSPECIFIED]: null,
      [V1GetUsersRequestSortBy.USERNAME]: null,
      [V1GetUsersRequestSortBy.NAME]: null,
      [V1GetUsersRequestSortBy.LASTAUTHTIME]: null,
    }),
    statusFilter: t.keyof({
      [UserStatus.ACTIVE]: null,
      [UserStatus.INACTIVE]: null,
    }),
  }),
]);
export type UserManagementSettings = t.TypeOf<typeof UserManagementSettings>;
export const DEFAULT_SETTINGS: UserManagementSettings = {
  columns: DEFAULT_COLUMNS,
  columnWidths: DEFAULT_COLUMNS.map((col) => DEFAULT_COLUMN_WIDTHS[col]),
  sortDesc: true,
  sortKey: V1GetUsersRequestSortBy.MODIFIEDTIME,
  tableLimit: MINIMUM_PAGE_SIZE,
  tableOffset: 0,
};
