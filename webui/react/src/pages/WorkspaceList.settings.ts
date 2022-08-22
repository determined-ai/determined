import { GridListView } from 'components/GridListRadioGroup';
import { InteractiveTableSettings } from 'components/InteractiveTable';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetWorkspacesRequestSortBy } from 'services/api-ts-sdk';

export type WorkspaceColumnName =
  | 'action'
  | 'archived'
  | 'name'
  | 'numProjects'
  | 'state'
  | 'userId';

export const DEFAULT_COLUMNS: WorkspaceColumnName[] = [
  'name',
  'numProjects',
  'userId',
];

export enum WhoseWorkspaces {
  All = 'ALL_WORKSPACES',
  Mine = 'MY_WORKSPACES',
  Others = 'OTHERS_WORKSPACES'
}

export const DEFAULT_COLUMN_WIDTHS: Record<WorkspaceColumnName, number> = {
  action: 46,
  archived: 75,
  name: 150,
  numProjects: 74,
  state: 74,
  userId: 85,
};

export interface WorkspaceListSettings extends InteractiveTableSettings {
  archived?: boolean;
  columns: WorkspaceColumnName[];
  name?: string;
  sortKey: V1GetWorkspacesRequestSortBy;
  user?: string[];
  view: GridListView;
  whose: WhoseWorkspaces;
}

const config: SettingsConfig = {
  settings: [
    {
      defaultValue: false,
      key: 'archived',
      storageKey: 'archived',
      type: { baseType: BaseType.Boolean },
    },
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
      defaultValue: DEFAULT_COLUMNS.map((col: WorkspaceColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
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
      defaultValue: false,
      key: 'sortDesc',
      storageKey: 'sortDesc',
      type: { baseType: BaseType.Boolean },
    },
    {
      defaultValue: V1GetWorkspacesRequestSortBy.NAME,
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
    {
      key: 'user',
      storageKey: 'user',
      type: {
        baseType: BaseType.String,
        isArray: true,
      },
    },
    {
      defaultValue: GridListView.Grid,
      key: 'view',
      skipUrlEncoding: true,
      storageKey: 'view',
      type: { baseType: BaseType.String },
    },
    {
      defaultValue: WhoseWorkspaces.All,
      key: 'whose',
      skipUrlEncoding: true,
      storageKey: 'whose',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'workspace-list',
};

export default config;
