import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { ValueOf } from 'shared/types';

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
  sortKey: WorkspaceMembersSortBy;
}

const config: SettingsConfig = {
  applicableRoutespace: 'members',
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
      defaultValue: DEFAULT_COLUMNS.map((col: MembersColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
      key: 'columnWidths',
      skipUrlEncoding: true,
      storageKey: 'columnWidths',
      type: {
        baseType: BaseType.Float,
        isArray: true,
      },
    },
    {
      key: 'name',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: true,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: WorkspaceMembersSortBy.USERNAME,
      key: 'sortKey',
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: 10,
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
  storagePath: 'workspace-members',
};

export default config;
