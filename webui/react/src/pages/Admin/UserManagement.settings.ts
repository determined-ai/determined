import * as t from 'io-ts';

import { MINIMUM_PAGE_SIZE } from 'components/Table/Table';
import { V1GetUsersRequestSortBy } from 'services/api-ts-sdk';
import { ValueOf } from 'types';

export const DEFAULT_COLUMN_WIDTHS = {
  action: 20,
  displayName: 120,
  isActive: 40,
  isAdmin: 40,
  lastAuthAt: 50,
  modifiedAt: 50,
  remote: 30,
} as const satisfies Record<string, number>;

export type UserColumnName = keyof typeof DEFAULT_COLUMN_WIDTHS;

export const DEFAULT_COLUMNS: UserColumnName[] = [
  'displayName',
  'isActive',
  'lastAuthAt',
  'isAdmin',
  'remote',
  'modifiedAt',
];

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
    sortDesc: t.boolean,
    tableLimit: t.number,
    tableOffset: t.number,
  }),
  t.partial({
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
      [V1GetUsersRequestSortBy.REMOTE]: null,
    }),
  }),
]);
export type UserManagementSettings = t.TypeOf<typeof UserManagementSettings>;
export const DEFAULT_SETTINGS: UserManagementSettings = {
  sortDesc: true,
  sortKey: V1GetUsersRequestSortBy.MODIFIEDTIME,
  tableLimit: MINIMUM_PAGE_SIZE,
  tableOffset: 0,
};
