import { array, boolean, literal, number, string, undefined as undefinedType, union } from 'io-ts';

import { GridListView } from 'components/GridListRadioGroup';
import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import { SettingsConfig } from 'hooks/useSettings';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';
import { ValueOf } from 'types';

export type ProjectColumnName =
  | 'action'
  | 'archived'
  | 'description'
  | 'lastUpdated'
  | 'name'
  | 'numExperiments'
  | 'state'
  | 'userId';

export const DEFAULT_COLUMNS: ProjectColumnName[] = [
  'name',
  'description',
  'numExperiments',
  'state',
  'lastUpdated',
  'userId',
];

export const DEFAULT_COLUMN_WIDTHS: Record<ProjectColumnName, number> = {
  action: 46,
  archived: 75,
  description: 147,
  lastUpdated: 120,
  name: 150,
  numExperiments: 74,
  state: 74,
  userId: 85,
};

export const WhoseProjects = {
  All: 'ALL_PROJECTS',
  Mine: 'MY_PROJECTS',
  Others: 'OTHERS_PROJECTS',
} as const;

export type WhoseProjects = ValueOf<typeof WhoseProjects>;

export interface WorkspaceDetailsSettings extends InteractiveTableSettings {
  archived?: boolean;
  columns: ProjectColumnName[];
  name?: string;
  sortKey: V1GetWorkspaceProjectsRequestSortBy;
  user?: string[];
  view: GridListView;
  whose: WhoseProjects;
}

export const configForWorkspace = (id: number): SettingsConfig<WorkspaceDetailsSettings> => ({
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
          literal('description'),
          literal('lastUpdated'),
          literal('name'),
          literal('numExperiments'),
          literal('state'),
          literal('userId'),
        ]),
      ),
    },
    columnWidths: {
      defaultValue: DEFAULT_COLUMNS.map((col: ProjectColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
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
      defaultValue: V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME,
      storageKey: 'sortKey',
      type: union([
        literal(V1GetWorkspaceProjectsRequestSortBy.CREATIONTIME),
        literal(V1GetWorkspaceProjectsRequestSortBy.DESCRIPTION),
        literal(V1GetWorkspaceProjectsRequestSortBy.LASTEXPERIMENTSTARTTIME),
        literal(V1GetWorkspaceProjectsRequestSortBy.ID),
        literal(V1GetWorkspaceProjectsRequestSortBy.NAME),
        literal(V1GetWorkspaceProjectsRequestSortBy.UNSPECIFIED),
      ]),
    },
    tableLimit: {
      defaultValue: 100,
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
      defaultValue: WhoseProjects.All,
      skipUrlEncoding: true,
      storageKey: 'whose',
      type: union([
        literal(WhoseProjects.All),
        literal(WhoseProjects.Mine),
        literal(WhoseProjects.Others),
      ]),
    },
  },
  storagePath: `workspace-${id}-details`,
});
