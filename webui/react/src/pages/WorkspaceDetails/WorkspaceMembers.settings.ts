import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { SettingsConfig } from 'hooks/useSettings';
import { ValueOf } from 'types';

export type MembersColumnName = 'name' | 'role' | 'action';

const WorkspaceMembersSortBy = {
  USERNAME: 'username',
} as const;

type WorkspaceMembersSortBy = ValueOf<typeof WorkspaceMembersSortBy>;

export const DEFAULT_COLUMNS: MembersColumnName[] = ['name', 'role'];

export const DEFAULT_COLUMN_WIDTHS: Record<MembersColumnName, number> = {
  action: 50,
  name: 100,
  role: 75,
};

export interface WorkspaceMembersSettings extends InteractiveTableSettings {
  columns: MembersColumnName[];
  name?: string;
  sortKey: string;
}

export const configForWorkspace = (id: number): SettingsConfig<WorkspaceMembersSettings> => ({
  settings: {
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(union([literal('name'), literal('role'), literal('action')])),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: MembersColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: array(number),
    },
    name: {
      defaultValue: undefined,
      storageKey: 'name',
      type: union([undefinedType, string]),
    },
    sortDesc: {
      defaultValue: true,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: WorkspaceMembersSortBy.USERNAME,
      storageKey: 'sortKey',
      type: string,
    },
    tableLimit: {
      defaultValue: 10,
      storageKey: 'tableLimit',
      type: number,
    },
    tableOffset: {
      defaultValue: 0,
      storageKey: 'tableOffset',
      type: number,
    },
  },
  storagePath: `workspace-${id}-members`,
});
