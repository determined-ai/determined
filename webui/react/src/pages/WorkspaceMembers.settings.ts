import { InteractiveTableSettings } from 'components/InteractiveTable';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';

export type MembersColumnName = | 'name' | 'role' | 'action';


export const DEFAULT_COLUMNS: MembersColumnName[] = 
['name', 'role']

export const DEFAULT_COLUMN_WIDTHS: Record<MembersColumnName, number> = {
  name: 150,
  role: 20,
  action: 100,
}

export interface WorkspaceMembersSettings extends InteractiveTableSettings {
  columns: MembersColumnName[];
  name?: string;
  sortKey: V1GetWorkspaceProjectsRequestSortBy;
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
      defaultValue: V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME,
      key: 'sortKey',
      storageKey: 'sortKey',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: 100,
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
