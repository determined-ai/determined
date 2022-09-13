import { GridListView } from 'components/GridListRadioGroup';
import { InteractiveTableSettings } from 'components/InteractiveTable';
import { BaseType, SettingsConfig } from 'hooks/useSettings';
import { V1GetWorkspaceProjectsRequestSortBy } from 'services/api-ts-sdk';

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

export enum WhoseProjects {
  All = 'ALL_PROJECTS',
  Mine = 'MY_PROJECTS',
  Others = 'OTHERS_PROJECTS'
}

export interface WorkspaceDetailsSettings extends InteractiveTableSettings {
  archived?: boolean;
  columns: ProjectColumnName[];
  name?: string;
  sortKey: V1GetWorkspaceProjectsRequestSortBy;
  user?: string[];
  view: GridListView;
  whose: WhoseProjects;
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
      defaultValue: DEFAULT_COLUMNS.map((col: ProjectColumnName) => DEFAULT_COLUMN_WIDTHS[col]),
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
      defaultValue: WhoseProjects.All,
      key: 'whose',
      skipUrlEncoding: true,
      storageKey: 'whose',
      type: { baseType: BaseType.String },
    },
  ],
  storagePath: 'workspace-details',
};

export default config;
