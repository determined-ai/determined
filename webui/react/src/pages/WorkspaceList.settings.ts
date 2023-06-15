import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { GridListView } from 'components/GridListRadioGroup';
import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { SettingsConfig } from 'hooks/useSettings';
import { V1GetWorkspacesRequestSortBy } from 'services/api-ts-sdk';
import { ValueOf } from 'types';

export type WorkspaceColumnName =
  | 'action'
  | 'archived'
  | 'name'
  | 'numProjects'
  | 'state'
  | 'userId';

export const DEFAULT_COLUMNS: WorkspaceColumnName[] = ['name', 'numProjects', 'userId'];

export const WhoseWorkspaces = {
  All: 'ALL_WORKSPACES',
  Mine: 'MY_WORKSPACES',
  Others: 'OTHERS_WORKSPACES',
} as const;

export type WhoseWorkspaces = ValueOf<typeof WhoseWorkspaces>;

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

const config: SettingsConfig<WorkspaceListSettings> = {
  settings: {
    archived: {
      defaultValue: false,
      storageKey: 'archived',
      type: union([undefinedType, boolean]),
    },
    columns: {
      defaultValue: DEFAULT_COLUMNS,
      skipUrlEncoding: true,
      storageKey: 'columns',
      type: array(
        union([
          literal('action'),
          literal('archived'),
          literal('name'),
          literal('numProjects'),
          literal('state'),
          literal('userId'),
        ]),
      ),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: WorkspaceColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
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
      defaultValue: false,
      storageKey: 'sortDesc',
      type: boolean,
    },
    sortKey: {
      defaultValue: V1GetWorkspacesRequestSortBy.NAME,
      storageKey: 'sortKey',
      type: union([
        literal(V1GetWorkspacesRequestSortBy.ID),
        literal(V1GetWorkspacesRequestSortBy.NAME),
        literal(V1GetWorkspacesRequestSortBy.UNSPECIFIED),
      ]),
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
    user: {
      defaultValue: undefined,
      storageKey: 'user',
      type: union([undefinedType, array(string)]),
    },
    view: {
      defaultValue: GridListView.Grid,
      skipUrlEncoding: true,
      storageKey: 'view',
      type: union([literal(GridListView.Grid), literal(GridListView.List)]),
    },
    whose: {
      defaultValue: WhoseWorkspaces.All,
      skipUrlEncoding: true,
      storageKey: 'whose',
      type: union([
        literal(WhoseWorkspaces.All),
        literal(WhoseWorkspaces.Mine),
        literal(WhoseWorkspaces.Others),
      ]),
    },
  },
  storagePath: 'workspace-list',
};

export default config;
